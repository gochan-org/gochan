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
// database compatible with gochan 3.x onward
type DBMigrator interface {
	// Init sets the variables for connecting to the databases
	Init(options *MigrationOptions) error

	// IsMigrated checks to see if the database has already been migrated and quits if it has
	// and returns any errors that aren't "table doesn't exist". if the boolean value is true,
	// it can be assumed that the database has already been migrated and gochan-migration
	// will exit
	IsMigrated() (bool, error)

	// IsMigratingInPlace returns true if the old database name is the same as the new database name,
	// meaning that the tables will be altered to match the new schema, instead of creating tables in the
	// new database and copying data over
	IsMigratingInPlace() bool

	// MigrateDB alters the database schema to match the new schema, then migrates the imageboard
	// data (posts, boards, etc) to the new database. It is assumed that MigrateDB will handle
	// logging any errors that occur during the migration
	MigrateDB() (bool, error)

	// MigrateBoards migrates the board sections (if they exist) and boards if each one
	// doesn't already exists
	MigrateBoards() error

	// MigratePosts gets the threads and replies (excluding deleted ones) in the old database, and inserts them into
	// the new database, creating new threads to avoid putting replies in threads that already exist
	MigratePosts() error

	// MigrateStaff gets the staff list in the old board and inserts them into the new board if
	// the username doesn't already exist. Migrated staff accounts will need to have their password reset
	// in order to be logged into
	MigrateStaff() error

	// MigrateBans migrates IP bans, appeals, and filters
	MigrateBans() error

	// MigrateAnnouncements gets the list of public and staff announcements in the old database
	// and inserts them into the new database
	MigrateAnnouncements() error

	// Close closes the database if initialized and deletes any temporary columns created
	Close() error
}
