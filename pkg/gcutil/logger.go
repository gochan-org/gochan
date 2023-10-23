package gcutil

import (
	"io/fs"
	"net/http"
	"os"
	"path"

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

func LogDiscard(events ...*zerolog.Event) {
	for e := range events {
		if events[e] == nil {
			continue
		}
		events[e] = events[e].Discard()
	}
}

func initLog(logPath string, debug bool) (err error) {
	if logFile != nil {
		// log file already initialized, skip
		return nil
	}
	logFile, err = os.OpenFile(logPath, logFlags, logFileMode) // skipcq: GSC-G302
	if err != nil {
		return err
	}

	if debug {
		multi := zerolog.MultiLevelWriter(logFile, zerolog.NewConsoleWriter())
		logger = zerolog.New(multi).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(logFile).With().Timestamp().Logger()
	}

	return nil
}

func initAccessLog(logPath string) (err error) {
	if accessFile != nil {
		// access log already initialized, skip
		return nil
	}
	accessFile, err = os.OpenFile(logPath, logFlags, logFileMode) // skipcq: GSC-G302
	if err != nil {
		return err
	}
	accessLogger = zerolog.New(accessFile).With().Timestamp().Logger()
	return nil
}

func InitLogs(logDir string, debug bool, uid int, gid int) (err error) {
	if err = initLog(path.Join(logDir, "gochan.log"), debug); err != nil {
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
