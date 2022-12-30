package gcplugin

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/config"
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

func registerLuaFunctions() {
	lState.Register("info_log", createLuaLogFunc("info"))
	lState.Register("warn_log", createLuaLogFunc("warn"))
	lState.Register("error_log", createLuaLogFunc("error"))
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
