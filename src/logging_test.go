package main

import "testing"

func TestGochanLog(t *testing.T) {
	gcl, err := initLogs("../access.log", "../error.log", "../staff.log")
	if err != nil {
		t.Fatal(err.Error())
	}
	gcl.Print(lStdLog, "os.Stdout log")
	gcl.Print(lStdLog|lAccessLog|lErrorLog|lStaffLog, "all logs")
	gcl.Print(lAccessLog, "Access log")
	gcl.Print(lErrorLog, "Error log")
	gcl.Print(lStaffLog, "Staff log")
	gcl.Print(lAccessLog|lErrorLog, "Access and error log")
	gcl.Print(lAccessLog|lStaffLog|lFatal, "Fatal access and staff log")
	gcl.Print(lAccessLog, "This shouldn't be here")
}
