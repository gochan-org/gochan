package gcsql

import (
	"testing"

	_ "github.com/DATA-DOG/go-sqlmock"
)

func makeTestPost(post *Post, bump bool) (err error) {
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT COALESCE(MAX(id), 0) + 1 FROM gc_threads`,
	)).ExpectQuery().WillReturnRows(
		sqm.NewRows([]string{"a"}).AddRow(1),
	)

	err = InsertPost(post, bump)

	if err != nil {
		return err
	}
	return sqm.ExpectationsWereMet()
}

func TestInsertPosts(t *testing.T) {
	err := makeTestPost(&Post{
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
