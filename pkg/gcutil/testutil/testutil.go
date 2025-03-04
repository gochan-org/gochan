package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
