package gcplugin

import (
	"errors"
	"fmt"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcutil"

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

func lvalueToInterface(v lua.LValue) interface{} {
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
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, args...)
	}
}

func registerLuaFunctions() {
	lState.Register("info_log", createLuaLogFunc("info"))
	lState.Register("warn_log", createLuaLogFunc("warn"))
	lState.Register("error_log", createLuaLogFunc("error"))
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
			data = append(data, lvalueToInterface(v))
		}
		fmt.Println("triggering", trigger)
		events.TriggerEvent(trigger, data...)
		return 0
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
