package gcsql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

func closeMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if gcdb == nil || gcdb.db == nil || mock == nil {
		return
	}
	mock.ExpectClose()
	assert.NoError(t, Close())
	err := mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func setupSQLConfig(dbDriver string, dbName string, dbPrefix string) *config.SQLConfig {
	return &config.SQLConfig{
		DBtype:               dbDriver,
		DBhost:               "localhost",
		DBname:               dbName,
		DBusername:           "gochan",
		DBpassword:           "gochan",
		DBprefix:             dbPrefix,
		DBTimeoutSeconds:     config.DefaultSQLTimeout,
		DBMaxOpenConnections: config.DefaultSQLMaxConns,
		DBMaxIdleConnections: config.DefaultSQLMaxConns,
		DBConnMaxLifetimeMin: config.DefaultSQLConnMaxLifetimeMin,
	}
}

func TestOpenMySQL(t *testing.T) {
	var err error
	gcdb, err = setupDBConn(setupSQLConfig("mysql", "gochan", ""))
	if !assert.NoError(t, err) {
		return
	}

	var mock sqlmock.Sqlmock
	gcdb.db, mock, err = sqlmock.New()
	assert.NoError(t, err)
	defer closeMock(t, mock)
}

func TestOpenPostgres(t *testing.T) {
	var err error
	gcdb, err = setupDBConn(setupSQLConfig("postgres", "gochan", ""))
	if !assert.NoError(t, err) {
		return
	}

	var mock sqlmock.Sqlmock
	gcdb.db, mock, err = sqlmock.New()
	assert.NoError(t, err)
	defer closeMock(t, mock)
}

func TestOpenSQLite3(t *testing.T) {
	var err error
	gcdb, err = setupDBConn(setupSQLConfig("sqlite3", "gochan", ""))
	if !assert.NoError(t, err) {
		return
	}

	var mock sqlmock.Sqlmock
	gcdb.db, mock, err = sqlmock.New()
	assert.NoError(t, err)
	defer closeMock(t, mock)
}

func TestOpenUnrecognizedDriver(t *testing.T) {
	assert.NoError(t, Close())
	_, err := setupDBConn(setupSQLConfig("wat", "gochan", ""))
	assert.Error(t, err)
}
