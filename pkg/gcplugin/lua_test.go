package gcplugin

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	versionStr = `return _GOCHAN_VERSION
`
	structPassingStr = `print(string.format("Receiving post from %q", post.Name))
print(string.format("Message before changing: %q", post.MessageText))
post.MessageText = "Message modified by a plugin\n"
post.MessageHTML = "Message modified by a plugin<br />"
print(string.format("Modified message text: %q", post.MessageText))
`
)

func initPluginTests() {
	config.SetVersion("3.1")
	initLua()
}

func TestVersionFunction(t *testing.T) {
	initPluginTests()
	err := lState.DoString(versionStr)
	if err != nil {
		t.Fatal(err.Error())
	}
	testingVersionStr := lState.Get(-1).(lua.LString)
	if testingVersionStr != "3.1" {
		t.Fatalf("%q != \"3.1\"", testingVersionStr)
	}
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
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("Modified message text after Lua: %q", p.MessageRaw)
	if p.MessageRaw != "Message modified by a plugin\n" || p.Message != "Message modified by a plugin<br />" {
		t.Fatal("message was not properly modified by plugin")
	}
}
