package gcsql

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/events"
)

var (
	errMissingViewFile = errors.New("unable to find reset_views.sql, please reinstall gochan")
)

func ResetViews() error {
	viewsFile := findSQLFile("reset_views.sql")
	if viewsFile == "" {
		return errMissingViewFile
	}
	err := RunSQLFile(viewsFile)
	if err != nil {
		return err
	}
	_, err, recovered := events.TriggerEvent("db-views-reset")
	if err != nil {
		return err
	}
	if recovered {
		return errors.New("recovered from panic while running reset views event")
	}
	return nil
}
