package gcsql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestOpenMySQL(t *testing.T) {
	var err error
	gcdb, err = setupDBConn(setupSqlTestConfig("mysql", "gochan", ""))
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
	gcdb, err = setupDBConn(setupSqlTestConfig("postgres", "gochan", ""))
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
	gcdb, err = setupDBConn(setupSqlTestConfig("sqlite3", "gochan", ""))
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
	_, err := setupDBConn(setupSqlTestConfig("wat", "gochan", ""))
	assert.Error(t, err)
}
