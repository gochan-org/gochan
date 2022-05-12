package common

import (
	"errors"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	DirNoAction = iota
	DirCopy
	DirMove
)

var (
	ErrInvalidSchema     = errors.New("invalid database schema for old database")
	ErrUnsupportedDBType = errors.New("unsupported SQL driver, currently only MySQL and Postgres are supported")
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
	if from != "" {
		from = " from " + from
	}
	return "unable to migrate " + from + ": " + me.errMessage
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
	DirAction     int
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

	// MigrateDB migrates the imageboard data (posts, boards, etc) to the new database
	MigrateDB() (bool, error)

	// MigrateBoards gets info about the old boards in the board table and inserts each one
	// into the new database if they don't already exist
	MigrateBoards() error

	// MigratePosts gets the threads and replies in the old database, and inserts them into
	// the new database, creating new threads to avoid putting replies in threads that already
	// exist
	MigratePosts() error

	// MigrateStaff gets the staff list in the old board and inserts them into the new board if
	// the username doesn't already exist. It sets the starting password to the given password
	MigrateStaff(password string) error

	// MigrateBans gets the list of bans and appeals in the old database and inserts them into the
	// new one if, for each entry, the IP/name/etc isn't already banned for the same length
	// e.g. 1.1.1.1 is permabanned on both, 1.1.2.2 is banned for 5 days on both, etc
	MigrateBans() error

	// MigrateAnnouncements gets the list of public and staff announcements in the old database
	// and inserts them into the new database,
	MigrateAnnouncements() error

	// Close closes the database if initialized
	Close() error
}
