package luautil

import (
	"errors"

	lua "github.com/yuin/gopher-lua"
)

var (
	ErrInvalidErrorType = errors.New("returned invalid error value (expected string, nil, table, or error object)")
)

// LValueToInterface converts a lua.LValue to a Go interface{}.
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

// LValueToError converts a lua.LValue to an error. It supports string, userdata (wrapping an error),
// and table (with "message" and optional status fields)
func LValueToError(errV lua.LValue) error {
	switch errV.Type() {
	case lua.LTNil:
		return nil
	case lua.LTString:
		errStr := lua.LVAsString(errV)
		if errStr == "" {
			return nil
		}
		return errors.New(errStr)
	case lua.LTUserData:
		errUD := errV.(*lua.LUserData)
		errGo, ok := errUD.Value.(error)
		if !ok {
			return ErrInvalidErrorType
		}
		return errGo
	case lua.LTTable:
		errTable := errV.(*lua.LTable)
		msgV, _ := GetTableValueAliased(errTable, "message", "error", "err")
		// statusV, key := GetTableValueAliased(errTable, "status", "code", "status_code", "http_status")
		// if key != "" {
		// 	return server.NewServerError(lua.LVAsString(msgV), int(lua.LVAsNumber(statusV)))
		// }
		return errors.New(lua.LVAsString(msgV))
	}
	return ErrInvalidErrorType
}

// GetTableValueAliased retrieves a value from a Lua table using multiple possible keys.
// It returns the first value and the key that was found. If none of the keys are found, it returns lua.LNil and an empty string.
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
