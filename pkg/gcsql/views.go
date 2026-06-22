package gcsql

import (
	"errors"
	"time"

	"github.com/gochan-org/gochan/pkg/events"
)

var (
	errMissingViewFile = errors.New("unable to find reset_views.sql, please reinstall gochan")
)

// view: DBPREFIXv_appeals
type Appeal struct {
	IPBanAppeal
	StaffUsername string    `json:"staff,omitempty"`
	IsBanActive   bool      `json:"-"`
	BanExpiresAt  time.Time `json:"expires"`
	Permanent     bool      `json:"permanent"`
	Timestamp     time.Time `json:"timestamp"`
}

// view: DBPREFIXv_post_reports
type PostReport struct {
	ID         int     `json:"id"`
	StaffID    *int    `json:"-"`
	StaffUser  *string `json:"staff,omitempty"`
	PostID     int     `json:"post"`
	ThreadOP   int     `json:"op"`
	Board      string  `json:"board"`
	ReporterIP string  `json:"reporter_ip"`
	PosterIP   string  `json:"poster_ip"`
	Reason     string  `json:"reason"`
	IsCleared  bool    `json:"is_cleared"`
}

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
