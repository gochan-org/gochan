package gctemplates

import (
	"bytes"
	"errors"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	maxSubdirs = 6 // max expected depth of the current directory before we throw an error
)

func goToGochanRoot(t *testing.T) (string, error) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for d := 0; d < maxSubdirs; d++ {
		if path.Base(dir) == "gochan" {
			return dir, nil
		}
		if err = os.Chdir(".."); err != nil {
			return dir, err
		}
		if dir, err = os.Getwd(); err != nil {
			return dir, err
		}
	}
	return dir, errors.New("test running from unexpected dir, should be in gochan root or the current testing dir")
}

func TestLuaTemplateFunctions(t *testing.T) {
	gochanRoot, err := goToGochanRoot(t)
	assert.NoError(t, err)
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
