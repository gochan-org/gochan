package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
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
	connectToSQLServer()
	parseCommandLine()

	printf(0, "Starting gochan v%s.%s, using verbosity level %d\n", config.Version, buildtimeString, config.Verbosity)
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

func parseCommandLine() {
	var newstaff string
	var delstaff string
	var rank int
	var err error
	flag.StringVar(&newstaff, "newstaff", "", "<newusername>:<newpassword>")
	flag.StringVar(&delstaff, "delstaff", "", "<username>")
	flag.IntVar(&rank, "rank", 0, "New staff member rank, to be used with -newstaff or -delstaff")
	flag.Parse()

	if newstaff != "" {
		arr := strings.Split(newstaff, ":")
		if len(arr) < 2 || delstaff != "" {
			flag.Usage()
			os.Exit(2)
		}
		printf(0, "Creating new staff: '%s', with password: '%s' and rank: %d\n", arr[0], arr[1], rank)
		if err = newStaff(arr[0], arr[1], rank); err != nil {
			handleError(0, err.Error())
			os.Exit(2)
		}
		os.Exit(0)
	}
	if delstaff != "" {
		if newstaff != "" {
			flag.Usage()
			os.Exit(2)
		}
		printf(0, "Are you sure you want to delete the staff account '%s'?[y/N]: ", delstaff)
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(answer)
		if answer == "y" || answer == "yes" {
			if err = deleteStaff(delstaff); err != nil {
				printf(0, "Error deleting '%s': %s\n", delstaff, err.Error())
				os.Exit(2)
			}
		} else {
			println(0, "Not deleting.")
		}
		os.Exit(0)
	}
}
