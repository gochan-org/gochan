package config

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

type preloadTest struct {
	desc        string
	luaIn       string
	expectOut   lua.LValue
	expectError bool
}

func (tC *preloadTest) run(t *testing.T) {
	l := lua.NewState()
	l.PreloadModule("config", PreloadModule)
	err := l.DoString(tC.luaIn)
	if tC.expectError {
		assert.Error(t, err)
		return
	} else if !assert.NoError(t, err) {
		return
	}
	luaOut := l.Get(-1)
	assert.Equal(t, tC.expectOut, luaOut)
}

func TestPreload(t *testing.T) {
	testutil.GoToGochanRoot(t)
	InitTestConfig()
	cfg.ListenAddress = "127.0.0.1"
	cfg.CookieMaxAge = "1y"
	cfg.DefaultStyle = "sitestyle.css"
	boardConfigs["b"] = BoardConfig{DefaultStyle: "boardstyle.css"}
	testCases := []preloadTest{
		{
			desc: "access system critical config from lua",
			luaIn: `local config = require("config")
sys_cfg = config.system_critical_config()
return sys_cfg.ListenAddress`,
			expectOut: lua.LString("127.0.0.1"),
		},
		{
			desc: "access site config from lua",
			luaIn: `local config = require("config")
site_cfg = config.site_config()
return site_cfg.CookieMaxAge`,
			expectOut: lua.LString("1y"),
		},
		{
			desc: "access board config from lua",
			luaIn: `local config = require("config")
board_cfg = config.board_config("b")
global_cfg = config.board_config("")
return global_cfg.DefaultStyle .. " " .. board_cfg.DefaultStyle`,
			expectOut: lua.LString("sitestyle.css boardstyle.css"),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, tC.run)
	}
}
