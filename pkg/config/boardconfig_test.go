package config

import (
	"path"
	"testing"

	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	"github.com/stretchr/testify/assert"
)

func TestWriteBoardConfig(t *testing.T) {
	basePath := t.TempDir()

	assert.NoError(t, initializeExampleConfig(t, basePath, func(c *GochanConfig) {
		c.DocumentRoot = basePath
		c.LogDir = path.Join(basePath, "logs")
		c.DBtype = "sqlite3"
		c.DBhost = path.Join(basePath, "gochan.db")
		c.DBname = "gochan"
		c.DBusername = "gochan"
		c.DBpassword = "gochan"
		c.StaffSessionDuration = "1mo"
		c.CookieMaxAge = "1mo"
		c.SiteName = "TestWriteBoardConfig"
		c.CustomFlags = []geoip.Country{{Flag: "flag.png", Name: "Country"}}
		writeJsonFile(t, path.Join(basePath, "equal-config.json"), c.BoardConfig)
		writeJsonFile(t, path.Join(basePath, "changed-config.json"), c.BoardConfig)
	}))
	assert.Equal(t, "TestWriteBoardConfig", cfg.SiteName)

	defer resetTestConfig(t)

	assert.NoError(t, gcutil.InitLogs(path.Join(basePath, "logs"), &gcutil.LogOptions{
		LogLevel: cfg.logLevel,
	}))

	assert.NoError(t, ReloadBoardConfig("equal"))
	assert.NoError(t, ReloadBoardConfig("changed"))

	equalCfg := GetBoardConfig("equal")
	changedCfg := GetBoardConfig("changed")
	assert.False(t, equalCfg.IsGlobal())
	assert.False(t, changedCfg.IsGlobal())
	assert.Equal(t, path.Join(basePath, "equal-config.json"), equalCfg.boardConfigPath)
	assert.Equal(t, path.Join(basePath, "changed-config.json"), changedCfg.boardConfigPath)
	assert.Equal(t, []geoip.Country{{Flag: "flag.png", Name: "Country"}}, changedCfg.CustomFlags)

	// make sure that the global config wasn't mistakenly changed by modifying the board config
	assert.True(t, cfg.isGlobal)
	assert.Empty(t, cfg.boardConfigPath)

	// test writing of board config and reloading to confirm changes were saved
	changedCfg.DefaultStyle = "new-style.css"
	boardConfigs["changed"] = *changedCfg
	assert.NoError(t, WriteBoardConfig("changed"))

	assert.NoError(t, ReloadBoardConfig("changed"))
	changedCfg = GetBoardConfig("changed")
	assert.Equal(t, "new-style.css", changedCfg.DefaultStyle)
}
