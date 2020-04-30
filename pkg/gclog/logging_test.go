package gclog

import "testing"

func TestGochanLog(t *testing.T) {
	err := InitLogs("../access.log", "../error.log", "../staff.log", true)
	if err != nil {
		t.Fatal(err.Error())
	}
	gclog.Print(LStdLog, "os.Stdout log")
	gclog.Print(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs")
	gclog.Print(LAccessLog, "Access log")
	gclog.Print(LErrorLog, "Error log")
	gclog.Print(LStaffLog, "Staff log")
	gclog.Print(LAccessLog|LErrorLog, "Access and error log")
	gclog.Print(LAccessLog|LStaffLog|LFatal, "Fatal access and staff log")
	gclog.Print(LAccessLog, "This shouldn't be here")
}
