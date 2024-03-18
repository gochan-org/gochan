package building

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

// BuildThreads builds thread(s) given a boardid, or if all = false, also given a threadid.
// if all is set to true, ignore which, otherwise, which = build only specified boardid
// TODO: make it variadic
func BuildThreads(all bool, boardid, threadid int) error {
	var threads []gcsql.Post
	var err error
	if all {
		threads, err = gcsql.GetBoardTopPosts(boardid)
	} else {
		var post *gcsql.Post
		post, err = gcsql.GetThreadTopPost(threadid)
		threads = []gcsql.Post{*post}
	}
	if err != nil {
		return err
	}

	for t := range threads {
		op := &threads[t]
		if err = BuildThreadPages(op); err != nil {
			return err
		}
	}
	return nil
}

// BuildThreadPages builds the pages for a thread given the top post. It fails if op is not the top post
func BuildThreadPages(op *gcsql.Post) error {
	errEv := gcutil.LogError(nil).
		Str("building", "thread").
		Int("postid", op.ID).
		Int("threadid", op.ThreadID)
	defer errEv.Discard()
	if !op.IsTopPost {
		errEv.Caller().Msg("non-OP passed to BuildThreadPages")
		return gcsql.ErrNotTopPost
	}
	err := gctemplates.InitTemplates(gctemplates.ThreadPage)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	var threadPageFile *os.File

	board, err := op.GetBoard()
	if err != nil {
		errEv.Err(err).Caller().Msg("failed building thread")
		return errors.New("failed building thread")
	}
	errEv.Str("boardDir", board.Dir)
	thread, err := gcsql.GetThread(op.ThreadID)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get thread info")
		return errors.New("unable to get thread info")
	}

	posts, err := getThreadPosts(thread)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("failed getting thread posts")
	}
	criticalCfg := config.GetSystemCriticalConfig()
	os.Remove(path.Join(criticalCfg.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html"))
	os.Remove(path.Join(criticalCfg.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".json"))

	threadPageFilepath := path.Join(criticalCfg.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html")
	threadPageFile, err = os.OpenFile(threadPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("unable to open /%s/res/%d.html: %s", board.Dir, op.ID, err.Error())
	}

	if err = config.TakeOwnershipOfFile(threadPageFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("unable to set file permissions for /%s/res/%d.html: %s", board.Dir, op.ID, err.Error())
	}
	errEv.Int("op", posts[0].ID)

	// render thread page
	captchaCfg := config.GetSiteConfig().Captcha
	if err = serverutil.MinifyTemplate(gctemplates.ThreadPage, map[string]interface{}{
		"boards":      gcsql.AllBoards,
		"board":       board,
		"boardConfig": config.GetBoardConfig(board.Dir),
		"sections":    gcsql.AllSections,
		"posts":       posts[1:],
		"op":          posts[0],
		"thread":      thread,
		"useCaptcha":  captchaCfg.UseCaptcha() && !captchaCfg.OnlyNeededForThreads,
		"captcha":     captchaCfg,
	}, threadPageFile, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed building /%s/res/%d threadpage: %s", board.Dir, posts[0].ID, err.Error())
	}
	if err = threadPageFile.Close(); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("failed closing thread page file")
	}

	// Put together the thread JSON
	threadJSONFile, err := os.OpenFile(
		path.Join(criticalCfg.DocumentRoot, board.Dir, "res", strconv.Itoa(posts[0].ID)+".json"),
		os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed opening /%s/res/%d.json", board.Dir, posts[0].ID)
	}

	if err = config.TakeOwnershipOfFile(threadJSONFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed setting file permissions for /%s/res/%d.json", board.Dir, posts[0].ID)
	}

	threadMap := make(map[string][]Post)

	threadMap["posts"] = posts
	if err = json.NewEncoder(threadJSONFile).Encode(threadMap); err != nil {
		errEv.Err(err).Caller().
			Msg("Unable to write thread JSON file")
		return fmt.Errorf("failed writing /%s/res/%d.json", board.Dir, posts[0].ID)
	}
	return threadJSONFile.Close()
}
