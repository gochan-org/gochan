package gcsql

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

var (
	insertIntoThreadsBase     = `INSERT INTO threads \(board_id, locked, stickied, anchored, cyclical, is_spoilered\) VALUES `
	insertIntoThreadsMySQL    = insertIntoThreadsBase + `\(\?,\?,\?,\?,\?,\?\)`
	insertIntoThreadsPostgres = insertIntoThreadsBase + `\(\$1,\$2,\$3,\$4,\$5,\$6\)`

	insertIntoPostsBase = `INSERT INTO posts\s*` +
		`\(thread_id, is_top_post, ip, created_on, name, tripcode, is_secure_tripcode, is_role_signature, email, subject,\s+` +
		`message, message_raw, password, flag, country\)\s+VALUES`
	insertIntoPostsMySQL    = insertIntoPostsBase + `\(\?,\?,INET6_ATON\(\?\),CURRENT_TIMESTAMP,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?\)`
	insertIntoPostsPostgres = insertIntoPostsBase + `\(\$1,\$2,\$3,CURRENT_TIMESTAMP,\$4,\$5,\$6,\$7,\$8,\$9,\$10,\$11,\$12,\$13,\$14\)`
)

func setupPostTest(t *testing.T, driver string) sqlmock.Sqlmock {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	config.InitTestConfig()
	config.SetTestDBConfig(driver, "localhost", "gochan", "gochan", "gochan", "")

	gcdb, err := setupDBConn(setupSqlTestConfig(driver, "gochan", ""))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	var mock sqlmock.Sqlmock
	gcdb.db, mock, err = sqlmock.New()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, SetTestingDB(driver, "gochan", "", gcdb.db)) {
		t.FailNow()
	}
	if !assert.NoError(t, setupAndProvisionMockDB(t, mock, driver, "gochan")) {
		t.FailNow()
	}
	return mock
}

func createThreadTestRun(t *testing.T, driver string) {
	mock := setupPostTest(t, driver)
	var query string
	if driver == "mysql" {
		query = `SELECT locked FROM boards WHERE id = \?`
	} else {
		query = `SELECT locked FROM boards WHERE id = \$1`
	}
	mock.ExpectPrepare(query).ExpectQuery().
		WithArgs(1).WillReturnRows(mock.NewRows([]string{"locked"}).AddRow(false))

	if driver == "mysql" {
		query = insertIntoThreadsMySQL
	} else {
		query = insertIntoThreadsPostgres
	}
	mock.ExpectPrepare(query).
		ExpectExec().WithArgs(1, false, false, false, false, false).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectPrepare(`SELECT MAX\(id\) FROM threads`).ExpectQuery().
		WillReturnRows(mock.NewRows([]string{"MAX(id)"}).AddRow(1))

	mock.ExpectBegin()
	if driver == "mysql" {
		query = `SELECT locked FROM threads WHERE id = \?`
	} else {
		query = `SELECT locked FROM threads WHERE id = \$1`
	}
	mock.ExpectPrepare(query).ExpectQuery().
		WithArgs(1).WillReturnRows(mock.NewRows([]string{"locked"}).AddRow(false))

	thread := &Thread{BoardID: 1}
	if !assert.NoError(t, CreateThread(nil, thread)) {
		t.FailNow()
	}
	p := Post{ThreadID: thread.ID, Message: "test", MessageRaw: "test", IP: "192.168.56.1", IsTopPost: true, CreatedOn: time.Now()}

	if driver == "mysql" {
		query = insertIntoPostsMySQL
	} else {
		query = insertIntoPostsPostgres
	}
	mock.ExpectPrepare(query).ExpectExec().
		WithArgs(p.ThreadID, p.IsTopPost, p.IP, p.Name, p.Tripcode, p.IsSecureTripcode, p.IsRoleSignature, p.Email,
			p.Subject, p.Message, p.MessageRaw, p.Password, p.Flag, p.Country).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectPrepare(`SELECT MAX\(id\) FROM posts`).ExpectQuery().WithoutArgs().
		WillReturnRows(mock.NewRows([]string{"MAX(id)"}).AddRow(1))
	if driver == "mysql" {
		query = `UPDATE threads SET last_bump = CURRENT_TIMESTAMP WHERE id = \?`
	} else {
		query = `UPDATE threads SET last_bump = CURRENT_TIMESTAMP WHERE id = \$1`
	}
	mock.ExpectPrepare(query).ExpectExec().
		WithArgs(thread.ID).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if !assert.NoError(t, p.Insert(true, thread, false)) {
		t.FailNow()
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateThread(t *testing.T) {
	for _, driver := range []string{"mysql", "postgres", "sqlite3"} {
		t.Run(driver, func(t *testing.T) {
			createThreadTestRun(t, driver)
		})
	}
}

func TestWebPath(t *testing.T) {
	var query string
	for _, driver := range []string{"mysql", "postgres", "sqlite3"} {
		post := Post{ID: 1, IsTopPost: true}
		t.Run(driver, func(t *testing.T) {
			mock := setupPostTest(t, driver)
			if driver == "mysql" {
				query = `SELECT op_id, dir FROM v_top_post_board_dir WHERE id = \?`
			} else {
				query = `SELECT op_id, dir FROM v_top_post_board_dir WHERE id = \$1`
			}
			mock.ExpectPrepare(query).ExpectQuery().WithArgs(1).
				WillReturnRows(mock.NewRows([]string{"op_id", "dir"}).AddRow(1, "test"))
			assert.Equal(t, "/test/res/1.html#1", post.WebPath())
			assert.Equal(t, 1, post.opID)
			assert.Equal(t, "test", post.boardDir)
			assert.Equal(t, "/test/res/1.html#1", post.WebPath())
			assert.NoError(t, mock.ExpectationsWereMet())

			post.opID = 0
			post.boardDir = ""
			mock.ExpectPrepare(query).ExpectQuery().WithArgs(1).
				WillReturnRows(mock.NewRows([]string{"op_id", "dir"}).AddRow(1, "test")).
				WillReturnError(ErrBoardDoesNotExist)
			assert.Equal(t, "/", post.WebPath())
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
