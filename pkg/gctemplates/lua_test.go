package gctemplates

import (
	"bytes"
	"path"
	"testing"

	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func TestLuaTemplateFunctions(t *testing.T) {
	gochanRoot, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	AddTemplateFuncs(funcMap)

	buf := new(bytes.Buffer)
	l := lua.NewState()
	l.SetGlobal("buffer", luar.New(l, buf))
	l.SetGlobal("luatemplate_html", lua.LString(path.Join(gochanRoot, "pkg/gctemplates/testdata/luatemplate.html")))
	l.PreloadModule("gctemplates", PreloadModule)

	testCases := []struct {
		desc      string
		luaStr    string
		expected  string
		expectErr bool
	}{
		{
			desc:     "get test buffer",
			luaStr:   `buffer:WriteString("blah blah blah")`,
			expected: "blah blah blah",
		},
		{
			desc: "load template",
			luaStr: `local gctemplates = require("gctemplates");
local tmpl, err = gctemplates.load_template(luatemplate_html);
assert(err == nil);
assert(tmpl);`,
		},
		{
			desc: "load and execute template on buffer",
			luaStr: `local gctemplates = require("gctemplates");
local tmpl, err = gctemplates.load_template(luatemplate_html);
assert(err == nil);
assert(tmpl);
local data = {
	X = 4,
	Y = 4,
	Message = "Testing"
};
err = tmpl:Execute(buffer, data);
assert(err == nil);`,
			expected: "vertex: (4, 4)\n<div>Test...</div>",
		},
		{
			desc: "parse template string",
			luaStr: `local gctemplates = require("gctemplates");
local tmpl, err = gctemplates.parse_template("tmpl", "({{.X}}, {{.Y}})");
assert(err == nil);
assert(tmpl);
local data = {
	X = 4,
	Y = 4
};
err = tmpl:Execute(buffer, data);
assert(err == nil);`,
			expected: "(4, 4)",
		},
	}

	for _, tC := range testCases {
		buf.Reset()
		t.Run(tC.desc, func(t *testing.T) {
			err = l.DoString(tC.luaStr)
			if tC.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tC.expected, buf.String())
		})
	}
}
