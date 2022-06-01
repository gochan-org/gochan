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
)

func TestVersionFunction(t *testing.T) {
	config.SetVersion("3.1")
	initLua()
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
	initLua()
	p := &gcsql.Post{
		Name:        "Joe Poster",
		Email:       "joeposter@gmail.com",
		MessageHTML: "Message test<br />",
		MessageText: "Message text\n",
	}
	uData := lState.NewUserData()
	uData.Value = p
	lState.SetGlobal("post", luar.New(lState, p))
	err := lState.DoString(`print("Receiving post from " .. post["Name"])`)

	if err != nil {
		t.Fatal(err.Error())
	}

}
