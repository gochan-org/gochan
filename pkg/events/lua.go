package events

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func luaEventRegisterHandlerAdapter(l *lua.LState, fn *lua.LFunction) EventHandler {
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

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"register_event": func(l *lua.LState) int {
			table := l.CheckTable(-2)
			var triggers []string
			table.ForEach(func(i, val lua.LValue) {
				triggers = append(triggers, val.String())
			})
			fn := l.CheckFunction(-1)
			RegisterEvent(triggers, luaEventRegisterHandlerAdapter(l, fn))
			return 0
		},
		"trigger_event": func(l *lua.LState) int {
			trigger := l.CheckString(1)
			numArgs := l.GetTop()
			var data []interface{}
			for i := 2; i <= numArgs; i++ {
				v := l.CheckAny(i)
				data = append(data, luautil.LValueToInterface(l, v))
			}
			TriggerEvent(trigger, data...)
			return 0
		},
	})
	l.Push(t)
	return 1
}
