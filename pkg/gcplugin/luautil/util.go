package luautil

import (
	lua "github.com/yuin/gopher-lua"
)

func LValueToInterface(l *lua.LState, v lua.LValue) interface{} {
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
		l.RaiseError("Incompatible Lua type")
	}
	return nil
}
