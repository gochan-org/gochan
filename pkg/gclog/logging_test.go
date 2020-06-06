package gclog

import (
	"os"
	"testing"
)

func TestGochanLog(t *testing.T) {
	_, err := os.Stat("./logtest")
	if err != nil {
		if err = os.Mkdir("./logtest", 0755); err != nil {
			t.Fatal(err.Error())
		}
	}

	err = InitLogs("./logtest/access.log", "./logtest/error.log", "./logtest/staff.log", true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// test Print usage
	gclog.Print(LStdLog, "os.Stdout log", "(Print)")
	gclog.Print(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs", "(Print)")
	gclog.Print(LAccessLog, "Access log", "(Print)")
	gclog.Print(LErrorLog, "Error log", "(Print)")
	gclog.Print(LStaffLog, "Staff log", "(Print)")
	gclog.Print(LAccessLog|LErrorLog, "Access and error log", "(Print)")
	gclog.Print(LAccessLog|LStaffLog, "Access and staff log", "(Print)")

	// test Printf usage
	gclog.Printf(LStdLog, "os.Stdout log %q", "(Println)")
	gclog.Printf(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs %q", "(Printf)")
	gclog.Printf(LAccessLog, "Access log %q", "(Printf)")
	gclog.Printf(LErrorLog, "Error log %q", "(Printf)")
	gclog.Printf(LStaffLog, "Staff log %q", "(Printf)")
	gclog.Printf(LAccessLog|LErrorLog, "Access and error log %q", "(Printf)")
	gclog.Printf(LAccessLog|LStaffLog, "Access and staff log %q", "(Printf)")

	// test Println usage (proper spacing and no extra newlines)
	gclog.Println(LStdLog, "os.Stdout log", "(Println)")
	gclog.Println(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs", "(Println)")
	gclog.Println(LAccessLog, "Access log", "(Println)")
	gclog.Println(LErrorLog, "Error log", "(Println)")
	gclog.Println(LStaffLog, "Staff log", "(Println)")
	gclog.Println(LAccessLog|LErrorLog, "Access and error log", "(Println)")
	gclog.Println(LAccessLog|LStaffLog|LFatal, "Fatal access and staff log", "(Println)")
	gclog.Println(LAccessLog, "This shouldn't be here", "(Println)")
}
