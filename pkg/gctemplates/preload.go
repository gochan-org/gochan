package gctemplates

import (
	"html/template"
	"path"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()

	l.SetFuncs(t, map[string]lua.LGFunction{
		"load_template": func(l *lua.LState) int {
			var tmplPaths []string
			for i := 0; i < l.GetTop(); i++ {
				tmplPaths = append(tmplPaths, l.CheckString(i+1))
			}
			tmpl, err := template.New(path.Base(tmplPaths[0])).Funcs(funcMap).ParseFiles(tmplPaths...)
			l.Push(luar.New(l, tmpl))
			l.Push(luar.New(l, err))
			return 2
		},
		"parse_template": func(l *lua.LState) int {
			tmplName := l.CheckString(1)
			tmplData := l.CheckString(2)
			tmpl, err := ParseTemplate(tmplName, tmplData)
			l.Push(luar.New(l, tmpl))
			l.Push(luar.New(l, err))
			return 2
		},
	})

	l.Push(t)
	return 1

}
