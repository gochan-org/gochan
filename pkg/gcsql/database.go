package gcsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

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

func (db *GCDB) PrepareSQL(query string, tx *sql.Tx) (*sql.Stmt, error) {
	return db.PrepareContextSQL(context.Background(), query, tx)
}

func (db *GCDB) PrepareContextSQL(ctx context.Context, query string, tx *sql.Tx) (*sql.Stmt, error) {
	var prepared string
	var err error
	if prepared, err = SetupSQLString(db.replacer.Replace(query), db); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
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
	if err != nil {
		return stmt, err
	}
	return stmt, sqlVersionError(err, db.driver, &prepared)
}

/*
ExecSQL executes the given SQL statement with the given parameters
Example:

	var intVal int
	var stringVal string
	result, err := db.ExecSQL("INSERT INTO tablename (intval,stringval) VALUES(?,?)", intVal, stringVal)
*/
func (db *GCDB) ExecSQL(query string, values ...any) (sql.Result, error) {
	stmt, err := db.PrepareSQL(query, nil)
	if err != nil {
		return nil, err
	}
	result, err := stmt.Exec(values...)
	if err != nil {
		stmt.Close()
		return nil, err
	}
	return result, stmt.Close()
}

/*
ExecContextSQL executes the given SQL statement with the given context, optionally with the given transaction (if non-nil)

Example:

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	var intVal int
	var stringVal string
	result, err := db.ExecContextSQL(ctx, nil, "INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func (db *GCDB) ExecContextSQL(ctx context.Context, tx *sql.Tx, sqlStr string, values ...any) (sql.Result, error) {
	stmt, err := db.PrepareContextSQL(ctx, sqlStr, tx)
	if err != nil {
		return nil, err
	}
	result, err := stmt.ExecContext(ctx, values...)
	if err != nil {
		stmt.Close()
		return nil, err
	}
	return result, stmt.Close()
}

/*
ExecTxSQL executes the given SQL statemtnt, optionally with the given transaction (if non-nil)

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
	stmt, err := db.PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(values...)
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

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:

	id := 32
	var intVal int
	var stringVal string
	err := db.QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]any{id},
		[]any{&intVal, &stringVal})
*/
func (db *GCDB) QueryRowSQL(query string, values, out []any) error {
	stmt, err := db.PrepareSQL(query, nil)
	if err != nil {
		return err
	}
	defer stmt.Close()
	return stmt.QueryRow(values...).Scan(out...)
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
	stmt, err := db.PrepareContextSQL(ctx, query, tx)
	if err != nil {
		return err
	}
	if err = stmt.QueryRowContext(ctx, values...).Scan(out...); err != nil {
		stmt.Close()
		return err
	}
	return stmt.Close()
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
	stmt, err := db.PrepareSQL(query, tx)
	if err != nil {
		return err
	}
	if err = stmt.QueryRow(values...).Scan(out...); err != nil {
		stmt.Close()
		return err
	}
	return stmt.Close()
}

/*
QuerySQL gets all rows from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
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
	stmt, err := db.PrepareSQL(query, nil)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Query(a...)
}

/*
QueryContextSQL queries the database with a prepared statement and the given parameters, using the given context
for a deadline

Example:

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	rows, err := db.QueryContextSQL(ctx, nil, "SELECT name from posts where NOT is_deleted")
*/
func (db *GCDB) QueryContextSQL(ctx context.Context, tx *sql.Tx, query string, a ...any) (*sql.Rows, error) {
	stmt, err := db.PrepareContextSQL(ctx, query, tx)
	if err != nil {
		return nil, err
	}
	return stmt.QueryContext(ctx, a...)
}

/*
QueryTxSQL gets all rows from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
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
	stmt, err := db.PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Query(a...)
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
		addrMatches := tcpHostIsolator.FindAllStringSubmatch(cfg.DBhost, -1)
		if len(addrMatches) > 0 && len(addrMatches[0]) > 2 {
			cfg.DBhost = addrMatches[0][2]
		}
		db.connStr = fmt.Sprintf(sqlite3ConnStr, cfg.DBhost, cfg.DBusername, cfg.DBpassword)
		replacerArr = append(replacerArr, sqlite3ReplacerArr...)
	default:
		return nil, ErrUnsupportedDB
	}
	db.replacer = strings.NewReplacer(replacerArr...)
	return db, nil
}

// Open opens and returns a new gochan database connection with the provided host, driver, DB name,
// username, password, and table prefix
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
	timeout := config.GetSQLConfig().DBTimeoutSeconds
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
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
	cfg := config.GetSQLConfig()
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Second*time.Duration(cfg.DBTimeoutSeconds))
	defer cancel()

	_, err := ExecContextSQL(ctx, nil, "REINDEX DATABASE "+cfg.DBname)
	return err
}

func optimizeSqlite3() error {
	timeout := time.Duration(config.GetSQLConfig().DBTimeoutSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
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

func sqlVersionError(err error, dbDriver string, query *string) error {
	if err == nil {
		return nil
	}
	errText := err.Error()
	switch dbDriver {
	case "mysql":
		if !strings.Contains(errText, "You have an error in your SQL syntax") {
			return err
		}
	case "postgres":
		if !strings.Contains(errText, "syntax error at or near") {
			return err
		}
	}
	if config.GetSystemCriticalConfig().Verbose {
		return fmt.Errorf(UnsupportedSQLVersionMsg+"\nQuery: "+*query, errText)
	}
	return fmt.Errorf(UnsupportedSQLVersionMsg, errText)
}
