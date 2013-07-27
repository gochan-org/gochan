package main

import (
	"os"
	//"syscall"
	"fmt"
)

var (
	version float32 = 0.4
)


func main() {
	defer db.Close()
	initConfig()
	fmt.Println("Config file loaded. Connecting to database...")
	_,err := os.Stat("initialsetupdb.sql")
	//check if initialsetup file exists
	if err != nil {
		needs_initial_setup = false
		connectToSQLServer()
	} else {
		needs_initial_setup = true
		runInitialSetup()
	}
	fmt.Println("Loading and parsing templates...")
	initTemplates()
	fmt.Println("Initializing server...")
	if db != nil {
		db.Exec("USE `"+config.DBname+"`;")
	}
	go initServer()
	select {}
}