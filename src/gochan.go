package main

import (
	"os"
)

var version string
var buildtimeString string // set in Makefile, format: YRMMDD.HHMM

func main() {
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()
	initConfig()
	printf(0, "Starting gochan v%s.%s, using verbosity level %d\n", config.Version, buildtimeString, config.Verbosity)
	println(0, "Config file loaded. Connecting to database...")
	connectToSQLServer()
	println(0, "Loading and parsing templates...")
	if err := initTemplates(); err != nil {
		handleError(0, customError(err))
		os.Exit(2)
	}

	println(0, "Initializing server...")
	if _, err := db.Exec("USE `" + config.DBname + "`"); err != nil {
		handleError(0, customError(err))
		os.Exit(2)
	}
	initServer()
}
