package gcsql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

type queryTestCase struct {
	sqlStr    string
	params    []any
	expectVal []any
}

func TestPrepareContextSQL(t *testing.T) {
	_, err := PrepareContextSQL(nil, "", nil)
	if !assert.Error(t, err) {
		return
	}
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "", "")
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	err = SetTestingDB("mysql", "gochan", "", db)
	if !assert.NoError(t, err) {
		return
	}
	ctx := context.Background()
	mock.ExpectBegin()
	mPrep := mock.ExpectPrepare(`SELECT \* FROM posts`)

	tx, err := BeginContextTx(ctx)
	if !assert.NoError(t, err) {
		return
	}
	stmt, err := PrepareContextSQL(ctx, "SELECT * FROM DBPREFIXposts", tx)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		return
	}
	mock.ExpectCommit()
	mPrep.WillBeClosed()

	assert.NoError(t, tx.Commit())
	assert.NoError(t, stmt.Close())
	assert.NoError(t, mock.ExpectationsWereMet())

	// test context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mPrep = mock.ExpectPrepare(`SELECT \* FROM posts`)
	mPrep.WillDelayFor(6 * time.Second).WillReturnError(sqlmock.ErrCancelled)
	_, err = PrepareContextSQL(ctx, `SELECT * FROM posts`, nil)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
