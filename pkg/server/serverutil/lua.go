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
			tmplLV := l.CheckAny(1)
			var tmpl *template.Template
			var tmplStr string
			switch tmplLV.Type() {
			case lua.LTUserData:
				tmpl = tmplLV.(*lua.LUserData).Value.(*template.Template)
			case lua.LTString:
				tmplStr = tmplLV.String()
			default:
				l.ArgError(1, "expected string or template")
				return 0
			}
			data := luautil.LValueToInterface(l, l.CheckAny(2))
			writer := l.CheckUserData(3).Value.(io.Writer)
			mediaType := l.CheckString(4)
			var err error
			switch tmplLV.Type() {
			case lua.LTString:
				err = MinifyTemplate(tmplStr, data, writer, mediaType)
			case lua.LTUserData:
				err = MinifyTemplate(tmpl, data, writer, mediaType)
			}
			l.Push(luar.New(l, err))
			return 1
		},
	})
	l.Push(t)
	return 1
}
