package gcutil

import (
	"fmt"

	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func createLuaLogFunc(which string) lua.LGFunction {
	return func(l *lua.LState) int {
		switch which {
		case "info":
			l.Push(luar.New(l, LogInfo()))
		case "warn":
			l.Push(luar.New(l, LogWarning()))
		case "error":
			numArgs := l.GetTop()
			if numArgs == 0 {
				l.Push(luar.New(l, LogError(nil)))
			} else {
				errVal := l.CheckAny(-1)
				errI := luautil.LValueToInterface(l, errVal)
				err := fmt.Errorf("%v", errI)

				l.Push(luar.New(l, LogError(err)))
			}
		}
		return 1
	}
}

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"info_log":  createLuaLogFunc("info"),
		"warn_log":  createLuaLogFunc("warn"),
		"error_log": createLuaLogFunc("error"),
	})
	l.Push(t)
	return 1
}
