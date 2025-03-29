package main

import (
	"os"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
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
		fatalEv.Err(err).Caller().
			Str("building", "initialization").
			Msg("Unable to initialize templates")
	}

	if buildFlag&buildBoards > 0 {
		gcsql.ResetBoardSectionArrays()
		if err = building.BuildBoardListJSON(); err != nil {
			fatalEv.Err(err).Caller().
				Str("building", "sections").
				Msg("Unable to build section array")
		}

		if err = building.BuildBoards(true); err != nil {
			fatalEv.Err(err).Caller().
				Str("building", "boards").
				Msg("Unable to build boards")
		}
		gcutil.LogInfo().Msg("Boards built successfully")
	}

	if buildFlag&buildJS > 0 {
		if err = building.BuildJS(); err != nil {
			fatalEv.Err(err).Caller().
				Str("building", "js").
				Msg("Unable to build consts.js")
		}
		gcutil.LogInfo().Msg("consts.js built successfully")
	}

	if buildFlag&buildFront > 0 {
		if err = building.BuildFrontPage(); err != nil {
			fatalEv.Err(err).Caller().
				Str("building", "front").
				Msg("Unable to build front page")
		}
		gcutil.LogInfo().Msg("Front page built successfully")
	}
	gcutil.LogInfo().Msg("Finished building without errors, exiting.")
	os.Exit(0)
}
