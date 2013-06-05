package main

import (
	"os"
	"fmt"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native" // Native engine
	//_ "github.com/ziutek/mymysql/thrsafe" // Thread safe engine
)

var (
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