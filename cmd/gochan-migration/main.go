package main

import (
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

}
