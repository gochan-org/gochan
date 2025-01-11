package common

import (
	"errors"
)

const (
	DirNoAction = iota
	DirCopy
	DirMove
)

var (
	ErrInvalidSchema = errors.New("invalid database schema for old database")
)

type MigrationError struct {
	oldChanType string
	errMessage  string
}

func (me *MigrationError) OldChanType() string {
	return me.oldChanType
}

func (me *MigrationError) Error() string {
	from := me.oldChanType
	errStr := "unable to migrate"
	if from != "" {
		errStr += " from " + from
	}
	if me.errMessage != "" {
		errStr += ": " + me.errMessage
	}
	return errStr
}

func NewMigrationError(oldChanType string, errMessage string) *MigrationError {
	return &MigrationError{oldChanType: oldChanType, errMessage: errMessage}
}

type MigrationOptions struct {
	ChanType      string
	OldChanRoot   string
	OldChanConfig string
	OldDBName     string
	NewDBName     string
}

// DBMigrator is used for handling the migration from one database type to a
// database compatible with the latest gochan database version
type DBMigrator interface {
	// Init sets up the migrator and sets up the database connection(s)
	Init(options *MigrationOptions) error

	// IsMigrated returns true if the database is already migrated, and an error if any occured,
	// excluding missing table errors
	IsMigrated() (bool, error)

	// IsMigratingInPlace returns true if the source database and the destination database are both the
	// same installation, meaning both have the same host/connection, database, table and prefix, meaning that
	// the tables will be altered during the migration to match the new schema, instead of creating tables in
	// the destination database and copying data over
	IsMigratingInPlace() bool

	// MigrateDB handles migration of the source database, altering it in place or migrating it to the configured
	// gochan database. It returns true if the database is already migrated and an error if any occured. It is
	// assumed that MigrateDB implementations will handle logging any errors that occur during the migration
	MigrateDB() (bool, error)

	// MigrateBoards migrates the board sections and boards if each one doesn't already exists
	MigrateBoards() error

	// MigratePosts migrates the threads and replies (excluding deleted ones), creating new threads where necessary
	MigratePosts() error

	// MigrateStaff migrates the staff, creating new staff accounts that don't already exist. Accounts created by this
	// will need to have their password reset in order to be logged into
	MigrateStaff() error

	// MigrateBans migrates IP bans, appeals, and filters
	MigrateBans() error

	// MigrateAnnouncements migrates the list of public and staff announcements, if applicable
	MigrateAnnouncements() error

	// Close closes the database if initialized and deletes any temporary columns created
	Close() error
}
