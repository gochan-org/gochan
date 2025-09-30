package manage

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
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

func luaHandlerOutputToGo(l *lua.LState) (any, error) {
	outV := l.Get(-2)
	errV := l.Get(-1)
	err := luautil.LValueToError(errV)
	if err != nil {
		return nil, err
	}
	switch outV.Type() {
	case lua.LTNil:
		return nil, nil
	case lua.LTString:
		return lua.LVAsString(outV), nil
	case lua.LTUserData:
		return outV.(*lua.LUserData).Value, nil
	case lua.LTTable:
		tableMap := make(map[string]any)
		outV.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
			tableMap[lua.LVAsString(key)] = value
		})
		return tableMap, nil
	default:
		return nil, fmt.Errorf("invalid output return type from handler: %s", outV.Type().String())
	}
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
			actionHandler := func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output any, err error) {
				if err = l.CallByParam(lua.P{
					Fn:   fn,
					NRet: 2,
					// Protect: true,
				}, luar.New(l, writer), luar.New(l, request), luar.New(l, staff), lua.LBool(wantsJSON), luar.New(l, infoEv), luar.New(l, errEv)); err != nil {
					return "", err
				}
				return luaHandlerOutputToGo(l)
			}
			RegisterManagePage(actionID, actionTitle, actionPerms, actionJSON, actionHandler)
			return 0
		},
		"register_staff_action": func(l *lua.LState) int {
			actionTable := l.CheckTable(1)
			var action Action
			id, _ := luautil.GetTableValueAliased(actionTable, "id", "ID")
			if id.Type() != lua.LTString || id.String() == "" {
				l.ArgError(1, "missing or invalid id field")
				return 0
			}
			action.ID = id.String()

			title, _ := luautil.GetTableValueAliased(actionTable, "title", "Title")
			if title.Type() != lua.LTString || title.String() == "" {
				l.ArgError(1, "missing or invalid title field")
				return 0
			}
			action.Title = title.String()

			perms, _ := luautil.GetTableValueAliased(actionTable, "permissions", "perms", "Permissions", "Perms")
			switch perms.Type() {
			case lua.LTNumber:
				action.Permissions = int(lua.LVAsNumber(perms))
			case lua.LTNil:
				action.Permissions = NoPerms
			case lua.LTString:
				switch perms.String() {
				case "no_perms", "no_permission", "no_permissions", "none", "public":
					action.Permissions = NoPerms
				case "janitor_perms", "janitor_permission", "janitor_permissions", "janitor":
					action.Permissions = JanitorPerms
				case "mod_perms", "mod_permission", "mod_permissions", "mod", "moderator":
					action.Permissions = ModPerms
				case "admin_perms", "admin_permission", "admin_permissions", "admin", "administrator":
					action.Permissions = AdminPerms
				default:
					l.ArgError(1, "invalid permissions field")
					return 0
				}
			default:
				l.ArgError(1, "invalid permissions field, expected number or string")
				return 0
			}
			hidden, _ := luautil.GetTableValueAliased(actionTable, "hidden", "Hidden")
			action.Hidden = lua.LVAsBool(hidden)

			jsonOut, _ := luautil.GetTableValueAliased(actionTable, "json_output", "jsonOutput", "JSONoutput", "json", "JSON")
			switch jsonOut.Type() {
			case lua.LTNumber:
				action.JSONoutput = int(lua.LVAsNumber(jsonOut))
			case lua.LTBool:
				if lua.LVAsBool(jsonOut) {
					action.JSONoutput = AlwaysJSON
				} else {
					action.JSONoutput = NoJSON
				}
			case lua.LTNil:
				action.JSONoutput = NoJSON
			case lua.LTString:
				switch jsonOut.String() {
				case "no_json", "nojson", "html":
					action.JSONoutput = NoJSON
				case "optional_json", "optionaljson", "optional", "sometimes":
					action.JSONoutput = OptionalJSON
				case "always_json", "alwaysjson", "json", "JSON":
					action.JSONoutput = AlwaysJSON
				default:
					l.ArgError(1, "invalid json_output field")
					return 0
				}
			default:
				l.ArgError(1, "invalid json_output field, expected number or string")
				return 0
			}

			fn, _ := luautil.GetTableValueAliased(actionTable, "callback", "Callback", "handler", "Handler", "function", "Function", "run")
			if fn.Type() != lua.LTFunction {
				l.ArgError(1, "missing or invalid callback field")
				return 0
			}
			action.Callback = func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output any, err error) {
				if err = l.CallByParam(lua.P{
					Fn:   fn,
					NRet: 2,
					// Protect: true,
				}, luar.New(l, writer), luar.New(l, request), luar.New(l, staff), lua.LBool(wantsJSON), luar.New(l, infoEv), luar.New(l, errEv)); err != nil {
					return "", err
				}
				return luaHandlerOutputToGo(l)
			}

			methodsVal := l.Get(2)
			var methods []string
			switch methodsVal.Type() {
			case lua.LTNil:
				methods = []string{"GET", "POST"}
			case lua.LTTable:
				methodsVal.(*lua.LTable).ForEach(func(_ lua.LValue, value lua.LValue) {
					if value.Type() != lua.LTString {
						l.ArgError(1, "invalid methods field, expected table of strings")
						return
					}
					methods = append(methods, value.String())
				})
			case lua.LTString:
				methods = []string{methodsVal.String()}
			default:
				l.ArgError(1, "invalid methods field, expected table, string, or nil")
				return 0
			}
			RegisterStaffAction(action, methods...)
			return 0
		},
		"get_action_request_params": func(l *lua.LState) int {
			reqV := l.CheckUserData(1)
			if reqV.Type() != lua.LTUserData {
				l.ArgError(1, "expected http.Request")
				return 0
			}
			req, ok := reqV.Value.(*http.Request)
			if !ok {
				l.ArgError(1, "expected http.Request")
				return 0
			}
			params := req.Context().Value(requestContextKey{}).(bunrouter.Params)
			l.Push(luar.New(l, params))
			return 1
		},
	})

	l.Push(t)
	return 1
}
