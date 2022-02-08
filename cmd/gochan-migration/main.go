package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/pre2021"
	"github.com/gochan-org/gochan/pkg/config"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	banner = `Welcome to the gochan database migration tool for gochan %s!
This migration tool is currently very unstable, and will likely go through
several changes before it can be considered "stable", so make sure you check
the README and/or the -h command line flag before you use it.

`
)

var (
	versionStr string
)

func fatalPrintln(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}

func main() {
	var options common.MigrationOptions

	config.InitConfig(versionStr)

	flag.StringVar(&options.ChanType, "oldchan", "", "The imageboard we are migrating from (currently only pre2021 is supported, but more are coming")
	flag.StringVar(&options.OldChanConfig, "oldconfig", "", "The path to the old chan's configuration file")
	flag.Parse()

	if options.ChanType == "" || options.OldChanConfig == "" {
		flag.PrintDefaults()
		fmt.Println("Missing required database connection info")
		os.Exit(1)
		return
	}

	fmt.Printf(banner, versionStr)
	var migrator common.DBMigrator
	switch options.ChanType {
	case "pre2021":
		migrator = &pre2021.Pre2021Migrator{}
	case "kusabax":
		fallthrough
	case "tinyboard":
		fallthrough
	default:
		fmt.Printf(
			"Unsupported chan type %q, Currently only pre2021 database migration is supported\n",
			options.ChanType)
		os.Exit(1)
	}
	err := migrator.Init(options)
	if err != nil {
		fmt.Printf("Unable to initialize %s migrator: %s\n", options.ChanType, err.Error())
		os.Exit(1)
	}
	defer migrator.Close()
	if err = migrator.MigrateDB(); err != nil {
		fmt.Println("Error migrating database: ", err.Error())
		os.Exit(1)
	}
	fmt.Println("Database migration successful!")
}
