package main

import (
	"os"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	buildNone   = iota
	buildBoards = 1 << (iota - 1)
	buildFront
	buildJS
	buildAll      = buildBoards | buildFront | buildJS
	buildLogFlags = gclog.LErrorLog | gclog.LStdLog | gclog.LFatal
)

func startupRebuild(buildFlag int) {
	var err *gcutil.GcError
	gcutil.InitMinifier()
	if err = gctemplates.InitTemplates(); err != nil {
		gclog.Print(buildLogFlags, "Error initializing templates: ", err.Error())
	}

	if buildFlag&buildBoards > 0 {
		gcsql.ResetBoardSectionArrays()
		if err = building.BuildBoardListJSON(); err != nil {
			gclog.Print(buildLogFlags, "Error building section array: ", err.Error())
		}

		if err = building.BuildBoards(true); err != nil {
			gclog.Print(buildLogFlags, "Error building boards: ", err.Error())
		}
		gclog.Print(gclog.LStdLog, "Boards built successfully")
	}

	if buildFlag&buildJS > 0 {
		if err = building.BuildJS(); err != nil {
			gclog.Print(buildLogFlags, "Error building JS: ", err.Error())
		}
		gclog.Print(gclog.LStdLog, "JavaScript built successfully")
	}

	if buildFlag&buildFront > 0 {
		if err = building.BuildFrontPage(); err != nil {
			gclog.Print(buildLogFlags, "Error building front page: ", err.Error())
		}
		gclog.Print(gclog.LStdLog, "Front page built successfully")
	}
	gclog.Print(gclog.LStdLog, "Finished building without errors, exiting.")
	os.Exit(0)
}
