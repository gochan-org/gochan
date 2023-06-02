package main

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	errDuplicateFilename = errors.New("a file with that filename has already been uploaded")
)

func InitPlugin() error {
	events.RegisterEvent([]string{"incoming-upload"}, func(trigger string, uploadInterface ...interface{}) error {
		if len(uploadInterface) == 0 {
			return nil
		}
		upload, ok := uploadInterface[0].(*gcsql.Upload)
		if !ok {
			return errors.New("invalid upload interface passed to incoming-upload")
		}
		if upload == nil {
			return nil
		}
		var count int
		err := gcsql.QueryRowSQL("SELECT COUNT(*) FROM DBPREFIXfiles WHERE original_filename = ?",
			[]interface{}{upload.OriginalFilename}, []interface{}{&count})
		if err != nil {
			return err
		}
		if count > 0 {
			// one or more posts with the same filename exist
			return errDuplicateFilename
		}
		return nil
	})

	return nil
}
