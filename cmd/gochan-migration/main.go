package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/kusabax"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/pre2021"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/tinyboard"
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
	bufIn      = bufio.NewReader(os.Stdin)
)

func fatalPrintln(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}

func readConfig(filename string, options *common.DBOptions) {
	ba, err := ioutil.ReadFile(filename)
	if err != nil {
		fatalPrintln(err)
		return
	}
	if err = json.Unmarshal(ba, options); err != nil {
		fatalPrintln(err)
	}
}

func main() {
	var options common.DBOptions
	var migrationConfigFile string

	flag.StringVar(&migrationConfigFile, "migrationconfig", "", "a JSON file to use for supplying the required migration information (ignores all other set arguments if used)")
	flag.StringVar(&options.OldChanType, "oldchan", "", "The imageboard we are migrating from (currently only pre2021 is supported, but more are coming")
	flag.StringVar(&options.Host, "dbhost", "", "The database host or socket file to connect to")
	flag.StringVar(&options.DBType, "dbtype", "mysql", "The kind of database server we are connecting to (currently only mysql is supported)")
	flag.StringVar(&options.Username, "dbusername", "", "The database username")
	flag.StringVar(&options.Password, "dbpassword", "", "The database password (if required by SQL account)")
	flag.StringVar(&options.OldDBName, "olddbname", "", "The name of the old database")
	flag.StringVar(&options.NewDBName, "newdbname", "", "The name of the new database")
	flag.StringVar(&options.TablePrefix, "tableprefix", "", "Prefix for the SQL tables' names")
	flag.Parse()

	if migrationConfigFile != "" {
		readConfig(migrationConfigFile, &options)
	}

	if options.OldChanType == "" || options.Host == "" || options.DBType == "" || options.Username == "" || options.OldDBName == "" || options.NewDBName == "" {
		flag.PrintDefaults()
		fmt.Println("Missing required database connection info")
		os.Exit(1)
		return
	}

	fmt.Printf(banner, versionStr)

	var migrator common.DBMigrator
	switch options.OldChanType {
	case "kusabax":
		migrator = &kusabax.KusabaXMigrator{}
	case "pre2021":
		migrator = &pre2021.Pre2021Migrator{}
	case "tinyboard":
		migrator = &tinyboard.TinyBoardMigrator{}
	default:
		fatalPrintln("Invalid oldchan value")
	}

	err := migrator.Init(options)
	if err != nil {
		fatalPrintln("Error initializing migrator:", err)
	}
	defer migrator.Close()

	// config.InitConfig(versionStr)
	/* gclog.Printf(gclog.LStdLog, "Starting gochan migration (gochan v%s)", versionStr)
	err := gcmigrate.Entry(1) //TEMP, get correct database version from command line or some kind of table. 1 Is the current version we are working towards
	if err != nil {
		gclog.Printf(gclog.LErrorLog, "Error while migrating: %s", err)
	} */
	if options.OldDBName == options.NewDBName {
		fatalPrintln("The old database name must not be the same as the new one.")
	}
	if err = migrator.MigrateDB(); err != nil {
		fatalPrintln(err)
	}
	fmt.Println("Database migration successful!")
}
