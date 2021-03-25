package building

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"strconv"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

// BuildThreads builds thread(s) given a boardid, or if all = false, also given a threadid.
// if all is set to true, ignore which, otherwise, which = build only specified boardid
// TODO: make it variadic
func BuildThreads(all bool, boardid, threadid int) error {
	var threads []gcsql.Post
	var err error
	if all {
		threads, err = gcsql.GetTopPostsNoSort(boardid)
	} else {
		var post gcsql.Post
		post, err = gcsql.GetSpecificTopPost(threadid)
		threads = []gcsql.Post{post}
	}
	if err != nil {
		return err
	}

	for _, op := range threads {
		if err = BuildThreadPages(&op); err != nil {
			return err
		}
	}
	return nil
}

// BuildThreadPages builds the pages for a thread given by a Post object.
func BuildThreadPages(op *gcsql.Post) error {
	err := gctemplates.InitTemplates("threadpage")
	if err != nil {
		return err
	}

	var replies []gcsql.Post
	var threadPageFile *os.File
	var board gcsql.Board
	if err = board.PopulateData(op.BoardID); err != nil {
		return err
	}

	replies, err = gcsql.GetExistingReplies(op.ID)
	if err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Error building thread %d: %s", op.ID, err.Error()))
	}
	os.Remove(path.Join(config.Config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html"))

	var repliesInterface []interface{}
	for _, reply := range replies {
		repliesInterface = append(repliesInterface, reply)
	}

	threadPageFilepath := path.Join(config.Config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html")
	threadPageFile, err = os.OpenFile(threadPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Failed opening /%s/res/%d.html: %s", board.Dir, op.ID, err.Error()))
	}

	// render thread page
	if err = serverutil.MinifyTemplate(gctemplates.ThreadPage, map[string]interface{}{
		"config":   config.Config,
		"boards":   gcsql.AllBoards,
		"board":    board,
		"sections": gcsql.AllSections,
		"posts":    replies,
		"op":       op,
	}, threadPageFile, "text/html"); err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Failed building /%s/res/%d threadpage: %s", board.Dir, op.ID, err.Error()))
	}

	// Put together the thread JSON
	threadJSONFile, err := os.OpenFile(path.Join(config.Config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Failed opening /%s/res/%d.json: %s", board.Dir, op.ID, err.Error()))
	}
	defer threadJSONFile.Close()

	threadMap := make(map[string][]gcsql.Post)

	// Handle the OP, of type *Post
	threadMap["posts"] = []gcsql.Post{*op}

	// Iterate through each reply, which are of type Post
	threadMap["posts"] = append(threadMap["posts"], replies...)
	threadJSON, err := json.Marshal(threadMap)
	if err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Failed to marshal to JSON: %s", err.Error()))
	}

	if _, err = threadJSONFile.Write(threadJSON); err != nil {
		return errors.New(gclog.Printf(gclog.LErrorLog,
			"Failed writing /%s/res/%d.json: %s", board.Dir, op.ID, err.Error()))
	}

	return nil
}
