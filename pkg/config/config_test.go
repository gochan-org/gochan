package config

import (
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
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
	testutil.GoToGochanRoot(t)
	if !assert.NoError(t, InitConfig("4.1.0")) {
		t.FailNow()
	}
	SetRandomSeed("test")
	assert.NoError(t, cfg.ValidateValues())

	cfg.CookieMaxAge = "not a duration"
	assert.Error(t, cfg.ValidateValues())
	cfg.CookieMaxAge = "1y"
	assert.NoError(t, cfg.ValidateValues())

	SetTestDBConfig("not a valid driver", "127.0.0.1", "gochan", "gochan", "", "")
	assert.Error(t, cfg.ValidateValues())
	SetTestDBConfig("postgresql", "127.0.0.1", "gochan", "gochan", "", "")
	assert.NoError(t, cfg.ValidateValues())
}

type webRootTest struct {
	webRoot    string
	pathArgs   []string
	expectPath string
}

func TestWebPath(t *testing.T) {
	testutil.GoToGochanRoot(t)
	if !assert.NoError(t, InitConfig("4.1.0")) {
		t.FailNow()
	}
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
			cfg.WebRoot = tC.webRoot
			wp := WebPath(tC.pathArgs...)
			assert.Equal(t, tC.expectPath, wp)
		})
	}
}
