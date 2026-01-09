package config

import (
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/rs/zerolog"
)

func setDefaultCfgIfNotSet() {
	if cfg == nil {
		cfg = defaultGochanConfig
	}
}

// InitTestConfig should only be used for tests, where a config file wouldn't be loaded
func InitTestConfig() {
	testutil.PanicIfNotTest()
	loadFileInTest = false
	cfg = defaultGochanConfig
	boardConfigs = make(map[string]BoardConfig)
	cfg.LogLevelStr = "trace"
	cfg.logLevelParsed = true
	cfg.logLevel = zerolog.TraceLevel
}

// SetTestTemplateDir sets the directory for templates, used only in testing. If it is not run via `go test`, it will panic.
func SetTestTemplateDir(dir string) {
	testutil.PanicIfNotTest()
	cfg = defaultGochanConfig
	boardConfigs = make(map[string]BoardConfig)

	cfg.TemplateDir = dir
}

// SetTestDBConfig sets up the database configuration for a testing environment. If it is not run via `go test`, it will panic
func SetTestDBConfig(dbType string, dbHost string, dbName string, dbUsername string, dbPassword string, dbPrefix string, jsonPath ...string) {
	testutil.PanicIfNotTest()
	cfg = defaultGochanConfig
	if boardConfigs == nil {
		boardConfigs = make(map[string]BoardConfig)
	}

	cfg.DBtype = dbType
	cfg.DBhost = dbHost
	cfg.DBname = dbName
	cfg.DBusername = dbUsername
	cfg.DBpassword = dbPassword
	cfg.DBprefix = dbPrefix
	cfg.DBTimeoutSeconds = 600
	if len(jsonPath) > 0 {
		cfg.jsonLocation = jsonPath[0]
	}
}

// SetRandomSeed is usd to set a deterministic seed to make testing easier. If it is not run via `go test`, it will panic
func SetRandomSeed(seed string) {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()
	cfg.RandomSeed = seed
}

// SetBoardConfig applies the configuration to the given board. It will panic if it is not run in a test environment
func SetBoardConfig(board string, boardCfg *BoardConfig) error {
	testutil.PanicIfNotTest()
	setDefaultCfgIfNotSet()

	if err := boardCfg.validateEmbedMatchers(); err != nil {
		return err
	}
	boardCfg.isGlobal = board == ""
	if board == "" {
		boardCfgPath := cfg.BoardConfig.boardConfigPath
		cfg.BoardConfig = *boardCfg
		cfg.boardConfigPath = boardCfgPath
	} else {
		boardConfigs[board] = *boardCfg
	}
	return nil
}
