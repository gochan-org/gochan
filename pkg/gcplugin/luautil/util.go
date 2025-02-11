package luautil

import (
	lua "github.com/yuin/gopher-lua"
)

func LValueToInterface(l *lua.LState, v lua.LValue) any {
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
	case lua.LTTable:
		t := v.(*lua.LTable)
		tableLength := t.Len()
		if tableLength > 0 {
			// Array
			arr := make([]any, tableLength)
			for i := 1; i <= tableLength; i++ {
				arr[i-1] = LValueToInterface(l, t.RawGetInt(i))
			}
			return arr
		}
		m := make(map[string]any)
		t.ForEach(func(k, v lua.LValue) {
			m[k.String()] = LValueToInterface(l, v)
		})
		return m
	default:
		l.ArgError(2, "Incompatible Lua type")
	}
	return nil
}

func GetTableValueAliased(t *lua.LTable, keys ...string) (lua.LValue, string) {
	val := lua.LNil
	for _, key := range keys {
		val = t.RawGetString(key)
		if val != lua.LNil {
			return val, key
		}
	}
	return val, ""
}
