package gcutil

import (
	"io"
	"testing"
)

func runErrorTests(name string, err *GcError, t *testing.T) {
	t.Log(name, "(pass as interface)", err)
	if err != nil {
		t.Log(name, ".Error()", err.Error())
		t.Log(name, ".JSON()", err.JSON())
	}
	t.Log()
}

func TestNewError(t *testing.T) {
	err := NewError("error message", false)
	runErrorTests("NewError", err, t)
	wrappedNil := FromError(nil, false)
	runErrorTests("Wrapped nil", wrappedNil, t)
	wrappedEOF := FromError(io.EOF, false)
	runErrorTests("Wrapped EOF", wrappedEOF, t)
	joined := JoinErrors(err, wrappedNil, wrappedEOF)
	runErrorTests("Joined errors", joined, t)
}
