package main

import (
	"os"
	"fmt"
	"database/sql"
	_ "go-mysql-driver/mysql"
)

var (
	db *sql.DB
)

func connectToDB() {
	db, err = sql.Open("mysql",db_username+":"+db_password+"@"+db_host+"/"+db_name+"?charset=utf8&keepalive="+db_persistent_str)
	if err != nil {
		error_log.Write(err.Error())
		fmt.Println("Failed to connect to the database, see log for details.")
		os.Exit(2)
	}
}

func dbTests() {
  	results,err := db.Query("SELECT * FROM `"+db_prefix+"modlog")
	if err != nil {
		error_log.Write(err.Error())
		fmt.Println("Failed to connect to the database, see log for details.")
		os.Exit(2)
	}
	var entry ModLogTable
	for results.Next() {
		err = results.Scan(&entry.entry,&entry.user,&entry.category,&entry.timestamp)
		//if err !=  nil { panic(err) }
	}
}