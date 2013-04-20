package main

import (
	"os"
	"fmt"
	//"database/sql"
	//_ "github.com/go-sql-driver/mysql"

	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native" // Native engine
	//_ "github.com/ziutek/mymysql/thrsafe" // Thread safe engine
)

var (
	//db *sql.DB
	db mysql.Conn
	db_connected = false
)

func connectToSQLServer(usedb bool) {
	//db, err = sql.Open("mysql",config.DBusername+":"+config.DBpassword+"@"+db_host+"/?charset=utf8")
	db = mysql.New("tcp", "", "127.0.0.1:3306", config.DBusername, config.DBpassword)
	err := db.Connect()
	if err != nil {
		error_log.Write(err.Error())
		fmt.Println("Failed to connect to the database, see log for details.")
		os.Exit(2)
	}
	if usedb {
		_,err = db.Start("USE `"+config.DBname+"`;")
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(2)
		}
	}
	db_connected = true
}


func dbTests() {
  	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"staff")
  	if err != nil {
  		fmt.Println(err.Error())
  		os.Exit(2)
  	}
  	/*var entry StaffTable
	if err != nil {
		error_log.Write(err.Error())
	}
	for results.Next() {
		err = results.Scan(&entry.username,&entry.password_checksum,&entry.rank)
		//if err !=  nil { panic(err) }
	}*/

	for {
	    row, err := results.GetRow()
	        if err != nil {
	        	error_log.Write(err.Error())
	        }

	        if row == nil {
	            // No more rows
	            break
	        }

	    // Print all cols
	    for _, col := range row {
	        if col == nil {
	            fmt.Print("<NULL>")
	        } else {
	            os.Stdout.Write(col.([]byte))
	        }
	        fmt.Print(" ")
	    }
	    fmt.Println()
	}
}