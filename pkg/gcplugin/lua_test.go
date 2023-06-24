package gcplugin

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
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

	eventsTestingStr = `event_register({"newPost"}, function(tr, ...)
	print("newPost triggered :D")
	for i, v in ipairs(arg) do
		print(i .. ": " .. tostring(v))
	end
end)

event_trigger("newPost", "blah", 16, 3.14, true, nil)`
)

func initPluginTests() {
	config.SetVersion("3.7.0")
	initLua()
}

func TestVersionFunction(t *testing.T) {
	initPluginTests()
	err := lState.DoString(versionStr)
	assert.Nil(t, err)
	testingVersionStr := lState.Get(-1).(lua.LString)
	assert.EqualValues(t, "3.7.0", testingVersionStr)
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
	assert.Nil(t, err)
	t.Logf("Modified message text after Lua: %q", p.MessageRaw)
	if p.MessageRaw != "Message modified by a plugin\n" || p.Message != "Message modified by a plugin<br />" {
		t.Fatal("message was not properly modified by plugin")
	}
}

func TestEventPlugins(t *testing.T) {
	initPluginTests()
	err := lState.DoString(eventsTestingStr)
	assert.Nil(t, err)
}
