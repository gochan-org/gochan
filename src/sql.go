package main

import (
	"bytes"
	"database/sql"
	"io"
	"io/ioutil"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	nil_timestamp         = "0000-00-00 00:00:00"
	mysql_datetime_format = "2006-01-02 15:04:05"
)

var (
	db           *sql.DB
	db_connected = false
)

// escapeString and escapeQuotes copied from github.com/ziutek/mymysql/native/codecs.go
func escapeString(txt string) string {
	var (
		esc string
		buf bytes.Buffer
	)
	last := 0
	for ii, bb := range txt {
		switch bb {
		case 0:
			esc = `\0`
		case '\n':
			esc = `\n`
		case '\r':
			esc = `\r`
		case '\\':
			esc = `\\`
		case '\'':
			esc = `\'`
		case '"':
			esc = `\"`
		case '\032':
			esc = `\Z`
		default:
			continue
		}
		io.WriteString(&buf, txt[last:ii])
		io.WriteString(&buf, esc)
		last = ii + 1
	}
	io.WriteString(&buf, txt[last:])
	return buf.String()
}

func escapeQuotes(txt string) string {
	var buf bytes.Buffer
	last := 0
	for ii, bb := range txt {
		if bb == '\'' {
			io.WriteString(&buf, txt[last:ii])
			io.WriteString(&buf, `''`)
			last = ii + 1
		}
	}
	io.WriteString(&buf, txt[last:])
	return buf.String()
}

func connectToSQLServer() {
	var err error

	db, err = sql.Open("mysql", config.DBusername+":"+config.DBpassword+"@"+config.DBhost+"/"+config.DBname+"?parseTime=true&collation=utf8mb4_unicode_ci")
	if err != nil {
		println(0, "Failed to connect to the database, see log for details.")
		error_log.Fatal(err.Error())
	}

	// get the number of tables in the database. If the number > 1, we can assume that initial setup has already been run
	var num_rows int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '" + config.DBname + "';").Scan(&num_rows)
	if err == sql.ErrNoRows {
		num_rows = 0
	} else if err != nil {
		println(0, "Failed retrieving list of tables in database.")
		error_log.Fatal(err.Error())
	}
	// Detect that there are at least the number of tables that we are setting up.
	// If there are fewer than that, then we either half-way set up, or there's other tables in our database.
	if num_rows >= 16 {
		// the initial setup has already been run
		needs_initial_setup = false
		db_connected = true
		println(0, "complete.")
		return
	} else {
		// does the  initialsetupdb.sql exist?
		_, err := os.Stat("initialsetupdb.sql")
		if err != nil {
			println(0, "Initial setup file (initialsetupdb.sql) missing. Please reinstall gochan")
			error_log.Fatal("Initial setup file (initialsetupdb.sql) missing. Please reinstall gochan")
		}

		// read the initial setup sql file into a string
		initial_sql_bytes, err := ioutil.ReadFile("initialsetupdb.sql")
		if err != nil {
			println(0, "failed, see log for details.")
			error_log.Fatal(err.Error())
		}
		initial_sql_str := string(initial_sql_bytes)
		initial_sql_bytes = nil
		printf(0, "Starting initial setup...")
		initial_sql_str = strings.Replace(initial_sql_str, "DBNAME", config.DBname, -1)
		initial_sql_str = strings.Replace(initial_sql_str, "DBPREFIX", config.DBprefix, -1)
		initial_sql_str += "\nINSERT INTO `" + config.DBname + "`.`" + config.DBprefix + "staff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '" + bcryptSum("password") + "', 'abc', 3);"
		initial_sql_arr := strings.Split(initial_sql_str, ";")
		initial_sql_str = ""

		for _, statement := range initial_sql_arr {
			if statement != "" {
				_, err := db.Exec(statement)
				if err != nil {
					println(0, "failed, see log for details.")
					error_log.Fatal(err.Error())
					return
				}
			}
		}
		println(0, "complete.")
		needs_initial_setup = false
		db_connected = true
	}
}
