package gcplugin

import (
	"database/sql"
	"errors"
	"html/template"
	"io"
	"net/http"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"

	luaFilePath "github.com/vadv/gopher-lua-libs/filepath"
	luaStrings "github.com/vadv/gopher-lua-libs/strings"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var (
	lState       *lua.LState
	eventPlugins map[string][]*lua.LFunction
)

func initLua() {
	if lState == nil {
		lState = lua.NewState()
		registerLuaFunctions()
	}
}

func ClosePlugins() {
	if lState != nil {
		lState.Close()
	}
}

func lvalueToInterface(l *lua.LState, v lua.LValue) interface{} {
	lt := v.Type()
	switch lt {
	case lua.LTNil:
		return nil
	case lua.LTBool:
		return lua.LVAsBool(v)
	case lua.LTNumber:
		return lua.LVAsNumber(v)
	case lua.LTString:
		return v.String()
	case lua.LTUserData:
		l.Push(v)
		return l.CheckUserData(l.GetTop()).Value
	default:
		gcutil.LogError(nil).Caller(1).
			Interface("lvalue", v).
			Str("type", lt.String()).
			Msg("Unrecognized or unsupported Lua type")
	}
	return nil
}

func createLuaLogFunc(which string) lua.LGFunction {
	return func(l *lua.LState) int {
		switch which {
		case "info":
			l.Push(luar.New(l, gcutil.LogInfo()))
		case "warn":
			l.Push(luar.New(l, gcutil.LogWarning()))
		case "error":
			numArgs := l.GetTop()
			if numArgs == 0 {
				l.Push(luar.New(l, gcutil.LogError(nil)))
			} else {
				l.Push(luar.New(l, gcutil.LogError(errors.New(l.CheckString(-1)))))
			}
		}
		return 1
	}
}

func luaEventRegisterHandlerAdapter(l *lua.LState, fn *lua.LFunction) events.EventHandler {
	return func(trigger string, data ...interface{}) {
		args := []lua.LValue{
			luar.New(l, trigger),
		}
		for _, i := range data {
			args = append(args, luar.New(l, i))
		}
		l.CallByParam(lua.P{
			Fn:   fn,
			NRet: 0,
			// Protect: true,
		}, args...)
	}
}

func registerLuaFunctions() {
	luaFilePath.Preload(lState)
	luaStrings.Preload(lState)
	lState.Register("info_log", createLuaLogFunc("info"))
	lState.Register("warn_log", createLuaLogFunc("warn"))
	lState.Register("error_log", createLuaLogFunc("error"))

	lState.Register("system_critical_config", func(l *lua.LState) int {
		l.Push(luar.New(l, config.GetSystemCriticalConfig()))
		return 1
	})
	lState.Register("site_config", func(l *lua.LState) int {
		l.Push(luar.New(l, config.GetSiteConfig()))
		return 1
	})
	lState.Register("board_config", func(l *lua.LState) int {
		l.Push(luar.New(l, config.GetBoardConfig(l.CheckString(1))))
		return 1
	})

	lState.Register("db_query", func(l *lua.LState) int {
		queryStr := l.CheckString(1)
		queryArgsL := l.CheckAny(2)

		var queryArgs []any
		if queryArgsL.Type() != lua.LTNil {
			table := queryArgsL.(*lua.LTable)
			table.ForEach(func(_ lua.LValue, val lua.LValue) {
				arg := lvalueToInterface(l, val)
				queryArgs = append(queryArgs, arg)
			})
		}

		rows, err := gcsql.QuerySQL(queryStr, queryArgs...)

		l.Push(luar.New(l, rows))
		l.Push(luar.New(l, err))
		return 2
	})

	lState.Register("db_scan_rows", func(l *lua.LState) int {
		rows := l.CheckUserData(1).Value.(*sql.Rows)
		table := l.CheckTable(2)
		var val any
		err := rows.Scan(&val)
		if err != nil {
			l.Push(luar.New(l, err))
			return 1
		}
		table.Append(luar.New(l, val))
		l.Push(lua.LNil)
		return 1
	})

	lState.Register("event_register", func(l *lua.LState) int {
		table := l.CheckTable(-2)
		var triggers []string
		table.ForEach(func(i, val lua.LValue) {
			triggers = append(triggers, val.String())
		})
		fn := l.CheckFunction(-1)
		events.RegisterEvent(triggers, luaEventRegisterHandlerAdapter(l, fn))
		return 0
	})
	lState.Register("event_trigger", func(l *lua.LState) int {
		trigger := l.CheckString(1)
		numArgs := l.GetTop()
		var data []interface{}
		for i := 2; i <= numArgs; i++ {
			v := l.CheckAny(i)
			data = append(data, lvalueToInterface(l, v))
		}
		events.TriggerEvent(trigger, data...)
		return 0
	})

	lState.Register("register_manage_page", func(l *lua.LState) int {
		actionID := l.CheckString(1)
		actionTitle := l.CheckString(2)
		actionPerms := l.CheckInt(3)
		actionJSON := l.CheckInt(4)
		fn := l.CheckFunction(5)
		actionHandler := func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			if err = l.CallByParam(lua.P{
				Fn:      fn,
				NRet:    2,
				Protect: true,
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
		manage.RegisterManagePage(actionID, actionTitle, actionPerms, actionJSON, actionHandler)
		return 0
	})
	lState.Register("load_template", func(l *lua.LState) int {
		var tmplPaths []string
		for i := 0; i < l.GetTop(); i++ {
			tmplPaths = append(tmplPaths, l.CheckString(i+1))
		}
		tmpl, err := gctemplates.LoadTemplate(tmplPaths...)
		l.Push(luar.New(l, tmpl))
		l.Push(luar.New(l, err))
		return 2
	})
	lState.Register("parse_template", func(l *lua.LState) int {
		tmplName := l.CheckString(1)
		tmplData := l.CheckString(2)
		tmpl, err := gctemplates.ParseTemplate(tmplName, tmplData)
		l.Push(luar.New(l, tmpl))
		l.Push(luar.New(l, err))
		return 2
	})
	lState.Register("minify_template", func(l *lua.LState) int {
		tmplUD := l.CheckUserData(1)
		tmpl := tmplUD.Value.(*template.Template)
		dataTable := l.CheckTable(2)
		data := map[string]interface{}{}
		dataTable.ForEach(func(l1, l2 lua.LValue) {
			data[l1.String()] = lvalueToInterface(l, l2)
		})
		writer := l.CheckUserData(3).Value.(io.Writer)
		mediaType := l.CheckString(4)
		err := serverutil.MinifyTemplate(tmpl, data, writer, mediaType)
		l.Push(luar.New(l, err))
		return 1
	})
	lState.SetGlobal("_GOCHAN_VERSION", lua.LString(config.GetVersion().String()))
}

func registerEventFunction(name string, fn *lua.LFunction) {
	switch name {
	case "onStartup":
		fallthrough
	case "onPost":
		fallthrough
	case "onDelete":
		eventPlugins[name] = append(eventPlugins[name], fn)
	}
}

func LoadPlugins(paths []string) error {
	var err error
	for _, pluginPath := range paths {
		initLua()
		if err = lState.DoFile(pluginPath); err != nil {
			return err
		}
		pluginTable := lState.NewTable()
		pluginTable.ForEach(func(key, val lua.LValue) {
			keyStr := key.String()
			fn, ok := val.(*lua.LFunction)
			if !ok {
				return
			}
			registerEventFunction(keyStr, fn)
		})
	}
	return nil
}
