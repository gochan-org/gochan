package testutil

import (
	"errors"
	"os"
	"path"
	"strings"
	"testing"
)

const (
	numParentDirsBeforeFail = 6
)

// PanicIfNotTest panics if the function was called directly or indirectly by a test function via go test
func PanicIfNotTest() {
	if !strings.HasSuffix(os.Args[0], ".test") && os.Args[1] != "-test.run" {
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
		if path.Base(dir) == "gochan" {
			return dir, nil
		}
		if err = os.Chdir(".."); err != nil {
			return dir, err
		}
	}
	return dir, errors.New("test running from unexpected dir, should be in gochan root or the current testing dir")
}
