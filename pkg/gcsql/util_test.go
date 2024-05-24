package gcsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

type testCase[T any] struct {
	name       string
	f          T
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

type prepareFunc func(context.Context, string, *sql.Tx) (*sql.Stmt, error)

func tcPrepareContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *testCase[prepareFunc]) {
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

	stmt, err := tC.f(ctx, query, tx)
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
	stmt, err := PrepareContextSQL(context.Background(), "", nil)
	if !assert.Error(t, err) {
		return
	}
	defer func() {
		if stmt != nil {
			assert.NoError(t, stmt.Close())
		}
	}()

	mock := setupMockDB(t)
	testCases := []testCase[prepareFunc]{
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

type execFunc func(context.Context, *sql.Tx, string, ...any) (sql.Result, error)

func tcExecContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *testCase[execFunc]) {
	const query = "INSERT INTO DBPREFIXposts (name) VALUES(?)"
	const expectQuery = `INSERT INTO posts \(name\) VALUES\(\?\)`

	var ctx context.Context
	if tC.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), tC.timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}
	mock.ExpectBegin()
	mock.ExpectPrepare(expectQuery).
		WillDelayFor(tC.delay).
		ExpectExec().WithArgs("Test").
		WillReturnResult(driver.ResultNoRows)

	tx, err := BeginContextTx(ctx)
	if !assert.NoError(t, err) {
		return
	}

	_, err = tC.f(ctx, tx, query, "Test")
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
	assert.NoError(t, tx.Commit())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecContextSQL(t *testing.T) {
	_, err := ExecContextSQL(context.Background(), nil, "")
	if !assert.Error(t, err) {
		return
	}

	mock := setupMockDB(t)
	testCases := []testCase[execFunc]{
		{
			name: "func",
			f:    ExecContextSQL,
		},
		{
			name: "gcdb method",
			f:    gcdb.ExecContextSQL,
		},
		{
			name:    "delay (no fail)",
			f:       ExecContextSQL,
			delay:   time.Second,
			timeout: 2 * time.Second,
		},
		{
			name:       "delay (times out)",
			f:          ExecContextSQL,
			delay:      2 * time.Second,
			timeout:    time.Second,
			shouldFail: true,
		},
	}
	for c, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			tcExecContextSQL(t, mock, &testCases[c])
		})
	}
}

type funcQueryRowContextSQL func(context.Context, *sql.Tx, string, []any, []any) error

func tcQueryRowContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *testCase[funcQueryRowContextSQL]) {
	const query = "SELECT NAME FROM DBPREFIXposts WHERE id = ?"
	const expectQuery = `SELECT NAME FROM posts WHERE id = \?`

	var ctx context.Context
	if tC.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), tC.timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}
	mock.ExpectBegin()
	mock.ExpectPrepare(expectQuery).
		ExpectQuery().WithArgs(1).
		WillDelayFor(tC.delay).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Name"))

	tx, err := BeginContextTx(ctx)
	if !assert.NoError(t, err) {
		return
	}

	var out string
	err = tC.f(ctx, tx, query, []any{1}, []any{&out})
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
	assert.NoError(t, tx.Commit())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryRowContextSQL(t *testing.T) {
	err := QueryRowContextSQL(context.Background(), nil, "", nil, nil)
	if !assert.Error(t, err) {
		return
	}

	mock := setupMockDB(t)
	testCases := []testCase[funcQueryRowContextSQL]{
		{
			name: "func",
			f:    QueryRowContextSQL,
		},
		{
			name: "gcdb method",
			f:    gcdb.QueryRowContextSQL,
		},
		{
			name:    "delay (no fail)",
			f:       QueryRowContextSQL,
			delay:   time.Second,
			timeout: 2 * time.Second,
		},
		{
			name:       "delay (times out)",
			f:          QueryRowContextSQL,
			delay:      2 * time.Second,
			timeout:    time.Second,
			shouldFail: true,
		},
	}
	for c, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			tcQueryRowContextSQL(t, mock, &testCases[c])
		})
	}
}
