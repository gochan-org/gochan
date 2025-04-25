package testutil

import (
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

const (
	numParentDirsBeforeFail = 6 // max expected depth of the current directory before we throw an error
)

// PanicIfNotTest panics if the function was called directly or indirectly by a test function via go test
func PanicIfNotTest() {
	if !testing.Testing() {
		panic("the testutil package should only be used in tests")
	}
}

// GoToGochanRoot gets the
func GoToGochanRoot(t *testing.T) (string, error) {
	t.Helper()

	var dir string
	var err error
	for d := 0; d < numParentDirsBeforeFail; d++ {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
		if filepath.Base(dir) == "gochan" || dir == "/vagrant" {
			return dir, nil
		}
		if err = os.Chdir(".."); err != nil {
			return dir, err
		}
	}
	return dir, errors.New("test running from unexpected dir, should be in gochan root or the current testing dir")
}

// GetTestLogs returns logs with info, warn, and error levels respectively for testing
func GetTestLogs(t *testing.T) (*zerolog.Event, *zerolog.Event, *zerolog.Event) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	return logger.Info(), logger.Warn(), logger.Error()
}

// FuzzyTime is a wrapper around time.Time that allows for fuzzy matching of time values within a 10-minute window
// to be used in SQL query tests
type FuzzyTime time.Time

func (f FuzzyTime) Match(val driver.Value) bool {
	ft := time.Time(f).Truncate(time.Minute)
	var t time.Time
	switch timeVal := val.(type) {
	case time.Time:
		t = timeVal
	case string:
		var err error
		t, err = time.Parse(time.RFC3339, timeVal)
		if err != nil {
			return false
		}
	default:
		return false
	}

	return t.Truncate(10 * time.Minute).Equal(ft.Truncate(10 * time.Minute))
}
