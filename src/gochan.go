package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var versionStr string
var buildtimeString string // set with build command, format: YRMMDD.HHMM

func main() {
	defer func() {
		gclog.Print(lErrorLog|lStdLog, "Cleaning up")
		execSQL("DROP TABLE DBPREFIXsessions")
		db.Close()
	}()
	initConfig()
	initMinifier()

	gclog.Printf(lErrorLog|lStdLog, "Starting gochan v%s", versionStr)
	connectToSQLServer()
	parseCommandLine()

	gclog.Print(lErrorLog|lStdLog, "Loading and parsing templates")
	if err := initTemplates("all"); err != nil {
		gclog.Printf(lErrorLog|lStdLog|lFatal, err.Error())
	}
	gclog.Print(lErrorLog|lStdLog, buildJS())
	initCaptcha()
	tempCleanerTicker = time.NewTicker(time.Minute * 5)
	go tempCleaner()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		initServer()
	}()
	<-sc
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
			os.Exit(1)
		}
		gclog.Printf(lStdLog|lErrorLog, "Creating new staff: %q, with password: %q and rank: %d", arr[0], arr[1], rank)
		if err = newStaff(arr[0], arr[1], rank); err != nil {
			gclog.Print(lStdLog|lFatal, err.Error())
		}
		os.Exit(0)
	}
	if delstaff != "" {
		if newstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		gclog.Print(lStdLog, "Are you sure you want to delete the staff account %q?[y/N]: ", delstaff)
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(answer)
		if answer == "y" || answer == "yes" {
			if err = deleteStaff(delstaff); err != nil {
				gclog.Print(lStdLog|lFatal, "Error deleting %q: %s", delstaff, err.Error())
			}
		} else {
			gclog.Print(lStdLog|lFatal, "Not deleting.")
		}
	}
}
