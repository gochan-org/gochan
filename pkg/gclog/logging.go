package gclog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

var gclog *GcLogger

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

// GcLogger is used for printing to access, error, and staff logs
type GcLogger struct {
	accessFile *os.File
	errorFile  *os.File
	staffFile  *os.File
	debug      bool
}

func (gcl *GcLogger) selectLogs(flags int) []*os.File {
	var logs []*os.File
	if flags&LAccessLog > 0 {
		logs = append(logs, gcl.accessFile)
	}
	if flags&LErrorLog > 0 {
		logs = append(logs, gcl.errorFile)
	}
	if flags&LStaffLog > 0 {
		logs = append(logs, gcl.staffFile)
	}
	if (flags&LStdLog > 0) || gcl.debug {
		logs = append(logs, os.Stdout)
	}
	return logs
}

func (gcl *GcLogger) getPrefix() string {
	prefix := time.Now().Format(logTimeFmt)
	_, file, line, _ := runtime.Caller(3)
	prefix += fmt.Sprint(file, ":", line, ": ")

	return prefix
}

// Print is comparable to log.Print but takes binary flags
func (gcl *GcLogger) Print(flags int, v ...interface{}) string {
	str := fmt.Sprint(v...)
	logs := gcl.selectLogs(flags)
	for _, l := range logs {
		if l == os.Stdout {
			io.WriteString(l, str+"\n")
		} else {
			io.WriteString(l, gcl.getPrefix()+str+"\n")
		}
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str
}

// Printf is comparable to log.Logger.Printf but takes binary OR'd flags
func (gcl *GcLogger) Printf(flags int, format string, v ...interface{}) string {
	str := fmt.Sprintf(format, v...)
	logs := gcl.selectLogs(flags)
	for _, l := range logs {
		if l == os.Stdout {
			io.WriteString(l, str+"\n")
		} else {
			io.WriteString(l, gcl.getPrefix()+str+"\n")
		}
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str
}

// Println is comparable to log.Logger.Println but takes binary OR'd flags
func (gcl *GcLogger) Println(flags int, v ...interface{}) string {
	str := fmt.Sprintln(v...)
	logs := gcl.selectLogs(flags)
	for _, l := range logs {
		if l == os.Stdout {
			io.WriteString(l, str+"\n")
		} else {
			io.WriteString(l, gcl.getPrefix()+str+"\n")
		}
	}
	if flags&LFatal > 0 {
		os.Exit(1)
	}
	return str
}

// Close closes the log file handles
func (gcl *GcLogger) Close() {
	gcl.accessFile.Close()
	gcl.errorFile.Close()
	gcl.staffFile.Close()
}

// Print is comparable to log.Print but takes binary OR'd flags
func Print(flags int, v ...interface{}) string {
	if gclog == nil {
		return ""
	}
	return gclog.Print(flags, v...)
}

// Printf is comparable to log.Printf but takes binary OR'd flags
func Printf(flags int, format string, v ...interface{}) string {
	if gclog == nil {
		return ""
	}
	return gclog.Printf(flags, format, v...)
}

// Println is comparable to log.Println but takes binary OR'd flags
func Println(flags int, v ...interface{}) string {
	if gclog == nil {
		return ""
	}
	return gclog.Println(flags, v...)
}

// InitLogs initializes the log files to be used by gochan
func InitLogs(accessLogPath, errorLogPath, staffLogPath string, debugMode bool) error {
	gclog = &GcLogger{debug: debugMode}
	var err error
	if gclog.accessFile, err = os.OpenFile(accessLogPath, logFileFlags, 0777); err != nil {
		return errors.New("Error loading " + accessLogPath + ": " + err.Error())
	}
	if gclog.errorFile, err = os.OpenFile(errorLogPath, logFileFlags, 0777); err != nil {
		return errors.New("Error loading " + errorLogPath + ": " + err.Error())
	}
	if gclog.staffFile, err = os.OpenFile(staffLogPath, logFileFlags, 0777); err != nil {
		return errors.New("Error loading " + staffLogPath + ": " + err.Error())

	}
	return nil
}
