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
}

func loadInitialSetupFile() {
	initial_sql_bytes,err := ioutil.ReadFile("initialsetupdb.sql")
	initial_sql_str := string(initial_sql_bytes)
	fmt.Println("Starting initial setup...")
	if err == nil {
		initial_sql_str = strings.Replace(initial_sql_str,"DBNAME",config.DBname, -1)
		initial_sql_str = strings.Replace(initial_sql_str,"DBPREFIX",config.DBprefix, -1)
		initial_sql_str += "\nINSERT INTO `"+config.DBname+"`.`"+config.DBprefix+"staff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '"+bcrypt_sum("password")+"', 'abc', 3);"
		_,err := db.Start(initial_sql_str)

		if err != nil {
			fmt.Println("Initial setup failed.")
			error_log.Write(err.Error())
		} else {
			/*_,err := db.Start("INSERT INTO `"+config.DBname+"`.`"+config.DBprefix+"_staff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '"+bcrypt_sum("password")+"', 'abc', 3);")
			if err != nil {
				fmt.Println("Failed creating administrator account.")
				error_log.Write(err.Error())
			} else {*/
				fmt.Println("Initial setup complete.")
			
		}
	} else {
		error_log.Write("Couldn't load initial sql file")
		os.Exit(2)
	}
}