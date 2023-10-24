package gctemplates

import (
	"html/template"
	"io"

	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
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
			tmpl, err := LoadTemplate(tmplPaths...)
			l.Push(luar.New(l, tmpl))
			l.Push(luar.New(l, err))
			return 2
		},
		"minify_template": func(l *lua.LState) int {
			tmplUD := l.CheckUserData(1)
			tmpl := tmplUD.Value.(*template.Template)
			dataTable := l.CheckTable(2)
			data := map[string]interface{}{}
			dataTable.ForEach(func(l1, l2 lua.LValue) {
				data[l1.String()] = luautil.LValueToInterface(l, l2)
			})
			writer := l.CheckUserData(3).Value.(io.Writer)
			mediaType := l.CheckString(4)
			err := serverutil.MinifyTemplate(tmpl, data, writer, mediaType)
			l.Push(luar.New(l, err))
			return 1
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
