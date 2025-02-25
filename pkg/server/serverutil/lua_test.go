package serverutil

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var (
	buf  bytes.Buffer
	data = map[string]any{
		"logText": "text goes here",
	}
	luaStringTemplateTestCases = []luaTemplateTestCase[string]{
		{
			desc:     "minify HTML",
			template: gctemplates.ErrorPage,
			data:     data,
			luaScript: `local serverutil = require("serverutil")
				return serverutil.minify_template("manage_viewlog.html", data, buf, "text/html")`,
			expectString: `<textarea class=viewlog rows=24 spellcheck=false readonly>text goes here</textarea>`,
		},
		{
			desc:     "minify HTML with nil data",
			template: gctemplates.ManageViewLog,
			luaScript: `local serverutil = require("serverutil")
			return serverutil.minify_template("manage_viewlog.html", nil, buf, "text/html")`,
			expectString: `<textarea class=viewlog rows=24 spellcheck=false readonly></textarea>`,
		},
		{
			desc:     "error, unrecognized template name",
			template: "invalid_template",
			luaScript: `local serverutil = require("serverutil")
			return serverutil.minify_template("invalid_template", nil, buf, "text/html")`,
			expectError: true,
		},
	}
)

type luaTemplateTestCase[T string | template.Template] struct {
	desc         string
	template     T
	data         any
	expectError  bool
	expectString string
	luaScript    string
}

func TestLuaTemplates(t *testing.T) {
	testutil.GoToGochanRoot(t)
	config.SetTestTemplateDir("templates")
	siteConfig := config.GetSiteConfig()
	siteConfig.MinifyJS = true
	config.SetSiteConfig(siteConfig)

	for _, tC := range luaStringTemplateTestCases {
		t.Run(tC.desc, func(t *testing.T) {
			buf.Reset()
			l := lua.NewState()
			defer l.Close()
			l.PreloadModule("serverutil", PreloadModule)
			l.SetGlobal("buf", luar.New(l, &buf))
			l.SetGlobal("data", luar.New(l, tC.data))
			if !assert.NoError(t, l.DoString(tC.luaScript)) {
				t.FailNow()
			}
			errLV := l.Get(-1)
			var err error
			if errLV.Type() != lua.LTNil {
				err = errLV.(*lua.LUserData).Value.(error)
			}
			if tC.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tC.expectString, buf.String())
			}
		})
	}
}
