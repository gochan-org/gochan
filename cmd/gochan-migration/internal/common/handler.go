package common

import (
	"errors"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
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
		from = " from " + from + " "
	}
	return "unable to migrate" + from + ": " + me.errMessage
}

func NewMigrationError(oldChanType string, errMessage string) *MigrationError {
	return &MigrationError{oldChanType: oldChanType, errMessage: errMessage}
}

type DBOptions struct {
	Host        string
	DBType      string
	Username    string
	Password    string
	OldDBName   string
	OldChanType string
	NewDBName   string
}

// DBMigrator is used for handling the migration from one database type to a
// database compatible with gochan 3.x onward
type DBMigrator interface {
	// Init sets the variables for connecting to the databases
	Init(options DBOptions) error

	// MigrateDB migrates the imageboard data (posts, boards, etc) to the new database
	MigrateDB() error

	// Close closes the database if initialized
	Close() error
}
