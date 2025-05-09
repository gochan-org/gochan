package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
	"golang.org/x/term"
)

var (
	errAborted = fmt.Errorf("aborted")
)

func getPassword() (string, error) {
	var password string
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, state)

	for {
		input := make([]byte, 1)
		if _, err := syscall.Read(int(fd), input); err != nil {
			term.Restore(fd, state)
			return "", err
		}
		if input[0] == '\n' || input[0] == '\r' {
			term.Restore(fd, state)
			fmt.Println()
			break
		}
		if input[0] == 127 || input[0] == 8 {
			if len(password) > 0 {
				password = password[:len(password)-1]
				fmt.Print("\b \b")
			}
		} else if input[0] == 3 {
			term.Restore(fd, state)
			return "", errAborted
		} else {
			password += string(input[0])
			fmt.Print("*")
		}
	}
	return password, nil
}

func printInfoAndLog(msg string, infoEv ...*zerolog.Event) {
	fmt.Println(msg)
	if len(infoEv) > 0 {
		infoEv[0].Msg(msg)
	} else {
		gcutil.LogInfo().Str("source", "commandLine").Msg(msg)
	}
}

func initCommandLine() *zerolog.Event {
	var err error
	if err = config.InitConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing config:", err)
		os.Exit(1)
	}
	systemCritical := config.GetSystemCriticalConfig()
	if err = gcutil.InitLogs(systemCritical.LogDir, &gcutil.LogOptions{FileOnly: true}); err != nil {
		os.Exit(1)
	}
	fatalEv := gcutil.LogFatal()

	initDB(fatalEv)
	return fatalEv
}

func fatalAndLog(msg string, err error, fatalEv *zerolog.Event) {
	fmt.Fprintln(os.Stderr, msg, err)
	fatalEv.Err(err).Caller(1).Msg(msg)
}

func parseCommandLine() {
	var newstaff string
	var delstaff string
	var password string
	var rank int
	var err error

	if len(os.Args) < 2 {
		return
	}

	cmd := os.Args[1]
	var fatalEv *zerolog.Event

	switch cmd {
	case "version", "-v", "-version":
		fmt.Println(config.GochanVersion)
		return
	case "help", "-h", "-help":
		fmt.Println("Usage: gochan [command] [options]")
		fmt.Println("Commands:")
		fmt.Println("  version       Show the version of gochan")
		fmt.Println("  help          Show this help message")
		fmt.Println("  newstaff      Create a new staff account")
		fmt.Println("  delstaff      Delete a staff account")
		fmt.Println("  rebuild       Rebuild the specified components")
		fmt.Println("Run 'gochan [command] --help' for more information on a command.")
	case "newstaff":
		flagSet := flag.NewFlagSet("newstaff", flag.ExitOnError)
		flagSet.StringVar(&newstaff, "username", "", "Username for the new staff account")
		flagSet.StringVar(&password, "password", "", "Password for the new staff account")
		flagSet.IntVar(&rank, "rank", 0, "Rank for the new staff account")
		flagSet.Parse(os.Args[2:])
		if newstaff == "" || rank <= 0 {
			fmt.Fprintln(os.Stderr, "Error: -username and -rank are required")
			flagSet.Usage()
			os.Exit(1)
		}

		if password == "" {
			fmt.Print("Enter password for new staff account: ")
			password, err = getPassword()
			if err != nil {
				if errors.Is(err, errAborted) {
					fmt.Println("Aborted.")
				} else {
					fmt.Fprintln(os.Stderr, "Error getting password:", err)
				}
				os.Exit(1)
			}
			if password == "" {
				fmt.Fprintln(os.Stderr, "Error: Password cannot be empty")
				os.Exit(1)
			}
			fmt.Print("Confirm password: ")
			confirm, err := getPassword()
			if err != nil {
				if errors.Is(err, errAborted) {
					fmt.Println("Aborted.")
				} else {
					fmt.Fprintln(os.Stderr, "Error getting password confirmation:", err)
				}
				os.Exit(1)
			}
			if password != confirm {
				fmt.Fprintln(os.Stderr, "Error: Passwords do not match")
				os.Exit(1)
			}
		}
		fatalEv = initCommandLine()

		staff, err := gcsql.NewStaff(newstaff, password, rank)
		if err != nil {
			fatalAndLog("Error creating new staff account:", err, fatalEv.Str("source", "commandLine").Str("username", newstaff))
		}
		printInfoAndLog("New staff account created successfully")
		gcutil.LogInfo().
			Str("source", "commandLine").
			Str("username", newstaff).
			Msg("New staff account created")
		fmt.Printf("New staff account %q created with rank %s\n", newstaff, staff.RankTitle())
	case "delstaff":
		var force bool
		flagSet := flag.NewFlagSet("delstaff", flag.ExitOnError)
		flagSet.StringVar(&delstaff, "username", "", "Username of the staff account to delete")
		flagSet.BoolVar(&force, "force", false, "Force deletion without confirmation")
		flagSet.Parse(os.Args[2:])
		if delstaff == "" {
			fmt.Fprintln(os.Stderr, "Error: -username is required")
			flagSet.Usage()
			os.Exit(1)
		}
		if !force {
			fmt.Printf("Are you sure you want to delete the staff account %q? [y/N]: ", delstaff)
			var answer string
			fmt.Scanln(&answer)
			answer = strings.ToLower(answer)
			if answer != "y" && answer != "yes" {
				fmt.Println("Not deleting.")
				return
			}
		}
		fatalEv = initCommandLine()
		if err = gcsql.DeactivateStaff(delstaff); err != nil {
			fatalAndLog("Error deleting staff account:", err, fatalEv.Str("source", "commandLine").Str("username", delstaff))
		}
		printInfoAndLog("Staff account deleted successfully", gcutil.LogInfo().Str("source", "commandLine").Str("username", delstaff))
	case "rebuild":
		flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)
		var rebuildAll bool
		var rebuildBoards bool
		var rebuildFront bool
		var rebuildJS bool
		flagSet.BoolVar(&rebuildBoards, "boards", false, "Rebuild boards and threads")
		flagSet.BoolVar(&rebuildFront, "front", false, "Rebuild front page")
		flagSet.BoolVar(&rebuildJS, "js", false, "Rebuild consts.js")
		flagSet.BoolVar(&rebuildAll, "all", false, "Rebuild all components (overrides other flags)")
		flagSet.Parse(os.Args[2:])
		var rebuildFlag int
		if rebuildAll {
			rebuildFlag = buildAll
		}
		if rebuildBoards {
			rebuildFlag |= buildBoards
		}
		if rebuildFront {
			rebuildFlag |= buildFront
		}
		if rebuildJS {
			rebuildFlag |= buildJS
		}
		if rebuildFlag == 0 {
			fmt.Fprintln(os.Stderr, "Error: At least one rebuild option is required")
			flagSet.Usage()
			os.Exit(1)
		}
		fatalEv = initCommandLine()
		startupRebuild(rebuildFlag, fatalEv)
	default:
		fmt.Fprintln(os.Stderr, "Unknown command:", cmd)
		fmt.Println("Run 'gochan help' for a list of commands.")
		os.Exit(1)
	}
}
