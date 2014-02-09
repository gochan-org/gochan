package main

import (
	"bytes"
	"io"
	"fmt"
	"database/sql"
	_ "github.com/ziutek/mymysql/godrv"
	"io/ioutil"
	"os"
	"strings"
)

const (
	nil_timestamp = "0000-00-00 00:00:00"
	mysql_datetime_format = "2006-01-02 15:04:05"
)

var (
	db *sql.DB
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
	// does the original initialsetupdb.sql (as opposed to .bak.sql) exist?
	var err error
	_, err1 := os.Stat("initialsetupdb.sql")
	_, err2 := os.Stat("initialsetupdb.bak.sql")
	if err2 == nil {
		// the .bak.sql file exists
		os.Remove("initialsetupdb.sql")
		fmt.Println("complete.")
		needs_initial_setup = false
		return
	} else {
		if err1 != nil {
			// neither one exists
			fmt.Println("failed...initial setup file doesn't exist, please reinstall gochan.")
			error_log.Fatal("Initial setup file doesn't exist, exiting.")
			return
		}
	}
	err1 = nil
	err2 = nil

	db, err = sql.Open("mymysql", config.DBhost + "*" + config.DBname + "/"+config.DBusername+"/"+config.DBpassword)
	if err != nil {
		fmt.Println("Failed to connect to the database, see log for details.")
		error_log.Fatal(err.Error())
	}
	// read the initial setup sql file into a string
	initial_sql_bytes,err := ioutil.ReadFile("initialsetupdb.sql")
	if err != nil {
		fmt.Println("failed.")
		error_log.Fatal(err.Error())
	}
	initial_sql_str := string(initial_sql_bytes)
	initial_sql_bytes = nil
	fmt.Printf("Starting initial setup...")
	initial_sql_str = strings.Replace(initial_sql_str,"DBNAME",config.DBname, -1)
	initial_sql_str = strings.Replace(initial_sql_str,"DBPREFIX",config.DBprefix, -1)
	initial_sql_str += "\nINSERT INTO `"+config.DBname+"`.`"+config.DBprefix+"staff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '"+bcrypt_sum("password")+"', 'abc', 3);"
	initial_sql_arr := strings.Split(initial_sql_str, ";")
	initial_sql_str = ""

	for _,statement := range initial_sql_arr {
		if statement != "" {
			_,err := db.Exec(statement+";")
			if err != nil {
				fmt.Println("failed.")
				error_log.Fatal(err.Error())
				return
			} 
		}
	}
	initial_sql_arr = nil
	// rename initialsetupdb.sql to initialsetup.bak.sql
	err = ioutil.WriteFile("initialsetupdb.bak.sql", initial_sql_bytes, 0777)
	if err != nil {
		fmt.Println("failed")
		error_log.Fatal(err.Error())
		return
	}

	err = os.Remove("initialsetupdb.bak.sql")
	if err != nil {
		fmt.Println("failed.")
		error_log.Fatal(err.Error())
		return
	}
	fmt.Println("complete.")

	needs_initial_setup = false
	db_connected = true
}