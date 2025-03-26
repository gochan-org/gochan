package gcutil

import (
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

// RunningInTerminal returns true if the ModeCharDevice bit is set, meaning that
// gochan is probably running in a standard terminal and not being piped
// to a file
func RunningInTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func init() {
	// guarantee that the logger is always available, even before the configuration is loaded
	logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.NoColor = !RunningInTerminal()
	})).With().Timestamp().Logger()
}

func initLog(logPath string, level zerolog.Level) (err error) {
	if logFile != nil {
		// log already initialized
		if err = logFile.Close(); err != nil {
			logger.Err(err).Msg("Unable to close log file")
			return err
		}
	}
	logFile, err = os.OpenFile(logPath, logFlags, logFileMode) // skipcq: GSC-G302
	if err != nil {
		logger.Err(err).Msg("Unable to open log file")
		return err
	}

	writer := zerolog.MultiLevelWriter(logFile, zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.NoColor = !RunningInTerminal()
	}))

	logger = zerolog.New(writer).With().Timestamp().Logger().Level(level)

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

func InitLogs(logDir string, level zerolog.Level, uid int, gid int) (err error) {
	if err = initLog(path.Join(logDir, "gochan.log"), level); err != nil {
		return err
	}
	if err = logFile.Chown(uid, gid); err != nil {
		return err
	}

	if err = initAccessLog(path.Join(logDir, "gochan_access.log")); err != nil {
		return err
	}
	return accessFile.Chown(uid, gid)
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

// LogAccess starts an info level zerolog event in the access log, with the requesters's
// path, IP, HTTP method, and user agent string
func LogAccess(request *http.Request) *zerolog.Event {
	ev := accessLogger.Info()
	if request != nil {
		ev.
			Str("path", request.URL.Path).
			Str("IP", GetRealIP(request)).
			Str("method", request.Method)
		ua := request.UserAgent()
		if ua != "" {
			ev.Str("userAgent", request.UserAgent())
		}
	}
	return ev
}

// LogRequest returns info and error level zerolog events with the requester's
// IP, the user-agent string, and the requested path and HTTP method
func LogRequest(request *http.Request) (*zerolog.Event, *zerolog.Event, *zerolog.Event) {
	infoEv := logger.Info()
	warnEv := logger.Warn()
	errEv := logger.Error()
	if request != nil {
		LogStr("IP", GetRealIP(request), infoEv, warnEv, errEv)
		LogStr("path", request.URL.Path, infoEv, warnEv, errEv)
		LogStr("method", request.Method, infoEv, warnEv, errEv)
		ua := request.UserAgent()
		if ua != "" {
			LogStr("userAgent", ua)
		}
	}
	return infoEv, warnEv, errEv
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
