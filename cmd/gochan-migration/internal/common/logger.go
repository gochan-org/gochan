package common

import (
	"io"
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

const (
	logFlags                = os.O_CREATE | os.O_APPEND | os.O_WRONLY
	logFileMode fs.FileMode = 0644
)

var (
	migrationLogFile *os.File
	migrationLog     zerolog.Logger
)

func InitTestMigrationLog(t *testing.T) (err error) {
	dir := os.TempDir()
	migrationLogFile, err = os.CreateTemp(dir, "migration-test")
	if err != nil {
		return err
	}
	migrationLog = zerolog.New(zerolog.NewTestWriter(t))
	return nil
}

func InitMigrationLog() (err error) {
	if migrationLogFile != nil {
		// Migration log already initialized
		return nil
	}
	logPath := path.Join(config.GetSystemCriticalConfig().LogDir, "migration.log")
	migrationLogFile, err = os.OpenFile(logPath, logFlags, logFileMode)
	if err != nil {
		return err
	}
	var writer io.Writer
	cw := zerolog.NewConsoleWriter()
	cw.NoColor = !gcutil.RunningInTerminal()
	writer = zerolog.MultiLevelWriter(migrationLogFile, cw)
	migrationLog = zerolog.New(writer).With().Timestamp().Logger()
	return nil
}

func Logger() *zerolog.Logger {
	return &migrationLog
}

func LogInfo() *zerolog.Event {
	return migrationLog.Info()
}

func LogWarning() *zerolog.Event {
	return migrationLog.Warn()
}

func LogError() *zerolog.Event {
	return migrationLog.Error()
}

func LogFatal() *zerolog.Event {
	return migrationLog.Fatal()
}

func CloseLog() error {
	if migrationLogFile == nil {
		return nil
	}
	return migrationLogFile.Close()
}
