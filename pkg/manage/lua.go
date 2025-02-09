package manage

import (
	"errors"
	"net/http"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	tableArgFmt = "invalid value for key %q passed to table, expected %s, got %s"
)

func luaBanIP(l *lua.LState) int {
	ban := &gcsql.IPBan{}
	ip := l.CheckString(1)
	var err error
	ban.RangeStart, ban.RangeEnd, err = gcutil.ParseIPRange(ip)
	if err != nil {
		l.Push(luar.New(l, err))
		return 1
	}

	ban.IsActive = true
	ban.AppealAt = time.Now()
	ban.CanAppeal = true

	durOrNil := l.CheckAny(2)
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
		luautil.GetTableValueAliased(t)

		val, key := luautil.GetTableValueAliased(t, "board", "BoardID", "board_id")
		valType := val.Type()
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
					l.Push(luar.New(l, err))
					return 1
				}
				ban.BoardID = new(int)
				*ban.BoardID = id
			}
		default:
			l.RaiseError(tableArgFmt, key, "string, number, or nil", valType)
			return 0
		}
		val, key = luautil.GetTableValueAliased(t, "post", "PostID", "post_id")
		valType = val.Type()
		if valType == lua.LTNumber {
			ban.BannedForPostID = new(int)
			*ban.BannedForPostID = int(lua.LVAsNumber(val))
		} else if valType != lua.LTNil {
			l.RaiseError(tableArgFmt, key, "number", valType)
			return 0
		}

		val, _ = luautil.GetTableValueAliased(t, "is_thread_ban", "IsThreadBan")
		ban.IsThreadBan = lua.LVAsBool(val)

		durOrNil, _ = luautil.GetTableValueAliased(t, "appeal_after", "AppealAfter")
		if durOrNil != lua.LNil {
			str := lua.LVAsString(durOrNil)
			dur, err := durationutil.ParseLongerDuration(str)
			if err != nil {
				l.Push(luar.New(l, err))
				return 1
			}
			ban.AppealAt = time.Now().Add(dur)
			ban.CanAppeal = true
		}
		val, _ = luautil.GetTableValueAliased(t, "can_appeal", "appealable", "CanAppeal")
		ban.CanAppeal = lua.LVAsBool(val)
		val, _ = luautil.GetTableValueAliased(t, "staff_note", "StaffNote")
		ban.StaffNote = lua.LVAsString(val)
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
