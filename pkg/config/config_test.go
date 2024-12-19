package config

import (
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestBadTypes(t *testing.T) {
	var c GochanConfig
	err := json.NewDecoder(strings.NewReader(badTypeJSON)).Decode(&c)
	assert.Error(t, err)
}

func TestValidJSON(t *testing.T) {
	var c GochanConfig
	err := json.NewDecoder(strings.NewReader(validCfgJSON)).Decode(&c)
	assert.NoError(t, err)
}

func TestValidateValues(t *testing.T) {
	InitConfig("3.1.0")
	SetRandomSeed("test")
	assert.NoError(t, Cfg.ValidateValues())

	Cfg.CookieMaxAge = "not a duration"
	assert.Error(t, Cfg.ValidateValues())
	Cfg.CookieMaxAge = "1y"
	assert.NoError(t, Cfg.ValidateValues())

	SetTestDBConfig("not a valid driver", "127.0.0.1", "gochan", "gochan", "", "")
	assert.Error(t, Cfg.ValidateValues())
	SetTestDBConfig("postgresql", "127.0.0.1", "gochan", "gochan", "", "")
	assert.NoError(t, Cfg.ValidateValues())
}

type webRootTest struct {
	webRoot    string
	pathArgs   []string
	expectPath string
}

func TestWebPath(t *testing.T) {
	InitConfig("4.0.0")
	testCases := []webRootTest{
		{
			webRoot:    "/",
			pathArgs:   []string{"b", "res", "1234.html"},
			expectPath: "/b/res/1234.html",
		},
		{
			webRoot:    "/chan",
			pathArgs:   []string{"b", "res", "1234.html"},
			expectPath: "/chan/b/res/1234.html",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.expectPath, func(t *testing.T) {
			Cfg.WebRoot = tC.webRoot
			wp := WebPath(tC.pathArgs...)
			assert.Equal(t, tC.expectPath, wp)
		})
	}
}
