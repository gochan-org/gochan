package main

import (
	"bytes"
	"io"
	"fmt"
	"database/sql"
	"database/sql/driver"
	_ "github.com/ziutek/mymysql/godrv"
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
	var err error
 	if needs_initial_setup {
		db, err = sql.Open("mymysql", config.DBhost+"*mysql/"+config.DBusername+"/"+config.DBpassword)
		if err != nil {
			fmt.Println("Failed to connect to the database, see log for details.")
			error_log.Fatal(err.Error())
		}
	} else {
		db, err = sql.Open("mymysql", config.DBhost+"*"+config.DBname+"/"+config.DBusername+"/"+config.DBpassword)
		if err != nil {
			fmt.Println("Failed to connect to the database, see log for details.")
			error_log.Fatal(err.Error())
		}
	}
	_, err = db.Exec("USE `mysql`")
	if err == driver.ErrBadConn {
		fmt.Println("Error: failed connecting to the database.")
		error_log.Fatal(err.Error())
	} else {
		db.Exec("USE `" + config.DBname + "`")
	}
	db_connected = true
}
