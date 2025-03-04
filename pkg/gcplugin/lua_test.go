package gcplugin

import (
	"html/template"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	versionStr = `return _GOCHAN_VERSION
`
	structPassingStr = `print(string.format("Receiving post from %q", post.Name))
print(string.format("Message before changing: %q", post.MessageRaw))
post.MessageRaw = "Message modified by a plugin\n"
post.Message = "Message modified by a plugin<br />"
print(string.format("Modified message text: %q", post.MessageText))`

	eventsTestingStr = `local events = require("events")
events.register_event({"newPost"}, function(tr, ...)
	print("newPost triggered :D")
	for i, v in ipairs(arg) do
		print(i .. ": " .. tostring(v))
	end
end)

events.trigger_event("newPost", "blah", 16, 3.14, true, nil)`

	configTestingStr = `local config = require("config")
local system_critical_cfg = config.system_critical_config()
local site_cfg = config.site_config()
local board_cfg = config.board_config()
return { ListenAddress = system_critical_cfg.ListenAddress, SiteSlogan = site_cfg.SiteSlogan, DefaultStyle = board_cfg.DefaultStyle }`
)

func initPluginTests() {
	config.SetVersion("4.0.2")
	initLua()
}

func TestVersionFunction(t *testing.T) {
	initPluginTests()
	err := lState.DoString(versionStr)
	assert.NoError(t, err)
	testingVersionStr := lState.Get(-1).(lua.LString)
	assert.EqualValues(t, config.GetVersion().String(), testingVersionStr)
}

func TestStructPassing(t *testing.T) {
	initPluginTests()
	p := &gcsql.Post{
		Name:       "Joe Poster",
		Email:      "joeposter@gmail.com",
		Message:    "Message test<br />",
		MessageRaw: "Message text\n",
	}
	lState.SetGlobal("post", luar.New(lState, p))
	err := lState.DoString(structPassingStr)
	assert.NoError(t, err)
	t.Logf("Modified message text after Lua: %q", p.MessageRaw)
	assert.Equal(t, "Message modified by a plugin\n", p.MessageRaw)
	assert.Equal(t, template.HTML("Message modified by a plugin<br />"), p.Message)
}

func TestEventModule(t *testing.T) {
	initPluginTests()
	err := lState.DoString(eventsTestingStr)
	assert.NoError(t, err)
}

func TestConfigModule(t *testing.T) {
	testutil.GoToGochanRoot(t)
	if !assert.NoError(t, config.InitConfig("4.1.0")) {
		t.FailNow()
	}
	initPluginTests()
	err := lState.DoString(configTestingStr)
	assert.NoError(t, err)
	returnTable := lState.CheckTable(-1)
	assert.Equal(t, "127.0.0.1", returnTable.RawGetString("ListenAddress").(lua.LString).String())
	assert.Equal(t, "Gochan testing", returnTable.RawGetString("SiteSlogan").(lua.LString).String())
	assert.Equal(t, "pipes.css", returnTable.RawGetString("DefaultStyle").(lua.LString).String())
}

func TestLuaURL(t *testing.T) {
	initPluginTests()
	err := lState.DoString(`local url = require("url")
local joined = url.join_path("test", "path")
local path_escaped = url.path_escape("test +/string")
local path_unescaped = url.path_unescape(path_escaped)
local query_escaped = url.query_escape("test +/string")
local query_unescaped, err = url.query_unescape(query_escaped)
return joined, query_escaped, query_unescaped, err`)
	assert.NoError(t, err)
	joined := lState.CheckString(-4)
	pathEscaped := lState.CheckString(-3)
	pathUnescaped := lState.CheckString(-2)
	queryEscaped := lState.CheckString(-3)
	queryUnescaped := lState.CheckString(-2)
	errLV := lState.CheckAny(-1)
	assert.Equal(t, "test/path", joined)
	assert.Equal(t, "test+%2B%2Fstring", pathEscaped)
	assert.Equal(t, "test +/string", pathUnescaped)
	assert.Equal(t, "test+%2B%2Fstring", queryEscaped)
	assert.Equal(t, "test +/string", queryUnescaped)
	assert.Equal(t, errLV.Type(), lua.LTNil)
	ClosePlugins()
}

func TestLoadPlugin(t *testing.T) {
	testutil.GoToGochanRoot(t)
	initPluginTests()
	assert.NoError(t, LoadPlugins([]string{"examples/plugins/uploadfilenameupper.lua"}))
	assert.NoError(t, LoadPlugins(nil))
	assert.Error(t, LoadPlugins([]string{"not_a_file.lua"}))
	assert.Error(t, LoadPlugins([]string{"invalid_ext.dll"}))
	assert.ErrorContains(t, LoadPlugins([]string{"not_a_file.so"}), "realpath failed")
	ClosePlugins()
}
