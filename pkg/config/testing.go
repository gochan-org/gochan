package config

import "github.com/gochan-org/gochan/pkg/gcutil/testutil"

func setDefaultCfgIfNotSet() {
	if Cfg == nil {
		Cfg = defaultGochanConfig
	}
}

// SetVersion should only be used for tests, where a config file wouldn't be loaded
func SetVersion(version string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	Cfg.Version = ParseVersion(version)
}

// SetTestTemplateDir sets the directory for templates, used only in testing. If it is not run via `go test`, it will panic.
func SetTestTemplateDir(dir string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	Cfg.TemplateDir = dir
}

// SetTestDBConfig sets up the database configuration for a testing environment. If it is not run via `go test`, it will panic
func SetTestDBConfig(dbType string, dbHost string, dbName string, dbUsername string, dbPassword string, dbPrefix string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	Cfg.DBtype = dbType
	Cfg.DBhost = dbHost
	Cfg.DBname = dbName
	Cfg.DBusername = dbUsername
	Cfg.DBpassword = dbPassword
	Cfg.DBprefix = dbPrefix
}

// SetRandomSeed is usd to set a deterministic seed to make testing easier. If it is not run via `go test`, it will panic
func SetRandomSeed(seed string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	Cfg.RandomSeed = seed
}

// SetSystemCriticalConfig sets system critical configuration values in testing. It will panic if it is not run in a
// test environment
func SetSystemCriticalConfig(systemCritical *SystemCriticalConfig) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	Cfg.SystemCriticalConfig = *systemCritical
}

// SetSiteConfig sets the site configuration values in testing. It will panic if it is not run in a test environment
func SetSiteConfig(siteConfig *SiteConfig) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	Cfg.SiteConfig = *siteConfig
}

// SetBoardConfig applies the configuration to the given board. It will panic if it is not run in a test environment
func SetBoardConfig(board string, boardCfg *BoardConfig) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	if board == "" {
		Cfg.BoardConfig = *boardCfg
	} else {
		boardConfigs[board] = *boardCfg
	}
}
