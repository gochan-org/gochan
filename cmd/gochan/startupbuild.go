package main

import (
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

const (
	buildNone   = iota
	buildBoards = 1 << (iota - 1)
	buildFront
	buildJS
	buildAll = buildBoards | buildFront | buildJS
)

func startupRebuild(buildFlag int, fatalEv *zerolog.Event) {
	var err error
	serverutil.InitMinifier()
	if err = gctemplates.InitTemplates(); err != nil {
		fatalAndLog("Unable to initialize templates:", err, fatalEv.Str("building", "initialization"))
	}

	if buildFlag&buildBoards > 0 {
		if err = gcsql.ResetBoardSectionArrays(); err != nil {
			fatalAndLog("Unable to reset board section arrays:", err, fatalEv.Str("building", "reset"))
		}

		if err = building.BuildBoardListJSON(); err != nil {
			fatalAndLog("Unable to build board list JSON:", err, fatalEv.Str("building", "boardListJSON"))
		}
		printInfoAndLog("Board list JSON built successfully")

		if err = building.BuildBoards(true); err != nil {
			fatalAndLog("Unable to build boards:", err, fatalEv.Str("building", "boards"))
		}
		printInfoAndLog("Boards built successfully")
	}

	if buildFlag&buildJS > 0 {
		if err = building.BuildJS(); err != nil {
			fatalAndLog("Unable to build consts.js:", err, fatalEv.Str("building", "js"))
		}
		printInfoAndLog("consts.js built successfully")
	}

	if buildFlag&buildFront > 0 {
		if err = building.BuildFrontPage(); err != nil {
			fatalAndLog("Unable to build front page:", err, fatalEv.Str("building", "front"))
		}
		printInfoAndLog("Front page built successfully")
	}
	printInfoAndLog("Finished building without errors, exiting.")
}
