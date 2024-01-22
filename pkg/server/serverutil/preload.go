package serverutil

import (
	"html/template"
	"io"

	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()

	l.SetFuncs(t, map[string]lua.LGFunction{
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
			err := MinifyTemplate(tmpl, data, writer, mediaType)
			l.Push(luar.New(l, err))
			return 1
		},
	})
	l.Push(t)
	return 1
}
