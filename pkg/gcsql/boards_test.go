package gcsql

import (
	"testing"
)

func TestCreateBoard(t *testing.T) {
	err := CreateDefaultBoardIfNoneExist()
	if err != nil {
		t.Fatalf("Failed creating default board if none exists: %s", err.Error())
	}
}
