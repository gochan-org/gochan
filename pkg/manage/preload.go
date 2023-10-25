package manage

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	tableArgFmt = "invalid value for key %q passed to table, expected %s, got %s"
)

func tableCheck(key string, val lua.LValue, ban *gcsql.IPBan) error {
	valType := val.Type()
	var err error
	switch key {
	case "board":
		fallthrough
	case "BoardID":
		fallthrough
	case "board_id":
		switch valType {
		case lua.LTNil:
			// global
		case lua.LTNumber:
			ban.BoardID = new(int)
			*ban.BoardID = int(lua.LVAsNumber(val))
		case lua.LTString:
			boardDir := lua.LVAsString(val)
			if boardDir != "" {
				var id int
				if id, err = gcsql.GetBoardIDFromDir(boardDir); err != nil {
					return err
				}
				ban.BoardID = new(int)
				*ban.BoardID = id
			}
		default:
			return fmt.Errorf(tableArgFmt, key, "string, number, or nil", valType)
		}
	case "post_id":
		fallthrough
	case "post":
		fallthrough
	case "PostID":
		if valType != lua.LTNumber {
			return fmt.Errorf(tableArgFmt, key, "number", valType)
		}
	case "is_thread_ban":
		fallthrough
	case "IsThreadBan":
		ban.IsThreadBan = lua.LVAsBool(val)
	case "appeal_after":
		fallthrough
	case "AppealAfter":
		str := lua.LVAsString(val)
		dur, err := durationutil.ParseLongerDuration(str)
		if err != nil {
			return err
		}
		ban.AppealAt = time.Now().Add(dur)
	case "can_appeal":
		fallthrough
	case "appealable":
		fallthrough
	case "CanAppeal":
		ban.CanAppeal = lua.LVAsBool(val)
	case "staff_note":
		fallthrough
	case "StaffNote":
		ban.StaffNote = lua.LVAsString(val)
	}
	return nil
}

func luaBanIP(l *lua.LState) int {
	now := time.Now()
	ban := &gcsql.IPBan{
		IP: l.CheckString(1),
	}
	ban.IsActive = true
	ban.AppealAt = now
	ban.CanAppeal = true

	durOrNil := l.CheckAny(2)
	var err error
	switch durOrNil.Type() {
	case lua.LTNil:
		ban.Permanent = true
	case lua.LTString:
		var duration time.Duration
		duration, err = durationutil.ParseLongerDuration(lua.LVAsString(durOrNil))
		if err != nil {
			l.Push(luar.New(l, err))
			return 1
		}
		ban.ExpiresAt = time.Now().Add(duration)
	default:
		l.ArgError(2, "Expected string or nil value")
	}

	ban.Message = l.CheckString(3)

	staff := l.CheckAny(4)
	switch staff.Type() {
	case lua.LTString:
		ban.StaffID, err = gcsql.GetStaffID(lua.LVAsString(staff))
		if err != nil {
			l.Push(luar.New(l, err))
			return 1
		}
	case lua.LTNumber:
		ban.StaffID = int(lua.LVAsNumber(staff))
	default:
		l.TypeError(4, staff.Type())
	}

	if l.GetTop() > 4 {
		t := l.CheckTable(5)
		var failed bool
		t.ForEach(func(keyLV, val lua.LValue) {
			key := lua.LVAsString(keyLV)
			if err = tableCheck(key, val, ban); err != nil {
				l.Push(luar.New(l, err))
			}
		})
		if failed {
			return 1
		}
	}
	if ban.StaffID < 1 {
		l.Push(luar.New(l, errors.New("missing staff key in table")))
		return 1
	}
	ban.IssuedAt = time.Now()
	err = gcsql.NewIPBan(ban)
	l.Push(luar.New(l, err))
	return 1
}

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"ban_ip": luaBanIP,
		"register_manage_page": func(l *lua.LState) int {
			actionID := l.CheckString(1)
			actionTitle := l.CheckString(2)
			actionPerms := l.CheckInt(3)
			actionJSON := l.CheckInt(4)
			fn := l.CheckFunction(5)
			actionHandler := func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				if err = l.CallByParam(lua.P{
					Fn:   fn,
					NRet: 2,
					// Protect: true,
				}, luar.New(l, writer), luar.New(l, request), luar.New(l, staff), lua.LBool(wantsJSON), luar.New(l, infoEv), luar.New(l, errEv)); err != nil {
					return "", err
				}
				out := lua.LVAsString(l.Get(-2))
				errStr := lua.LVAsString(l.Get(-1))
				if errStr != "" {
					err = errors.New(errStr)
				}
				return out, err
			}
			RegisterManagePage(actionID, actionTitle, actionPerms, actionJSON, actionHandler)
			return 0
		},
	})

	l.Push(t)
	return 1
}
