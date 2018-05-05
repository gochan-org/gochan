package main

import (
	"os"
	"strconv"
)

// set in Makefile via -ldflags
var version string

// verbose = 0 for no debugging info. Critical errors and general output only
// verbose = 1 for non-critical warnings and important info
// verbose = 2 for all debugging/benchmarks/warnings
// set in Makefile via -ldflags
var verbosityString string
var buildtimeString string // set in Makefile, format: YRMMDD.HHMM

func main() {
	defer db.Close()
	initConfig()
	config.Verbosity, _ = strconv.Atoi(verbosityString)
	config.Version = version
	printf(0, "Starting gochan v%s.%s, using verbosity level %d\n", config.Version, buildtimeString, config.Verbosity)
	printf(0, "Config file loaded. Connecting to database...")
	connectToSQLServer()

	println(0, "Loading and parsing templates...")
	if err := initTemplates(); err != nil {
		println(0, err.Error())
		os.Exit(2)
	}

	println(0, "Initializing server...")
	if db != nil {
		_, err := db.Exec("USE `" + config.DBname + "`")
		if err != nil {
			println(0, customError(err))
		}
	}
	initServer()
}
