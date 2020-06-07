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
	Print(LStdLog, "os.Stdout log", "(Print)")
	Print(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs", "(Print)")
	Print(LAccessLog, "Access log", "(Print)")
	Print(LErrorLog, "Error log", "(Print)")
	Print(LStaffLog, "Staff log", "(Print)")
	Print(LAccessLog|LErrorLog, "Access and error log", "(Print)")
	Print(LAccessLog|LStaffLog, "Access and staff log", "(Print)")

	// test Printf usage
	Printf(LStdLog, "os.Stdout log %q", "(Println)")
	Printf(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs %q", "(Printf)")
	Printf(LAccessLog, "Access log %q", "(Printf)")
	Printf(LErrorLog, "Error log %q", "(Printf)")
	Printf(LStaffLog, "Staff log %q", "(Printf)")
	Printf(LAccessLog|LErrorLog, "Access and error log %q", "(Printf)")
	Printf(LAccessLog|LStaffLog, "Access and staff log %q", "(Printf)")

	// test Println usage (proper spacing and no extra newlines)
	Println(LStdLog, "os.Stdout log", "(Println)")

	t.Logf("%q", Println(LStdLog, "Testing log chaining for errors", "(Println)"))

	Println(LStdLog|LAccessLog|LErrorLog|LStaffLog, "all logs", "(Println)")
	Println(LAccessLog, "Access log", "(Println)")
	Println(LErrorLog, "Error log", "(Println)")
	Println(LStaffLog, "Staff log", "(Println)")
	Println(LAccessLog|LErrorLog, "Access and error log", "(Println)")
	Println(LAccessLog|LStaffLog|LFatal, "Fatal access and staff log", "(Println)")
	Println(LAccessLog, "This shouldn't be here", "(Println)")
}
