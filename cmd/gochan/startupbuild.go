package main

import (
	"fmt"
	"os"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

const (
	buildNone   = iota
	buildBoards = 1 << (iota - 1)
	buildFront
	buildJS
	buildAll = buildBoards | buildFront | buildJS
)

func startupRebuild(buildFlag int) {
	var err error
	serverutil.InitMinifier()
	if err = gctemplates.InitTemplates(); err != nil {
		fmt.Println("Error initializing templates:", err.Error())
		gcutil.Logger().Fatal().
			Str("building", "initialization").
			Err(err).Send()
	}

	if buildFlag&buildBoards > 0 {
		gcsql.ResetBoardSectionArrays()
		if err = building.BuildBoardListJSON(); err != nil {
			fmt.Println("Error building section array:", err.Error())
			gcutil.Logger().Fatal().
				Str("building", "sections").
				Err(err).Send()
		}

		if err = building.BuildBoards(true); err != nil {
			fmt.Println("Error building boards:", err.Error())
			gcutil.Logger().Fatal().
				Str("building", "boards").
				Err(err).Send()
		}
		fmt.Println("Boards built successfully")
	}

	if buildFlag&buildJS > 0 {
		if err = building.BuildJS(); err != nil {
			fmt.Println("Error building JS:", err.Error())
			gcutil.Logger().Fatal().
				Str("building", "js").
				Err(err).Send()
		}
		fmt.Println("JavaScript built successfully")
	}

	if buildFlag&buildFront > 0 {
		if err = building.BuildFrontPage(); err != nil {
			fmt.Println("Error building front page:", err.Error())
			gcutil.Logger().Fatal().
				Str("building", "front").
				Err(err).Send()
		}
		fmt.Println("Front page built successfully")
	}
	fmt.Println("Finished building without errors, exiting.")
	os.Exit(0)
}
