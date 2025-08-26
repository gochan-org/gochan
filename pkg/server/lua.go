package server

import (
	lua "github.com/yuin/gopher-lua"
)

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"register_ext_headers": func(l *lua.LState) int {
			ext := l.CheckString(1)
			header := l.CheckTable(2)

			var headerData StaticFileHeaders
			header.ForEach(func(kv, vv lua.LValue) {
				k := kv.String()
				v := vv.String()
				switch k {
				case "Content-Type":
					headerData.ContentType = v
				case "Cache-Control":
					headerData.CacheControl = v
				default:
					if headerData.Other == nil {
						headerData.Other = make(map[string]string)
					}
					headerData.Other[k] = v
				}
			})
			if headerData.ContentType == "" {
				l.Error(lua.LString("Content-Type key is missing for extension "+ext), 1)
				return 0
			}

			knownFileHeaders[ext] = headerData
			return 0
		},
	})
	l.Push(t)
	return 1
}
