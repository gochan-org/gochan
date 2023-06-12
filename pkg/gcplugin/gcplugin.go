package gcplugin

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"plugin"
	"reflect"

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
	lState             *lua.LState
	eventPlugins       map[string][]*lua.LFunction
	ErrInvalidInitFunc = errors.New("invalid InitPlugin, expected function with 0 arguments and 1 return value (error type)")
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

type lvalueScanner struct {
	val   lua.LValue
	state *lua.LState
}

func (lvs *lvalueScanner) Scan(src any) error {
	typeof := reflect.TypeOf(src)
	if typeof != nil && typeof.String() == "[]uint8" {
		src = string(src.([]uint8))
	}
	lvs.val = luar.New(lvs.state, src)
	return nil
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
		return lua.LVAsString(v)
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
	return func(trigger string, data ...interface{}) error {
		args := []lua.LValue{
			luar.New(l, trigger),
		}
		for _, i := range data {
			args = append(args, luar.New(l, i))
		}
		l.CallByParam(lua.P{
			Fn:   fn,
			NRet: 1,
			// Protect: true,
		}, args...)
		errStr := lua.LVAsString(l.Get(-1))
		if errStr != "" {
			return errors.New(errStr)
		}
		return nil
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
		var scanners []any
		colNames, err := rows.Columns()
		if err != nil {
			l.Push(luar.New(l, err))
			return 1
		}

		for range colNames {
			scanners = append(scanners, &lvalueScanner{state: l})
		}

		if err = rows.Scan(scanners...); err != nil {
			l.Push(luar.New(l, err))
			return 1
		}
		for i, name := range colNames {
			table.RawSetString(name, scanners[i].(*lvalueScanner).val)
		}
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
	var luaInitialized bool
	for _, pluginPath := range paths {
		ext := path.Ext(pluginPath)
		fmt.Println("Loading plugin", pluginPath)
		switch ext {
		case ".lua":
			if !luaInitialized {
				initLua()
				luaInitialized = true
			}
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
		case ".so":
			nativePlugin, err := plugin.Open(pluginPath)
			if err != nil {
				return err
			}
			initFuncSymbol, err := nativePlugin.Lookup("InitPlugin")
			if err != nil {
				return err
			}
			initFunc, ok := initFuncSymbol.(func() error)
			if !ok {
				return ErrInvalidInitFunc
			}
			if err = initFunc(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized plugin type (expected .lua or .so extension): %s", pluginPath)
		}
	}
	return nil
}
