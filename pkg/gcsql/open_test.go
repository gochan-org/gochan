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

func initMock(t *testing.T, dbDriver string) (sqlmock.Sqlmock, error) {
	t.Helper()
	err := Close()
	assert.NoError(t, err)

	gcdb, err = setupDBConn("localhost", dbDriver, "gochan", "gochan", "gochan", "")
	if !assert.NoError(t, err) {
		return nil, err
	}

	var mock sqlmock.Sqlmock
	gcdb.db, mock, err = sqlmock.New()
	return mock, err
}

func TestOpenMySQL(t *testing.T) {
	var err error
	gcdb, err = setupDBConn("localhost", "mysql", "gochan", "gochan", "gochan", "")
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
	gcdb, err = setupDBConn("localhost", "postgres", "gochan", "gochan", "gochan", "")
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
	gcdb, err = setupDBConn("localhost", "sqlite3", "gochan", "gochan", "gochan", "")
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
	_, err := setupDBConn("localhost", "wat", "gochan", "gochan", "gochan", "")
	assert.Error(t, err)
}
