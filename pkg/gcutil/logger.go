package gcutil

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/rs/zerolog"
)

const (
	logFlags                = os.O_CREATE | os.O_APPEND | os.O_WRONLY
	logFileMode fs.FileMode = 0644
)

var (
	logFile    *os.File
	accessFile *os.File

	logger       zerolog.Logger
	accessLogger zerolog.Logger
)

func LogStr(key, val string, events ...*zerolog.Event) {
	for e := range events {
		if events[e] != nil {
			events[e] = events[e].Str(key, val)
		}
	}
}

func LogInt(key string, i int, events ...*zerolog.Event) {
	for e := range events {
		if events[e] != nil {
			events[e] = events[e].Int(key, i)
		}
	}
}

func LogBool(key string, b bool, events ...*zerolog.Event) {
	for e := range events {
		if events[e] != nil {
			events[e] = events[e].Bool(key, b)
		}
	}
}

func LogTime(key string, t time.Time, events ...*zerolog.Event) {
	for e := range events {
		if events[e] != nil {
			events[e] = events[e].Time(key, t)
		}
	}
}

func LogDiscard(events ...*zerolog.Event) {
	for e := range events {
		if events[e] == nil {
			continue
		}
		events[e] = events[e].Discard()
	}
}

// isTerminal returns true if the ModeCharDevice bit is set, meaning that
// gochan is probably running in a standard terminal and not being piped
// to a file
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func initLog(logPath string, logToConsole bool) (err error) {
	if logFile != nil {
		// log already initialized
		if err = logFile.Close(); err != nil {
			return err
		}
	}
	logFile, err = os.OpenFile(logPath, logFlags, logFileMode) // skipcq: GSC-G302
	if err != nil {
		return err
	}

	var writer io.Writer
	if logToConsole {
		cw := zerolog.NewConsoleWriter()
		cw.NoColor = !isTerminal()
		writer = zerolog.MultiLevelWriter(logFile, cw)
	} else {
		writer = logFile
	}
	logger = zerolog.New(writer).With().Timestamp().Logger()

	return nil
}

func initAccessLog(logPath string) (err error) {
	if accessFile != nil {
		// access log already initialized, close it first before reopening
		if err = accessFile.Close(); err != nil {
			return err
		}
	}
	accessFile, err = os.OpenFile(logPath, logFlags, logFileMode) // skipcq: GSC-G302
	if err != nil {
		return err
	}
	accessLogger = zerolog.New(accessFile).With().Timestamp().Logger()
	return nil
}

func InitLogs(logDir string, verbose bool, uid int, gid int) (err error) {
	if err = initLog(path.Join(logDir, "gochan.log"), verbose); err != nil {
		return err
	}
	if err = logFile.Chown(uid, gid); err != nil {
		return err
	}

	if err = initAccessLog(path.Join(logDir, "gochan_access.log")); err != nil {
		return err
	}
	if err = accessFile.Chown(uid, gid); err != nil {
		return err
	}

	return nil
}

func Logger() *zerolog.Logger {
	return &logger
}

func LogInfo() *zerolog.Event {
	return logger.Info()
}

func LogWarning() *zerolog.Event {
	return logger.Warn()
}

func LogAccess(request *http.Request) *zerolog.Event {
	ev := accessLogger.Info()
	if request != nil {
		return ev.
			Str("access", request.URL.Path).
			Str("IP", GetRealIP(request))
	}
	return ev
}

func LogError(err error) *zerolog.Event {
	if err != nil {
		return logger.Err(err)
	}
	return logger.Error()
}

func LogFatal() *zerolog.Event {
	return logger.Fatal()
}

func LogDebug() *zerolog.Event {
	return logger.Debug()
}

func CloseLog() error {

	if logFile == nil {
		return nil
	}
	return logFile.Close()
}
