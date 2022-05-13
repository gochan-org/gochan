package gclog

import (
	"fmt"
	"os"

	"github.com/gochan-org/gzlog"
)

const (
	logMaxSize   = 1000000 // 1 MB (TODO: make this configurable)
	logTimeFmt   = "2006/01/02 15:04:05 "
	logFileFlags = os.O_APPEND | os.O_CREATE | os.O_RDWR
	// LAccessLog should be used for incoming requests
	LAccessLog = 1 << iota
	// LErrorLog should be used for internal errors, not HTTP errors like 4xx
	LErrorLog
	// LStaffLog is comparable to LAccessLog for staff actions
	LStaffLog
	// LStdLog prints directly to standard output
	LStdLog
	// LFatal exits gochan after printing (used for fatal initialization errors)
	LFatal
)

var (
	accessLog *gzlog.GzLog
	errorLog  *gzlog.GzLog
	staffLog  *gzlog.GzLog
	debugLog  bool
)

func wantStdout(flags int) bool {
	return debugLog || flags&LStdLog > 0
}

// selectLogs returns an array of GzLogs that we should loop through given
// the flags passed to it
func selectLogs(flags int) []*gzlog.GzLog {
	var logs []*gzlog.GzLog
	if flags&LAccessLog > 0 {
		logs = append(logs, accessLog)
	}
	if flags&LErrorLog > 0 {
		logs = append(logs, errorLog)
	}
	if flags&LStaffLog > 0 {
		logs = append(logs, staffLog)
	}
	return logs
}

// Print is comparable to log.Print but takes binary flags
func Print(flags int, v ...interface{}) string {
	str := fmt.Sprint(v...)
	logs := selectLogs(flags)
	for _, l := range logs {
		l.Print(v...)
	}
	if wantStdout(flags) {
		fmt.Print(v...)
		fmt.Println()
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str
}

// Printf is comparable to log.Logger.Printf but takes binary OR'd flags
func Printf(flags int, format string, v ...interface{}) string {
	str := fmt.Sprintf(format, v...)
	logs := selectLogs(flags)
	for _, l := range logs {
		l.Printf(format, v...)
	}
	if wantStdout(flags) {
		fmt.Printf(format+"\n", v...)
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str
}

// Println is comparable to log.Logger.Println but takes binary OR'd flags
func Println(flags int, v ...interface{}) string {
	str := fmt.Sprintln(v...)
	logs := selectLogs(flags)
	for _, l := range logs {
		l.Println(v...)
	}
	if wantStdout(flags) {
		fmt.Println(v...)
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str[:len(str)-1]
}

// Close closes the log files
func Close() {
	if accessLog != nil {
		accessLog.Close()
	}
	if errorLog != nil {
		errorLog.Close()
	}
	if staffLog != nil {
		staffLog.Close()
	}
}

// InitLogs initializes the log files to be used by gochan
func InitLogs(accessLogBasePath, errorLogBasePath, staffLogBasePath string, debugMode bool) error {
	debugLog = debugMode
	var err error

	if accessLog, err = gzlog.OpenFile(accessLogBasePath, logMaxSize, 0640); err != nil {
		return err
	}
	if errorLog, err = gzlog.OpenFile(errorLogBasePath, logMaxSize, 0640); err != nil {
		return err
	}
	if staffLog, err = gzlog.OpenFile(staffLogBasePath, logMaxSize, 0640); err != nil {
		return err
	}
	return nil
}
