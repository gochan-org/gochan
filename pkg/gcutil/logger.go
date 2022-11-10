package gcutil

import (
	"net/http"
	"os"

	"github.com/rs/zerolog"
)

var (
	logFile      *os.File
	accessFile   *os.File
	logger       zerolog.Logger
	accessLogger zerolog.Logger
)

type logHook struct{}

func (*logHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level != zerolog.Disabled && level != zerolog.NoLevel {
		e.Timestamp()
	}
}

func LogStr(key, val string, events ...*zerolog.Event) {
	for e := range events {
		if events[e] == nil {
			continue
		}
		events[e] = events[e].Str(key, val)
	}
}

func LogInt(key string, i int, events ...*zerolog.Event) {
	for e := range events {
		if events[e] == nil {
			continue
		}
		events[e] = events[e].Int(key, i)
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

func InitLog(logPath string) (err error) {
	if logFile != nil {
		// log file already initialized, skip
		return nil
	}
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	logger = zerolog.New(logFile).Hook(&logHook{})
	return nil
}

func InitAccessLog(logPath string) (err error) {
	if accessFile != nil {
		// access log already initialized, skip
		return nil
	}
	accessFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	accessLogger = zerolog.New(accessFile).Hook(&logHook{})
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
