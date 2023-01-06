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

const (
	dirIsAFileStr = `unable to create %q, path exists and is a file`
	genericErrStr = `unable to create %q: %s`
	pathExistsStr = `unable to create %q, path already exists`
)

var (
	ErrNoBoardDir   = errors.New("board must have a directory before it is built")
	ErrNoBoardTitle = errors.New("board must have a title before it is built")
)

type boardJSON struct {
	Dir             string `json:"board"`
	Title           string `json:"title"`
	Subtitle        string `json:"meta_description"`
	MaxFilesize     int    `json:"max_filesize"`
	Locked          bool   `json:"is_archived"`
	BumpLimit       int    `json:"bump_limit"`
	ImageLimit      int    `json:"image_limit"`
	MaxCommentChars int    `json:"max_comment_chars"`
	MinCommentChars int    `json:"min_comment_chars"`

	Cooldowns config.BoardCooldowns `json:"cooldowns"`
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// BuildBoardPages builds the front pages for the given board, and returns any error it encountered.
func BuildBoardPages(board *gcsql.Board) error {
	errEv := gcutil.LogError(nil).
		Int("boardID", board.ID).
		Str("boardDir", board.Dir)
	defer errEv.Discard()
	err := gctemplates.InitTemplates("boardpage")
	if err != nil {
		errEv.Err(err).Caller().Msg("unable to initialize boardpage template")
		return err
	}
	var currentPageFile *os.File
	var stickiedThreads []gcsql.Thread
	var nonStickiedThreads []gcsql.Thread
	var catalog boardCatalog
	var catalogThreads []catalogThreadData

	threads, err := board.GetThreads(true, true)
	if err != nil {
		errEv.Err(err).
			Caller().Msg("Failed getting board threads")
		return fmt.Errorf("error getting OP posts for /%s/: %s", board.Dir, err.Error())
	}

	for _, thread := range threads {
		catalogThread := catalogThreadData{
			Locked: boolToInt(thread.Locked),
			Sticky: boolToInt(thread.Stickied),
		}
		errEv.Int("threadID", thread.ID)
		if catalogThread.Images, err = thread.GetReplyFileCount(); err != nil {
			errEv.Err(err).
				Caller().Msg("Failed getting file count")
			return err
		}

		var maxRepliesOnBoardPage int
		postCfg := config.GetBoardConfig(board.Dir).PostConfig
		if thread.Stickied {
			// If the thread is stickied, limit replies on the archive page to the
			// configured value for stickied threads.
			maxRepliesOnBoardPage = postCfg.StickyRepliesOnBoardPage
		} else {
			// Otherwise, limit the replies to the configured value for normal threads.
			maxRepliesOnBoardPage = postCfg.RepliesOnBoardPage
		}
		catalogThread.Replies, err = thread.GetReplyCount()
		if err != nil {
			errEv.Err(err).
				Caller().Msg("Failed getting reply count")
			return errors.New("Error getting reply count: " + err.Error())
		}

		catalogThread.Posts, err = getThreadPosts(&thread)
		if err != nil {
			errEv.Err(err).
				Caller().Msg("Failed getting replies")
			return errors.New("Failed getting replies: " + err.Error())
		}
		if len(catalogThread.Posts) == 0 {
			continue
		}
		if len(catalogThread.Posts) > maxRepliesOnBoardPage {
			op := catalogThread.Posts[0]
			replies := catalogThread.Posts[len(catalogThread.Posts)-maxRepliesOnBoardPage:]
			catalogThread.Posts = []Post{op}
			catalogThread.Posts = append(catalogThread.Posts, replies...)
		}
		catalogThread.uploads, err = thread.GetUploads()
		if err != nil {
			errEv.Err(err).
				Caller().Msg("Failed getting thread uploads")
			return errors.New("Failed getting thread uploads: " + err.Error())
		}

		var imagesOnBoardPage int
		for _, upload := range catalogThread.uploads {
			for _, post := range catalogThread.Posts {
				if post.ID == upload.PostID {
					imagesOnBoardPage++
				}
			}
		}
		if catalogThread.Replies > maxRepliesOnBoardPage {
			catalogThread.OmittedPosts = catalogThread.Replies - len(catalogThread.Posts)
			catalogThread.OmittedImages = len(catalogThread.uploads) - imagesOnBoardPage
		}
		catalogThread.OmittedPosts = catalogThread.Replies - len(catalogThread.Posts)

		// Add thread struct to appropriate list
		if thread.Stickied {
			stickiedThreads = append(stickiedThreads, thread)
		} else {
			nonStickiedThreads = append(nonStickiedThreads, thread)
		}
		catalogThreads = append(catalogThreads, catalogThread)
	}

	criticalCfg := config.GetSystemCriticalConfig()
	gcutil.DeleteMatchingFiles(path.Join(criticalCfg.DocumentRoot, board.Dir), "\\d.html$")
	// Order the threads, stickied threads first, then nonstickied threads.
	threads = append(stickiedThreads, nonStickiedThreads...)

	// If there are no posts on the board
	var boardPageFile *os.File
	boardConfig := config.GetBoardConfig(board.Dir)
	if len(threads) == 0 {
		catalog.currentPage = 1

		// Open 1.html for writing to the first page.
		boardPageFile, err = os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "1.html"),
			os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
		if err != nil {
			errEv.Err(err).Caller().
				Str("page", "board.html").
				Msg("Failed getting board page")
			return fmt.Errorf("failed opening /%s/board.html: %s", board.Dir, err.Error())
		}
		defer boardPageFile.Close()

		if err = config.TakeOwnershipOfFile(boardPageFile); err != nil {
			errEv.Err(err).Caller().
				Msg("Unable to take ownership of board.html")
			return fmt.Errorf("unable to take ownership of /%s/board.html: %s", board.Dir, err.Error())
		}
		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		captchaCfg := config.GetSiteConfig().Captcha
		if err = serverutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"boards":      gcsql.AllBoards,
			"sections":    gcsql.AllSections,
			"threads":     threads,
			"numPages":    1,
			"currentPage": 1,
			"board":       board,
			"boardConfig": boardConfig,
			"useCaptcha":  captchaCfg.UseCaptcha(),
			"captcha":     captchaCfg,
		}, boardPageFile, "text/html"); err != nil {
			errEv.Err(err).
				Str("page", "board.html").
				Caller().Msg("Failed building board")
			return fmt.Errorf("failed building /%s/: %s", board.Dir, err.Error())
		}
		return nil
	}

	// Create the archive pages.
	boardCfg := config.GetBoardConfig(board.Dir)
	catalog.fillPages(boardCfg.ThreadsPerPage, catalogThreads)

	// Create array of page wrapper objects, and open the file.
	var catalogPages boardCatalog

	// catalog JSON file is built with the pages because pages are recorded in the JSON file
	catalogJSONFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
	if err != nil {
		errEv.Err(err).Caller().
			Msg("Failed opening catalog.json")
		return fmt.Errorf("failed opening /%s/catalog.json: %s", board.Dir, err.Error())
	}
	defer catalogJSONFile.Close()

	if err = config.TakeOwnershipOfFile(catalogJSONFile); err != nil {
		errEv.Err(err).Caller().
			Msg("Unable to take ownership of catalog.json")
		return fmt.Errorf("unable to take ownership of /%s/catalog.json: %s", board.Dir, err.Error())
	}
	for _, page := range catalog.pages {
		catalog.currentPage++
		var currentPageFilepath string
		pageFilename := strconv.Itoa(catalog.currentPage) + ".html"
		currentPageFilepath = path.Join(criticalCfg.DocumentRoot, board.Dir, pageFilename)
		currentPageFile, err = os.OpenFile(currentPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
		if err != nil {
			errEv.Err(err).Caller().
				Str("page", pageFilename).
				Msg("Failed getting board page")
			continue
		}
		defer currentPageFile.Close()

		if err = config.TakeOwnershipOfFile(currentPageFile); err != nil {
			errEv.Err(err).Caller().
				Str("page", pageFilename).
				Msg("Unable to update file ownership")
			return errors.New("unable to set board page file ownership")
		}

		// Render the boardpage template
		captchaCfg := config.GetSiteConfig().Captcha
		if err = serverutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"boards":      gcsql.AllBoards,
			"sections":    gcsql.AllSections,
			"threads":     page.Threads,
			"numPages":    len(threads) / boardConfig.ThreadsPerPage,
			"currentPage": catalog.currentPage,
			"board":       board,
			"boardConfig": boardCfg,
			"useCaptcha":  captchaCfg.UseCaptcha(),
			"captcha":     captchaCfg,
		}, currentPageFile, "text/html"); err != nil {
			errEv.Err(err).
				Caller().Send()
			return fmt.Errorf("failed building /%s/ boardpage: %s", board.Dir, err.Error())
		}

		// Collect up threads for this page.
		page := catalogPage{}
		page.PageNum = catalog.currentPage
		catalogPages.pages = append(catalogPages.pages, page)
	}

	var catalogJSON []byte
	if catalogJSON, err = json.Marshal(catalog.pages); err != nil {
		errEv.Err(err).
			Caller().Send()
		return errors.New("failed to marshal to JSON: " + err.Error())
	}
	if _, err = catalogJSONFile.Write(catalogJSON); err != nil {
		errEv.Err(err).
			Caller().Msg("Failed writing catalog.json")
		return fmt.Errorf("failed writing /%s/catalog.json: %s", board.Dir, err.Error())
	}
	return nil
}

// BuildBoards builds the specified board IDs, or all boards if no arguments are passed
// it returns any errors that were encountered
func BuildBoards(verbose bool, which ...int) error {
	var boards []gcsql.Board
	var err error

	if which == nil {
		boards, err = gcsql.GetAllBoards(false)
		if err != nil {
			return err
		}
	} else {
		for _, boardID := range which {
			board, err := gcsql.GetBoardFromID(boardID)
			if err != nil {
				gcutil.LogError(err).
					Int("boardid", boardID).
					Caller().Msg("Unable to get board information")
				return fmt.Errorf("unable to get board information (ID: %d): %s", boardID, err.Error())
			}
			boards = append(boards, *board)
		}
	}
	if len(boards) == 0 {
		return nil
	}

	for _, board := range boards {
		if err = buildBoard(&board, true); err != nil {
			return err
		}
		if verbose {
			fmt.Printf("Built /%s/ successfully\n", board.Dir)
		}
	}
	return nil
}

// Build builds the board and its thread files
// if force is true, it doesn't fail if the directories exist but does fail if it is a file
func buildBoard(board *gcsql.Board, force bool) error {
	var err error
	errEv := gcutil.LogError(nil).
		Str("boardDir", board.Dir).
		Int("boardID", board.ID)
	if board.Dir == "" {
		errEv.Err(err).Caller().Send()
		return ErrNoBoardDir
	}
	if board.Title == "" {
		errEv.Err(err).Caller().Send()
		return ErrNoBoardTitle
	}

	dirPath := board.AbsolutePath()
	resPath := board.AbsolutePath("res")
	srcPath := board.AbsolutePath("src")
	thumbPath := board.AbsolutePath("thumb")
	dirInfo, _ := os.Stat(dirPath)
	resInfo, _ := os.Stat(resPath)
	srcInfo, _ := os.Stat(srcPath)
	thumbInfo, _ := os.Stat(thumbPath)
	if dirInfo != nil {
		if !force {
			errEv.Err(os.ErrExist).
				Str("dirPath", dirPath).
				Caller().Send()
			return fmt.Errorf(pathExistsStr, dirPath)
		}
		if !dirInfo.IsDir() {
			errEv.Err(os.ErrExist).
				Str("dirPath", dirPath).
				Caller().Send()
			return fmt.Errorf(dirIsAFileStr, dirPath)
		}
	} else if err = os.Mkdir(dirPath, config.GC_DIR_MODE); err != nil {
		errEv.Err(os.ErrExist).
			Str("dirPath", dirPath).
			Caller().Send()
		return fmt.Errorf(genericErrStr, dirPath, err.Error())
	}
	if err = config.TakeOwnership(dirPath); err != nil {
		errEv.Err(err).Caller().
			Str("dirPath", dirPath).Send()
		return fmt.Errorf(genericErrStr, dirPath, err.Error())
	}

	if resInfo != nil {
		if !force {
			err = fmt.Errorf(pathExistsStr, resPath)
			errEv.Err(err).
				Str("resPath", resPath).
				Caller().Send()
			return err
		}
		if !resInfo.IsDir() {
			err = fmt.Errorf(dirIsAFileStr, resPath)
			errEv.Err(err).
				Str("resPath", resPath).
				Caller().Send()
			return err
		}
	} else if err = os.Mkdir(resPath, config.GC_DIR_MODE); err != nil {
		err = fmt.Errorf(genericErrStr, resPath, err.Error())
		errEv.Err(err).
			Str("resPath", resPath).
			Caller().Send()
		return fmt.Errorf(genericErrStr, resPath, err.Error())
	}
	if err = config.TakeOwnership(resPath); err != nil {
		errEv.Err(err).Caller().
			Str("resPath", resPath).Send()
		return fmt.Errorf(genericErrStr, resPath, err.Error())
	}

	if srcInfo != nil {
		if !force {
			err = fmt.Errorf(pathExistsStr, srcPath)
			errEv.Err(err).
				Str("srcPath", srcPath).
				Caller().Send()
			return err
		}
		if !srcInfo.IsDir() {
			err = fmt.Errorf(dirIsAFileStr, srcPath)
			errEv.Err(err).
				Str("srcPath", srcPath).
				Caller().Send()
			return err
		}
	} else if err = os.Mkdir(srcPath, config.GC_DIR_MODE); err != nil {
		err = fmt.Errorf(genericErrStr, srcPath, err.Error())
		errEv.Err(err).
			Str("srcPath", srcPath).
			Caller().Send()
		return err
	}
	if config.TakeOwnership(srcPath); err != nil {
		errEv.Err(err).Caller().
			Str("srcPath", srcPath).Send()
		return fmt.Errorf(genericErrStr, srcPath, err.Error())
	}

	if thumbInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, thumbPath)
		}
		if !thumbInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, thumbPath)
		}
	} else if err = os.Mkdir(thumbPath, config.GC_DIR_MODE); err != nil {
		errEv.Err(err).Caller().
			Str("thumbPath", thumbPath).Send()
		return fmt.Errorf(genericErrStr, thumbPath, err.Error())
	}
	if config.TakeOwnership(thumbPath); err != nil {
		errEv.Err(err).Caller().
			Str("thumbPath", thumbPath).Send()
		return fmt.Errorf(genericErrStr, thumbPath, err.Error())
	}

	if err = BuildBoardPages(board); err != nil {
		return err
	}
	if err = BuildThreads(true, board.ID, 0); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if err = gcsql.ResetBoardSectionArrays(); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if err = BuildFrontPage(); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if board.EnableCatalog {
		if err = BuildCatalog(board.ID); err != nil {
			return err
		}
	}
	if err = BuildBoardListJSON(); err != nil {
		return err
	}
	return nil
}

// BuildBoardListJSON generates a JSON file with info about the boards
func BuildBoardListJSON() error {
	boardsJsonPath := path.Join(config.GetSystemCriticalConfig().DocumentRoot, "boards.json")
	boardListFile, err := os.OpenFile(boardsJsonPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.GC_FILE_MODE)
	errEv := gcutil.LogError(nil).Str("building", "boards.json")
	defer errEv.Discard()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("unable to open boards.json for writing: " + err.Error())
	}
	defer boardListFile.Close()

	if err = config.TakeOwnershipOfFile(boardListFile); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("unable to update boards.json ownership: " + err.Error())
	}

	boardsMap := map[string][]boardJSON{
		"boards": {},
	}
	for _, board := range gcsql.AllBoards {
		boardsMap["boards"] = append(boardsMap["boards"], boardJSON{
			Dir:             board.Dir,
			Title:           board.Title,
			Subtitle:        board.Subtitle,
			MaxFilesize:     board.MaxFilesize,
			Locked:          board.Locked,
			BumpLimit:       board.AutosageAfter,
			ImageLimit:      board.NoImagesAfter,
			MaxCommentChars: board.MaxMessageLength,
			MinCommentChars: board.MinMessageLength,
			Cooldowns:       config.GetBoardConfig(board.Dir).Cooldowns,
		})
	}

	// TODO: properly check if the board is in a hidden section
	boardJSON, err := json.Marshal(boardsMap)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed to create boards.json: " + err.Error())
	}

	if _, err = serverutil.MinifyWriter(boardListFile, boardJSON, "application/json"); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed writing boards.json file: " + err.Error())
	}
	return nil
}
