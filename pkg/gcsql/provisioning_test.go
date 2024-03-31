package gcsql

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

var (
	testingDBDrivers = []string{"mysql", "postgres", "sqlite3"}
)

func TestProvision(t *testing.T) {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}
	config.SetVersion("3.10.1")
	config.SetRandomSeed("test")

	for _, driver := range testingDBDrivers {
		t.Run(driver, func(t *testing.T) {
			config.SetTestDBConfig(driver, "localhost", "gochan", "gochan", "gochan", "")

			gcdb, err = setupDBConn("localhost", driver, "gochan", "gochan", "gochan", "")
			if !assert.NoError(t, err) {
				return
			}

			mock, err := setupMockDB(t, driver, "gochan")
			if !assert.NoError(t, err) {
				return
			}

			if !assert.NoError(t, setupAndProvisionMockDB(t, mock, driver, "gochan")) {
				return
			}
			closeMock(t, mock)
		})
	}
}
