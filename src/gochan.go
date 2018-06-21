package main

import (
	"os"
)

// set in Makefile via -ldflags
var version string
var buildtimeString string // set in Makefile, format: YRMMDD.HHMM

func main() {
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()
	initConfig()
	config.Version = version
	printf(0, "Starting gochan v%s.%s, using verbosity level %d\n", config.Version, buildtimeString, config.Verbosity)
	println(0, "Config file loaded. Connecting to database...")
	connectToSQLServer()

	println(0, "Loading and parsing templates...")
	if err := initTemplates(); err != nil {
		handleError(0, customError(err))
		os.Exit(2)
	}

	println(0, "Initializing server...")
	if db != nil {
		_, err := db.Exec("USE `" + config.DBname + "`")
		if err != nil {
			handleError(0, customError(err))
			os.Exit(2)
		}
	}
	initServer()
}
