package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/gochan-org/gochan/cmd/gochan-migration/gcmigrate"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
)

var (
	versionStr string
	stdFatal   = gclog.LStdLog | gclog.LFatal
)

func main() {
	config.InitConfig(versionStr)
	gclog.Printf(gclog.LStdLog, "Starting gochan migration (gochan v%s)", versionStr)
	gcmigrate.Entry(1) //TEMP, get correct database version from command line or some kind of table. 1 Is the current version we are working towards
}
