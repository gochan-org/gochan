package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var (
	versionStr string
	stdFatal   = gclog.LStdLog | gclog.LFatal
)

func main() {
	defer func() {
		gclog.Print(gclog.LStdLog, "Cleaning up")
		gcsql.ExecSQL("DROP TABLE DBPREFIXsessions")
		gcsql.Close()
	}()

	gclog.Printf(gclog.LStdLog, "Starting gochan v%s", versionStr)
	config.InitConfig(versionStr)

	gcsql.ConnectToDB(
		config.Config.DBhost, config.Config.DBtype, config.Config.DBname,
		config.Config.DBusername, config.Config.DBpassword, config.Config.DBprefix)
	parseCommandLine()
	gcutil.InitMinifier()

	posting.InitCaptcha()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	posting.InitPosting()
	go initServer()
	<-sc
}

func parseCommandLine() {
	var newstaff string
	var delstaff string
	var rank int
	var err error
	flag.StringVar(&newstaff, "newstaff", "", "<newusername>:<newpassword>")
	flag.StringVar(&delstaff, "delstaff", "", "<username>")
	flag.IntVar(&rank, "rank", 0, "New staff member rank, to be used with -newstaff or -delstaff")
	flag.Parse()

	if newstaff != "" {
		arr := strings.Split(newstaff, ":")
		if len(arr) < 2 || delstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		gclog.Printf(gclog.LStdLog|gclog.LStaffLog, "Creating new staff: %q, with password: %q and rank: %d from command line", arr[0], arr[1], rank)
		if err = gcsql.NewStaff(arr[0], arr[1], rank); err != nil {
			gclog.Print(stdFatal, err.Error())
		}
		os.Exit(0)
	}
	if delstaff != "" {
		if newstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		gclog.Printf(gclog.LStdLog, "Are you sure you want to delete the staff account %q? [y/N]: ", delstaff)
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(answer)
		if answer == "y" || answer == "yes" {
			if err = gcsql.DeleteStaff(delstaff); err != nil {
				gclog.Printf(stdFatal, "Error deleting %q: %s", delstaff, err.Error())
			}
		} else {
			gclog.Print(stdFatal, "Not deleting.")
		}
	}
}
