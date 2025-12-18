package config

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

// initializeExampleConfig copies the example config file to a temporary directory, (with optional modifications
// applied by modifyReadCfg before writing to the temporary file) and initializes the global config from it.
func initializeExampleConfig(t *testing.T, basePath string, modifyReadCfg func(cfg *GochanConfig)) error {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	cfg = nil
	boardConfigs = make(map[string]BoardConfig)

	// copy example config to temporary directory
	exampleFd, err := os.Open("examples/configs/gochan.example.json")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer exampleFd.Close()

	var baseCfg GochanConfig
	if !assert.NoError(t, json.NewDecoder(exampleFd).Decode(&baseCfg)) {
		t.FailNow()
	}
	if modifyReadCfg != nil {
		modifyReadCfg(&baseCfg)
	}

	tempFd, err := os.Create(path.Join(basePath, "gochan.json"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer tempFd.Close()
	writeJsonFile(t, path.Join(basePath, "gochan.json"), baseCfg)

	loadFileInTest = true
	StandardConfigSearchPaths = []string{path.Join(basePath, "gochan.json")}
	return InitConfig()
}

func writeJsonFile(t *testing.T, filePath string, data any) {
	t.Helper()
	fileFd, err := os.Create(filePath)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer fileFd.Close()
	if !assert.NoError(t, json.NewEncoder(fileFd).Encode(data)) {
		t.FailNow()
	}
	if !assert.NoError(t, fileFd.Close()) {
		t.FailNow()
	}
}
