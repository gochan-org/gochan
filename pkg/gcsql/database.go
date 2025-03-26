package gcsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// GochanVersionKeyConstant is the key value used in the version table of the database to store and receive the (database) version of base gochan
	gochanVersionKeyConstant = "gochan"
	UnsupportedSQLVersionMsg = `syntax error in SQL query, confirm you are using a supported driver and SQL server (error text: %s)`
	mysqlConnStr             = "%s:%s@tcp(%s)/%s?parseTime=true&collation=utf8mb4_unicode_ci"
	postgresConnStr          = "postgres://%s:%s@%s/%s?sslmode=disable"
	sqlite3ConnStr           = "file:%s?_auth&_auth_user=%s&_auth_pass=%s&_auth_crypt=sha1"
)

var (
	gcdb             *GCDB
	mysqlReplacerArr = []string{
		"RANGE_START_ATON", "INET6_ATON(range_start)",
		"RANGE_START_NTOA", "INET6_NTOA(range_start)",
		"RANGE_END_ATON", "INET6_ATON(range_end)",
		"RANGE_END_NTOA", "INET6_NTOA(range_end)",
		"IP_ATON", "INET6_ATON(ip)",
		"IP_NTOA", "INET6_NTOA(ip)",
		"PARAM_ATON", "INET6_ATON(?)",
		"PARAM_NTOA", "INET6_NTOA(?)",
	}
	postgresReplacerArr = []string{
		"RANGE_START_ATON", "range_start",
		"RANGE_START_NTOA", "range_start",
		"RANGE_END_ATON", "range_end",
		"RANGE_END_NTOA", "range_end",
		"IP_ATON", "ip",
		"IP_NTOA", "ip",
		"PARAM_ATON", "?",
		"PARAM_NTOA", "?",
	}
	sqlite3ReplacerArr = []string{
		"RANGE_START_ATON", "range_start",
		"RANGE_START_NTOA", "range_start",
		"RANGE_END_ATON", "range_end",
		"RANGE_END_NTOA", "range_end",
		"IP_ATON", "ip",
		"IP_NTOA", "ip",
		"PARAM_ATON", "?",
		"PARAM_NTOA", "?",
	}
)

type GCDB struct {
	db             *sql.DB
	connStr        string
	driver         string
	defaultTimeout time.Duration
	replacer       *strings.Replacer
}

func (db *GCDB) ConnectionString() string {
	return db.connStr
}

func (db *GCDB) Connection() *sql.DB {
	return db.db
}

func (db *GCDB) SQLDriver() string {
	return db.driver
}

func (db *GCDB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *GCDB) GetBaseDB() *sql.DB {
	return db.db
}

func (db *GCDB) PrepareSQL(query string, tx *sql.Tx) (*sql.Stmt, error) {
	return db.PrepareContextSQL(context.Background(), query, tx)
}

func (db *GCDB) PrepareContextSQL(ctx context.Context, query string, tx *sql.Tx) (*sql.Stmt, error) {
	var prepared string
	var err error
	if prepared, err = SetupSQLString(db.replacer.Replace(query), db); err != nil {
		return nil, err
	}
	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), db.defaultTimeout)
		defer cancel()
	}
	var stmt *sql.Stmt
	if tx != nil {
		stmt, err = tx.PrepareContext(ctx, prepared)
	} else {
		stmt, err = db.db.PrepareContext(ctx, prepared)
	}
	return stmt, err
}

// Exec executes the given SQL statement with the given parameters, optionally with the given RequestOptions struct
// or a background context and transaction if nil
func (db *GCDB) Exec(opts *RequestOptions, query string, values ...any) (sql.Result, error) {
	opts = setupOptions(opts)
	stmt, err := db.PrepareContextSQL(opts.Context, query, opts.Tx)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	result, err := stmt.ExecContext(opts.Context, values...)
	if err != nil {
		return nil, err
	}
	return result, stmt.Close()
}

/*
ExecSQL executes the given SQL statement with the given parameters.
Deprecated: Use Exec instead

Example:

	var intVal int
	var stringVal string
	result, err := db.ExecSQL("INSERT INTO tablename (intval,stringval) VALUES(?,?)", intVal, stringVal)
*/
func (db *GCDB) ExecSQL(query string, values ...any) (sql.Result, error) {
	return db.Exec(nil, query, values...)
}

/*
ExecContextSQL executes the given SQL statement with the given context, optionally with the given transaction (if non-nil).
Deprecated: Use Exec instead, with a RequestOptions struct for the context and transaction

Example:

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	var intVal int
	var stringVal string
	result, err := db.ExecContextSQL(ctx, nil, "INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func (db *GCDB) ExecContextSQL(ctx context.Context, tx *sql.Tx, sqlStr string, values ...any) (sql.Result, error) {
	return db.Exec(&RequestOptions{Context: ctx, Tx: tx}, sqlStr, values...)
}

/*
ExecTxSQL executes the given SQL statemtnt, optionally with the given transaction (if non-nil).
Deprecated: Use Exec instead, with a RequestOptions struct for the transaction

Example:

	tx, err := BeginTx()
	// do error handling stuff
	defer tx.Rollback()
	var intVal int
	var stringVal string
	result, err := db.ExecTxSQL(tx, "INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func (db *GCDB) ExecTxSQL(tx *sql.Tx, query string, values ...any) (sql.Result, error) {
	return db.Exec(&RequestOptions{Tx: tx}, query, values...)
}

/*
Begin creates and returns a new SQL transaction using the GCDB. Note that it doesn't use gochan's
database variables, e.g. DBPREFIX, DBNAME, etc so it should be used sparingly or with
gcsql.SetupSQLString
*/
func (db *GCDB) Begin() (*sql.Tx, error) {
	return db.db.Begin()
}

/*
BeginTx creates and returns a new SQL transaction using the GCDB with the specified context
and transaction options. Note that it doesn't use gochan's database variables, e.g. DBPREFIX,
DBNAME, etc so it should be used sparingly or with gcsql.SetupSQLString
*/
func (db *GCDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.db.BeginTx(ctx, opts)
}

// QueryRow gets a row from the db with the values in values[] and fills the respective pointers in out[],
// with an optional RequestOptions struct for the context and transaction
func (db *GCDB) QueryRow(opts *RequestOptions, query string, values []any, out []any) error {
	opts = setupOptions(opts)
	stmt, err := db.PrepareContextSQL(opts.Context, query, opts.Tx)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if err = stmt.QueryRowContext(opts.Context, values...).Scan(out...); err != nil {
		return err
	}
	return stmt.Close()
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[].
Deprecated: Use QueryRow instead

Example:

	id := 32
	var intVal int
	var stringVal string
	err := db.QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]any{id},
		[]any{&intVal, &stringVal})
*/
func (db *GCDB) QueryRowSQL(query string, values, out []any) error {
	return db.QueryRow(nil, query, values, out)
}

/*
QueryRowContextSQL gets a row from the database with the values in values[] and fills the respective pointers in out[]
using the given context as a deadline, and the given transaction (if non-nil)

Example:

	id := 32
	var name string
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	err := db.QueryRowContextSQL(ctx, nil, "SELECT name FROM DBPREFIXposts WHERE id = ? LIMIT 1",
		[]any{id}, []any{&name})
*/
func (db *GCDB) QueryRowContextSQL(ctx context.Context, tx *sql.Tx, query string, values, out []any) error {
	return db.QueryRow(&RequestOptions{Context: ctx, Tx: tx}, query, values, out)
}

/*
QueryRowTxSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query

Example:

	id := 32
	var intVal int
	var stringVal string
	tx, err := BeginTx()
	// do error handling stuff
	defer tx.Rollback()
	err = QueryRowTxSQL(tx, "SELECT intval,stringval FROM table WHERE id = ?",
		[]any{id},
		[]any{&intVal, &stringVal})
*/
func (db *GCDB) QueryRowTxSQL(tx *sql.Tx, query string, values, out []any) error {
	return db.QueryRow(&RequestOptions{Tx: tx}, query, values, out)
}

// Query sends the query to the database with the given options (or a background context if nil), and the given parameters
func (db *GCDB) Query(opts *RequestOptions, query string, a ...any) (*sql.Rows, error) {
	opts = setupOptions(opts)
	stmt, err := db.PrepareContextSQL(opts.Context, query, opts.Tx)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(opts.Context, a...)
	if err != nil {
		return rows, err
	}

	return rows, stmt.Close()
}

/*
QuerySQL gets all rows from the db with the values in values[] and fills the respective pointers in out[].
Deprecated: Use Query instead
Example:

	rows, err := db.QuerySQL("SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func (db *GCDB) QuerySQL(query string, a ...any) (*sql.Rows, error) {
	return db.Query(nil, query, a...)
}

/*
QueryContextSQL queries the database with a prepared statement and the given parameters, using the given context
for a deadline.
Deprecated: Use Query instead, with a RequestOptions struct for the context and transaction

Example:

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	rows, err := db.QueryContextSQL(ctx, nil, "SELECT name from posts where NOT is_deleted")
*/
func (db *GCDB) QueryContextSQL(ctx context.Context, tx *sql.Tx, query string, a ...any) (*sql.Rows, error) {
	return db.Query(&RequestOptions{Context: ctx, Tx: tx}, query, a...)
}

/*
QueryTxSQL gets all rows from the db with the values in values[] and fills the respective pointers in out[].
Deprecated: Use Query instead, with a RequestOptions struct for the transaction
Example:

	tx, _ := db.Begin()
	rows, err := db.QueryTxSQL(tx, "SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func (db *GCDB) QueryTxSQL(tx *sql.Tx, query string, a ...any) (*sql.Rows, error) {
	return db.Query(&RequestOptions{Tx: tx}, query, a...)
}

func setupDBConn(cfg *config.SQLConfig) (db *GCDB, err error) {
	db = &GCDB{
		driver:         cfg.DBtype,
		defaultTimeout: time.Duration(cfg.DBTimeoutSeconds) * time.Second,
	}
	replacerArr := []string{
		"DBNAME", cfg.DBname,
		"DBPREFIX", cfg.DBprefix,
		"\n", " ",
	}
	switch cfg.DBtype {
	case "mysql":
		db.connStr = fmt.Sprintf(mysqlConnStr, cfg.DBusername, cfg.DBpassword, cfg.DBhost, cfg.DBname)
		replacerArr = append(replacerArr, mysqlReplacerArr...)
	case "postgres":
		db.connStr = fmt.Sprintf(postgresConnStr, cfg.DBusername, cfg.DBpassword, cfg.DBhost, cfg.DBname)
		replacerArr = append(replacerArr, postgresReplacerArr...)
	case "sqlite3":
		db.connStr = fmt.Sprintf(sqlite3ConnStr, cfg.DBhost, cfg.DBusername, cfg.DBpassword)
		replacerArr = append(replacerArr, sqlite3ReplacerArr...)
	default:
		return nil, ErrUnsupportedDB
	}
	db.replacer = strings.NewReplacer(replacerArr...)
	return db, nil
}

func setupSqlTestConfig(dbDriver string, dbName string, dbPrefix string) *config.SQLConfig {
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

// SetupMockDB sets up a mock database connection for testing
func SetupMockDB(driver string) (sqlmock.Sqlmock, error) {
	var err error
	gcdb, err = setupDBConn(setupSqlTestConfig(driver, "gochan", ""))
	if err != nil {
		return nil, err
	}
	var mock sqlmock.Sqlmock
	gcdb.db, mock, err = sqlmock.New()

	return mock, err
}

// Open opens and returns a new gochan database connection with the provided SQL options
func Open(cfg *config.SQLConfig) (db *GCDB, err error) {
	db, err = setupDBConn(cfg)
	if err != nil {
		return nil, err
	}
	db.db, err = sql.Open(db.driver, db.connStr)
	if err != nil {
		db.db.SetConnMaxLifetime(time.Minute * time.Duration(cfg.DBConnMaxLifetimeMin))
		db.db.SetMaxOpenConns(cfg.DBMaxOpenConnections)
		db.db.SetMaxIdleConns(cfg.DBMaxIdleConnections)
	}
	return db, err
}

func optimizeMySQL() error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	var wg sync.WaitGroup
	rows, err := QueryContextSQL(ctx, nil, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE()")
	if err != nil {
		return err
	}

	for rows.Next() {
		wg.Add(1)
		var table string
		if err = rows.Scan(&table); err != nil {
			rows.Close()
			return err
		}
		go func(table string) {
			if _, err = ExecContextSQL(ctx, nil, "OPTIMIZE TABLE "+table); err != nil {
				rows.Close()
				return
			}
			wg.Done()
		}(table)
	}
	wg.Wait()
	if err != nil {
		return err
	}
	return rows.Close()
}

func optimizePostgres() error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	_, err := ExecContextSQL(ctx, nil, "REINDEX DATABASE "+config.GetSQLConfig().DBname)
	return err
}

func optimizeSqlite3() error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	_, err := ExecContextSQL(ctx, nil, "VACUUM")
	return err
}

// OptimizeDatabase peforms a database optimisation
func OptimizeDatabase() error {
	switch config.GetSQLConfig().DBtype {
	case "mysql":
		return optimizeMySQL()
	case "postgresql":
		return optimizePostgres()
	case "sqlite3":
		return optimizeSqlite3()
	default:
		// this shouldn't happen under normal circumstances since this is assumed to have already been checked
		return ErrUnsupportedDB
	}
}
