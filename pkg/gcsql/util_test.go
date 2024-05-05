package gcsql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

type testCasePrepareContextSQL struct {
	name       string
	f          func(context.Context, string, *sql.Tx) (*sql.Stmt, error)
	shouldFail bool
	timeout    time.Duration
	delay      time.Duration
}

func setupMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "", "")
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return nil
	}
	err = SetTestingDB("mysql", "gochan", "", db)
	if !assert.NoError(t, err) {
		return nil
	}
	return mock
}

func tcPrepareContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *testCasePrepareContextSQL) {
	const query = "SELECT * FROM DBPREFIXposts"
	const expectQuery = `SELECT \* FROM posts`

	var ctx context.Context
	if tC.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), tC.timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}
	mock.ExpectBegin()
	mPrep := mock.ExpectPrepare(expectQuery).WillDelayFor(tC.delay)
	tx, err := BeginContextTx(ctx)
	if !assert.NoError(t, err) {
		return
	}

	stmt, err := PrepareContextSQL(ctx, query, tx)
	if tC.shouldFail {
		assert.Error(t, err)
		return
	}
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
}

func TestPrepareContextSQL(t *testing.T) {
	_, err := PrepareContextSQL(nil, "", nil)
	if !assert.Error(t, err) {
		return
	}

	mock := setupMockDB(t)
	testCases := []testCasePrepareContextSQL{
		{
			name: "func",
			f:    PrepareContextSQL,
		},
		{
			name: "gcdb method",
			f:    gcdb.PrepareContextSQL,
		},
		{
			name:    "delay (no fail)",
			f:       PrepareContextSQL,
			delay:   time.Second,
			timeout: 2 * time.Second,
		},
		{
			name:       "delay (times out)",
			f:          PrepareContextSQL,
			delay:      2 * time.Second,
			timeout:    time.Second,
			shouldFail: true,
		},
	}
	for c, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			tcPrepareContextSQL(t, mock, &testCases[c])
		})
	}
}
