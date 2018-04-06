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
var verbosity_str string
var buildtime_str string // set in build.sh, format: YRMMDD.HHMM

func main() {
	defer db.Close()
	initConfig()
	config.Verbosity, _ = strconv.Atoi(verbosity_str)
	config.Version = version
	printf(0, "Starting gochan v%s.%s, using verbosity level %d\n", config.Version, buildtime_str, config.Verbosity)
	printf(0, "Config file loaded. Connecting to database...")
	connectToSQLServer()

	println(0, "Loading and parsing templates...")
	if err := initTemplates(); err != nil {
		println(0, err.Error())
		os.Exit(2)
	}

	println(0, "Initializing server...")
	if db != nil {
		db.Exec("USE `" + config.DBname + "`;")
	}
	initServer()
}
