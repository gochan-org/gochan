// functions for setting up SQL tables and the base administrator account

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func runInitialSetup() {
	if !db_connected {
		connectToSQLServer(false)
	}
	loadInitialSetupFile()
	db.Close()
	connectToSQLServer(true)
}

func loadInitialSetupFile() {
	initial_sql_bytes,err := ioutil.ReadFile("initialsetupdb.sql")
	initial_sql_str := string(initial_sql_bytes)
	fmt.Printf("Starting initial setup...")
	if err == nil {
		initial_sql_str = strings.Replace(initial_sql_str,"DBNAME",config.DBname, -1)
		initial_sql_str = strings.Replace(initial_sql_str,"DBPREFIX",config.DBprefix, -1)
		initial_sql_str += "\nINSERT INTO `"+config.DBname+"`.`"+config.DBprefix+"staff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '"+bcrypt_sum("password")+"', 'abc', 3);"
		initial_sql_arr := strings.Split(initial_sql_str, ";")
		for _,statement := range initial_sql_arr {
			if statement != "" {
				_,err := db.Exec(statement+";")
				if err != nil {
					fmt.Println("failed.")
					db.Exec("USE `"+config.DBname+"`;")
					error_log.Write(err.Error())
					return
				} 
			}
		}

		fmt.Println("complete.")
		db.Exec("USE `"+config.DBname+"`;")
	} else {
		error_log.Write("failed. Couldn't load initial sql file")
		os.Exit(2)
	}
}