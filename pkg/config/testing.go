package config

import (
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
)

func setDefaultCfgIfNotSet() {
	if cfg == nil {
		cfg = defaultGochanConfig
	}
}

// InitTestConfig should only be used for tests, where a config file wouldn't be loaded
func InitTestConfig() {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
}

// SetTestTemplateDir sets the directory for templates, used only in testing. If it is not run via `go test`, it will panic.
func SetTestTemplateDir(dir string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	cfg.TemplateDir = dir
}

// SetTestDBConfig sets up the database configuration for a testing environment. If it is not run via `go test`, it will panic
func SetTestDBConfig(dbType string, dbHost string, dbName string, dbUsername string, dbPassword string, dbPrefix string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	cfg.DBtype = dbType
	cfg.DBhost = dbHost
	cfg.DBname = dbName
	cfg.DBusername = dbUsername
	cfg.DBpassword = dbPassword
	cfg.DBprefix = dbPrefix
}

// SetRandomSeed is usd to set a deterministic seed to make testing easier. If it is not run via `go test`, it will panic
func SetRandomSeed(seed string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	cfg.RandomSeed = seed
}

// SetSystemCriticalConfig sets system critical configuration values in testing. It will panic if it is not run in a
// test environment
func SetSystemCriticalConfig(systemCritical *SystemCriticalConfig) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	cfg.SystemCriticalConfig = *systemCritical
}

// SetSiteConfig sets the site configuration values in testing. It will panic if it is not run in a test environment
func SetSiteConfig(siteConfig *SiteConfig) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	cfg.SiteConfig = *siteConfig
}

// SetBoardConfig applies the configuration to the given board. It will panic if it is not run in a test environment
func SetBoardConfig(board string, boardCfg *BoardConfig) error {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	if err := boardCfg.validateEmbedMatchers(); err != nil {
		return err
	}
	if board == "" {
		cfg.BoardConfig = *boardCfg
	} else {
		boardConfigs[board] = *boardCfg
	}
	return nil
}
