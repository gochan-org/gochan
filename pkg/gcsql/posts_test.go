package gcsql

import (
	"testing"
)

func TestInsertPosts(t *testing.T) {
	err := InsertPost(&Post{
		ParentID:         0,
		BoardID:          1,
		Name:             "Joe Poster",
		Tripcode:         "Blah",
		Email:            "any@example.com",
		Subject:          "First thread",
		MessageHTML:      "First post best post",
		MessageText:      "First post best post",
		Password:         "12345",
		Filename:         "12345.png",
		FilenameOriginal: "somefile.png",
		FileChecksum:     "abcd1234",
		FileExt:          ".png",
		Filesize:         1000,
		ImageW:           2000,
		ImageH:           2000,
		ThumbW:           250,
		ThumbH:           250,
		IP:               "192.168.56.1",
	}, false)
	if err != nil {
		t.Fatal(err.Error())
	}
}
