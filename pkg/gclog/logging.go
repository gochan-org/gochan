package gclog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

const (
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
	accessFile *os.File
	errorFile  *os.File
	staffFile  *os.File
	debugLog   bool
)

func selectLogs(flags int) []*os.File {
	var logs []*os.File
	if flags&LAccessLog > 0 {
		logs = append(logs, accessFile)
	}
	if flags&LErrorLog > 0 {
		logs = append(logs, errorFile)
	}
	if flags&LStaffLog > 0 {
		logs = append(logs, staffFile)
	}
	if (flags&LStdLog > 0) || debugLog {
		logs = append(logs, os.Stdout)
	}
	return logs
}

func getPrefix() string {
	prefix := time.Now().Format(logTimeFmt)
	_, file, line, _ := runtime.Caller(2)
	prefix += fmt.Sprint(file, ":", line, ": ")

	return prefix
}

// Print is comparable to log.Print but takes binary flags
func Print(flags int, v ...interface{}) string {
	str := fmt.Sprint(v...)
	logs := selectLogs(flags)
	for _, l := range logs {
		if l == os.Stdout {
			io.WriteString(l, str+"\n")
		} else {
			io.WriteString(l, getPrefix()+str+"\n")
		}
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
		if l == os.Stdout {
			io.WriteString(l, str+"\n")
		} else {
			io.WriteString(l, getPrefix()+str+"\n")
		}
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
		if l == os.Stdout {
			io.WriteString(l, str)
		} else {
			io.WriteString(l, getPrefix()+str)
		}
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str[:len(str)-1]
}

// Close closes the log file handles
func Close() {
	if accessFile != nil {
		accessFile.Close()
	}
	if errorFile != nil {
		errorFile.Close()
	}
	if staffFile != nil {
		staffFile.Close()
	}
}

// InitLogs initializes the log files to be used by gochan
func InitLogs(accessLogPath, errorLogPath, staffLogPath string, debugMode bool) error {
	debugLog = debugMode
	var err error
	if accessFile, err = os.OpenFile(accessLogPath, logFileFlags, 0777); err != nil {
		return errors.New("Error loading " + accessLogPath + ": " + err.Error())
	}
	if errorFile, err = os.OpenFile(errorLogPath, logFileFlags, 0777); err != nil {
		return errors.New("Error loading " + errorLogPath + ": " + err.Error())
	}
	if staffFile, err = os.OpenFile(staffLogPath, logFileFlags, 0777); err != nil {
		return errors.New("Error loading " + staffLogPath + ": " + err.Error())

	}
	return nil
}
