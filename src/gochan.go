package main

import (
	"strconv"
)

// set in build.sh via -ldflags
var version string

// verbose = 0 for no debugging info. Critical errors and general output only
// verbose = 1 for non-critical warnings and important info
// verbose = 2 for all debugging/benchmarks/warnings
// set in build.sh via -ldflags
var verbose_str string
var buildtime_str string // set in build.sh, format: YRMMDD.HHMM

func main() {
	defer db.Close()
	initConfig()
	config.Verbosity, _ = strconv.Atoi(verbose_str)
	config.Version = version
	printf(0, "Starting gochan v%s.%s, using verbosity level %d\n", config.Version, buildtime_str, config.Verbosity)
	printf(0, "Config file loaded. Connecting to database...")
	connectToSQLServer()

	println(0, "Loading and parsing templates...")
	initTemplates()
	println(0, "Initializing server...")
	if db != nil {
		db.Exec("USE `" + config.DBname + "`;")
	}
	initServer()
}
