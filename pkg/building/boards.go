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
	"github.com/gochan-org/gochan/pkg/serverutil"
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// BuildBoardPages builds the front pages for the given board, and returns any error it encountered.
func BuildBoardPages(board *gcsql.Board) error {
	err := gctemplates.InitTemplates("boardpage")
	if err != nil {
		return err
	}
	var currentPageFile *os.File
	var stickiedThreads []gcsql.Thread
	var nonStickiedThreads []gcsql.Thread
	var catalog boardCatalog
	var catalogThreads []catalogThreadData

	threads, err := board.GetThreads(true)
	if err != nil {
		gcutil.LogError(err).
			Int("boardID", board.ID).
			Caller().Msg("Failed getting board threads")
		return fmt.Errorf("error getting OP posts for /%s/: %s", board.Dir, err.Error())
	}

	for _, thread := range threads {
		catalogThread := catalogThreadData{
			Locked: boolToInt(thread.Locked),
			Sticky: boolToInt(thread.Stickied),
		}
		if catalogThread.Images, err = thread.GetReplyFileCount(); err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Int("threadID", thread.ID).
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
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Int("threadID", thread.ID).
				Caller().Msg("Failed getting reply count")
			return errors.New("Error getting reply count: " + err.Error())
		}
		catalogThread.posts, err = thread.GetPosts(false, true, maxRepliesOnBoardPage)
		if err != nil {
			gcutil.LogError(err).
				Int("threadid", thread.ID).
				Str("boardDir", board.Dir).
				Msg("Failed getting replies")
			return errors.New("Failed getting replies: " + err.Error())
		}
		catalogThread.uploads, err = thread.GetUploads()
		if err != nil {
			gcutil.LogError(err).
				Int("threadid", thread.ID).
				Str("boardDir", board.Dir).
				Caller().Msg("Failed getting thread uploads")
			return errors.New("Failed getting thread uploads: " + err.Error())
		}

		var imagesOnBoardPage int
		for _, upload := range catalogThread.uploads {
			for _, post := range catalogThread.posts {
				if post.ID == upload.PostID {
					imagesOnBoardPage++
				}
			}
		}
		if catalogThread.Replies > maxRepliesOnBoardPage {
			catalogThread.OmittedPosts = catalogThread.Replies - len(catalogThread.posts)
			catalogThread.OmittedImages = len(catalogThread.uploads) - imagesOnBoardPage
		}
		catalogThread.OmittedPosts = catalogThread.Replies - len(catalogThread.posts)

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
	if len(threads) == 0 {
		catalog.currentPage = 1

		// Open 1.html for writing to the first page.
		boardPageFile, err = os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "1.html"),
			os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Str("page", "board.html").
				Caller().Msg("Failed getting board page")
			return fmt.Errorf("failed opening /%s/board.html: %s", board.Dir, err.Error())
		}

		boardConfig := config.GetBoardConfig(board.Dir)
		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		if err = serverutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"webroot":      criticalCfg.WebRoot,
			"boards":       gcsql.AllBoards,
			"sections":     gcsql.AllSections,
			"threads":      threads,
			"numPages":     len(threads) / boardConfig.ThreadsPerPage,
			"board":        board,
			"board_config": boardConfig,
		}, boardPageFile, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
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

	catalogJSONFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		gcutil.LogError(err).
			Str("subject", "catalog.json").
			Str("boardDir", board.Dir).
			Msg("Failed opening catalog.json")
		return fmt.Errorf("failed opening /%s/catalog.json: %s", board.Dir, err.Error())
	}
	defer catalogJSONFile.Close()

	// currentBoardPage := catalog.currentPage
	for _, page := range catalog.pages {
		catalog.currentPage++
		var currentPageFilepath string
		pageFilename := strconv.Itoa(catalog.currentPage) + ".html"
		currentPageFilepath = path.Join(criticalCfg.DocumentRoot, board.Dir, pageFilename)
		currentPageFile, err = os.OpenFile(currentPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Str("page", pageFilename).
				Msg("Failed getting board page")
			continue
		}
		defer currentPageFile.Close()

		// Render the boardpage template
		if err = serverutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"webroot":      criticalCfg.WebRoot,
			"boards":       gcsql.AllBoards,
			"sections":     gcsql.AllSections,
			"threads":      page.Threads,
			"board":        board,
			"board_config": boardCfg,
			"posts": []interface{}{
				gcsql.Post{},
			},
		}, currentPageFile, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Msg("Failed building boardpage")
			return fmt.Errorf("Failed building /%s/ boardpage: %s", board.Dir, err.Error())
		}

		// Collect up threads for this page.
		page := catalogPage{}
		page.PageNum = catalog.currentPage
		// page.Threads = page.Threads
		catalogPages.pages = append(catalogPages.pages, page)
	}
	// board.CurrentPage = currentBoardPage

	var catalogJSON []byte
	if catalogJSON, err = json.Marshal(catalog.pages); err != nil {
		gcutil.LogError(err).
			Str("boardDir", board.Dir).
			Msg("Failed to marshal to JSON")
		return errors.New("failed to marshal to JSON: " + err.Error())
	}
	if _, err = catalogJSONFile.Write(catalogJSON); err != nil {
		gcutil.LogError(err).
			Str("boardDir", board.Dir).
			Msg("Failed writing catalog.json")
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
		boards = gcsql.AllBoards
	} else {
		for _, boardID := range which {
			board, err := gcsql.GetBoardFromID(boardID)
			if err != nil {
				gcutil.LogError(err).
					Int("boardid", boardID).
					Caller().Msg("Unable to get board information")
				return fmt.Errorf("Error getting board information (ID: %d): %s", boardID, err.Error())
			}
			boards = append(boards, *board)
		}
	}
	if len(boards) == 0 {
		return nil
	}

	for _, board := range boards {
		// board := &boards[b]
		if err = buildBoard(&board, true); err != nil {
			return err
		}
		if verbose {
			fmt.Printf("Built /%s/ successfully\n", board.Dir)
		}
	}
	return nil
}

// BuildCatalog builds the catalog for a board with a given id
func BuildCatalog(boardID int) error {
	errEv := gcutil.LogError(nil).
		Str("building", "catalog").
		Int("boardID", boardID)
	err := gctemplates.InitTemplates("catalog")
	if err != nil {
		errEv.Err(err).Send()
		return err
	}

	board, err := gcsql.GetBoardFromID(boardID)
	if err != nil {
		errEv.Err(err).
			Caller().Msg("Unable to get board information")
		return err
	}
	errEv.Str("boardDir", board.Dir)
	criticalCfg := config.GetSystemCriticalConfig()
	catalogPath := path.Join(criticalCfg.DocumentRoot, board.Dir, "catalog.html")
	catalogFile, err := os.OpenFile(catalogPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed opening /%s/catalog.html: %s<br/>", board.Dir, err.Error())
	}

	threadOPs, err := gcsql.GetBoardTopPosts(boardID)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed building catalog for /%s/: %s<br/>", board.Dir, err.Error())
	}
	boardConfig := config.GetBoardConfig(board.Dir)

	if err = serverutil.MinifyTemplate(gctemplates.Catalog, map[string]interface{}{
		"boards":       gcsql.AllBoards,
		"webroot":      criticalCfg.WebRoot,
		"board":        board,
		"board_config": boardConfig,
		"sections":     gcsql.AllSections,
		"threads":      threadOPs,
	}, catalogFile, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed building catalog for /%s/: %s<br/>", board.Dir, err.Error())
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
			return fmt.Errorf(dirIsAFileStr, dirPath)
		}
	} else if err = os.Mkdir(dirPath, 0666); err != nil {
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
	} else if err = os.Mkdir(resPath, 0666); err != nil {
		err = fmt.Errorf(genericErrStr, resPath, err.Error())
		errEv.Err(err).
			Str("resPath", resPath).
			Caller().Send()

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
	} else if err = os.Mkdir(srcPath, 0666); err != nil {
		err = fmt.Errorf(genericErrStr, srcPath, err.Error())
		errEv.Err(err).
			Str("srcPath", srcPath).
			Caller().Send()
		return err
	}

	if thumbInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, thumbPath)
		}
		if !thumbInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, thumbPath)
		}
	} else if err = os.Mkdir(thumbPath, 0666); err != nil {
		return fmt.Errorf(genericErrStr, thumbPath, err.Error())
	}

	if err = BuildBoardPages(board); err != nil {
		errEv.Err(err).Caller().Send()
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
		errEv.Caller().Send()
		if err = BuildCatalog(board.ID); err != nil {
			return err
		}
	}
	if err = BuildBoardListJSON(); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}

	return nil
}
