package gcsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

type queryTestCase[T any] struct {
	name       string
	f          T
	shouldFail bool
	timeout    time.Duration
	delay      time.Duration
}

type setupSQLTestCase struct {
	name        string
	inputSQL    string
	expectedSQL string
	driver      string
	prefix      string
	expectError bool
}

var (
	setupSQLTestCases = []setupSQLTestCase{
		{
			name:        "unsupported driver",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			driver:      "unsupported",
			expectError: true,
		},
		{
			name:        "MySQL generic query, no prefix",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			expectedSQL: "SELECT * FROM posts WHERE id = ? and name = ?",
			driver:      "mysql",
		},
		{
			name:        "Postgres generic query, no prefix",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			expectedSQL: "SELECT * FROM posts WHERE id = $1 and name = $2",
			driver:      "postgres",
		},
		{
			name:        "SQLite generic query, no prefix",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			expectedSQL: "SELECT * FROM posts WHERE id = $1 and name = $2",
			driver:      "sqlite3",
		},
		{
			name:        "MySQL generic query, with prefix",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			expectedSQL: "SELECT * FROM gc_posts WHERE id = ? and name = ?",
			driver:      "mysql",
			prefix:      "gc_",
		},
		{
			name:        "Postgres generic query, with prefix",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			expectedSQL: "SELECT * FROM gc_posts WHERE id = $1 and name = $2",
			driver:      "postgres",
			prefix:      "gc_",
		},
		{
			name:        "SQLite generic query, with prefix",
			inputSQL:    "SELECT * FROM DBPREFIXposts WHERE id = ? and name = ?",
			expectedSQL: "SELECT * FROM gc_posts WHERE id = $1 and name = $2",
			driver:      "sqlite3",
			prefix:      "gc_",
		},
		{
			name:        "MySQL query with IP replacement",
			inputSQL:    "SELECT INET6_ATON(ip), INET6_NTOA(ip), INET6_ATON(some_param), INET6_NTOA(some_param) FROM DBPREFIXposts WHERE ip = INET6_ATON(?) OR ip = INET6_NTOA(?)",
			expectedSQL: "SELECT INET6_ATON(ip), INET6_NTOA(ip), INET6_ATON(some_param), INET6_NTOA(some_param) FROM gc_posts WHERE ip = INET6_ATON(?) OR ip = INET6_NTOA(?)",
			driver:      "mysql",
			prefix:      "gc_",
		},
		{
			name:        "Postgres query with IP replacement",
			inputSQL:    "SELECT INET6_ATON(ip), INET6_NTOA(ip), INET6_ATON(some_param), INET6_NTOA(some_param) FROM DBPREFIXposts WHERE ip = INET6_ATON(?) OR ip = INET6_NTOA(?)",
			expectedSQL: "SELECT ip, ip, some_param, some_param FROM gc_posts WHERE ip = $1 OR ip = $2",
			driver:      "postgres",
			prefix:      "gc_",
		},
		{
			name:        "SQLite query with IP replacement",
			inputSQL:    "SELECT INET6_ATON(ip), INET6_NTOA(ip), INET6_ATON(some_param), INET6_NTOA(some_param) FROM DBPREFIXposts WHERE ip = INET6_ATON(?) OR ip = INET6_NTOA(?)",
			expectedSQL: "SELECT INET6_ATON(ip), INET6_NTOA(ip), INET6_ATON(some_param), INET6_NTOA(some_param) FROM gc_posts WHERE ip = INET6_ATON($1) OR ip = INET6_NTOA($2)",
			driver:      "sqlite3",
			prefix:      "gc_",
		},
	}
)

type prepareFunc func(context.Context, string, *sql.Tx) (*sql.Stmt, error)

func tcPrepareContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *queryTestCase[prepareFunc]) {
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

	mock := SetupMockDB(t, "mysql")
	testCases := []queryTestCase[prepareFunc]{
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

func tcExecContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *queryTestCase[execFunc]) {
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

	mock := SetupMockDB(t, "mysql")
	testCases := []queryTestCase[execFunc]{
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

func tcQueryRowContextSQL(t *testing.T, mock sqlmock.Sqlmock, tC *queryTestCase[funcQueryRowContextSQL]) {
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

	mock := SetupMockDB(t, "mysql")
	testCases := []queryTestCase[funcQueryRowContextSQL]{
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

func TestSetupSQLString(t *testing.T) {
	config.InitTestConfig()
	t.Run("not connected", func(t *testing.T) {
		_, err := SetupSQLString("test", nil)
		assert.Error(t, err)
	})
	for _, tC := range setupSQLTestCases {
		t.Run(tC.name, func(t *testing.T) {
			systemCriticalCfg := config.GetSystemCriticalConfig()
			systemCriticalCfg.DBprefix = tC.prefix
			config.SetSystemCriticalConfig(systemCriticalCfg)

			db := &GCDB{
				driver: tC.driver,
			}
			replacerArr := []string{
				"DBNAME", "gochan",
				"DBPREFIX", tC.prefix,
				"DBVERSION", strconv.Itoa(DatabaseVersion),
				"\n", " ",
			}

			switch tC.driver {
			case "mysql":
				replacerArr = append(replacerArr, mysqlReplacerArr...)
			case "postgres":
				replacerArr = append(replacerArr, postgresReplacerArr...)
			case "sqlite3":
				replacerArr = append(replacerArr, sqlite3ReplacerArr...)
			}
			db.replacer = strings.NewReplacer(replacerArr...)

			prepared, err := SetupSQLString(tC.inputSQL, db)
			if tC.expectError {
				assert.Error(t, err)
			} else {
				if !assert.NoError(t, err, tC.name) {
					t.FailNow()
				}
				assert.Equal(t, tC.expectedSQL, prepared, tC.name)
			}
		})
	}
}
