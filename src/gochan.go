package main

import (
	"fmt"
)

var (
	version float32 = 0.5
)


func main() {
	defer db.Close()
	initConfig()
	fmt.Println("Config file loaded. Connecting to database...")
	connectToSQLServer()

	fmt.Println("Loading and parsing templates...")
	initTemplates()
	fmt.Println("Initializing server...")
	if db != nil {
		db.Exec("USE `"+config.DBname+"`;")
	}
	initServer()
}