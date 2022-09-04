package gcplugin

import (
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"

	lua "github.com/yuin/gopher-lua"
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

func createLuaLogFunc(which string) lua.LGFunction {
	return func(l *lua.LState) int {
		args := []interface{}{}
		for v := 1; v <= l.GetTop(); v++ {
			args = append(args, l.Get(v))
		}
		switch which {
		case "access":
			gcutil.LogInfo().
				Interface("pluginAccess", args)
		case "error":
			gcutil.LogError(nil).
				Interface("pluginError", args)
		case "staff":
			gcutil.LogInfo().
				Interface("pluginStaff", args)
		}
		return 0
	}
}

func registerLuaFunctions() {
	lState.Register("access_log", createLuaLogFunc("access"))
	lState.Register("error_log", createLuaLogFunc("error"))
	lState.Register("staff_log", createLuaLogFunc("staff"))
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
		pluginTable := lState.CheckTable(-1)
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
