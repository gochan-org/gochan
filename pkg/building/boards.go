package building

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

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

// BuildBoardPages builds the pages for the board archive.
// `board` is a Board object representing the board to build archive pages for.
// The return value is a string of HTML with debug information from the build process.
func BuildBoardPages(board *gcsql.Board) error {
	err := gctemplates.InitTemplates("boardpage")
	if err != nil {
		return err
	}
	var currentPageFile *os.File
	var threadPages [][]interface{}
	var stickiedThreads []interface{}
	var nonStickiedThreads []interface{}
	var opPosts []gcsql.Post

	threads, err := gcsql.GetThreadsWithBoardID(board.ID, true)
	if err != nil {
		gcutil.LogError(err).
			Int("boardID", board.ID).
			Msg("Failed getting OP posts")
		return fmt.Errorf("error getting OP posts for /%s/: %s", board.Dir, err.Error())
	}

	// Get all top level posts for the board
	if opPosts, err = gcsql.GetTopPosts(board.ID); err != nil {
		gcutil.LogError(err).
			Str("boardDir", board.Dir).
			Msg("Failed getting OP posts")
		return fmt.Errorf("error getting OP posts for /%s/: %s", board.Dir, err.Error())
	}

	// For each top level post, start building a Thread struct
	for p := range opPosts {
		op := &opPosts[p]
		var thread gcsql.Thread
		var postsInThread []gcsql.Post

		var replyCount, err = gcsql.GetReplyCount(op.ID)
		if err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Int("op", op.ID).
				Msg("Failed getting thread replies")
			return fmt.Errorf("Error getting replies to /%s/%d: %s", board.Dir, op.ID, err.Error())
		}
		thread.NumReplies = replyCount

		fileCount, err := gcsql.GetReplyFileCount(op.ID)
		if err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Int("op", op.ID).
				Msg("Failed getting file count")
			return fmt.Errorf("Error getting file count to /%s/%d: %s", board.Dir, op.ID, err.Error())
		}
		thread.NumImages = fileCount

		thread.OP = *op

		var numRepliesOnBoardPage int
		postCfg := config.GetBoardConfig("").PostConfig
		if op.Stickied {
			// If the thread is stickied, limit replies on the archive page to the
			// configured value for stickied threads.
			numRepliesOnBoardPage = postCfg.StickyRepliesOnBoardPage
		} else {
			// Otherwise, limit the replies to the configured value for normal threads.
			numRepliesOnBoardPage = postCfg.RepliesOnBoardPage
		}

		postsInThread, err = gcsql.GetExistingRepliesLimitedRev(op.ID, numRepliesOnBoardPage)
		if err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Int("op", op.ID).
				Msg("Failed getting thread posts")
			return fmt.Errorf("Error getting posts in /%s/%d: %s", board.Dir, op.ID, err.Error())
		}

		var reversedPosts []gcsql.Post
		for i := len(postsInThread); i > 0; i-- {
			reversedPosts = append(reversedPosts, postsInThread[i-1])
		}

		if len(postsInThread) > 0 {
			// Store the posts to show on board page
			//thread.BoardReplies = postsInThread
			thread.BoardReplies = reversedPosts

			// Count number of images on board page
			imageCount := 0
			for p := range postsInThread {
				reply := &postsInThread[p]
				if reply.Filesize != 0 {
					imageCount++
				}
			}
			// Then calculate number of omitted images.
			thread.OmittedImages = thread.NumImages - imageCount
		}

		// Add thread struct to appropriate list
		if op.Stickied {
			stickiedThreads = append(stickiedThreads, thread)
		} else {
			nonStickiedThreads = append(nonStickiedThreads, thread)
		}
	}
	criticalCfg := config.GetSystemCriticalConfig()
	gcutil.DeleteMatchingFiles(path.Join(criticalCfg.DocumentRoot, board.Dir), "\\d.html$")
	// Order the threads, stickied threads first, then nonstickied threads.
	threads = append(stickiedThreads, nonStickiedThreads...)

	// If there are no posts on the board
	var boardPageFile *os.File
	if len(threads) == 0 {
		board.CurrentPage = 1

		// Open 1.html for writing to the first page.
		boardPageFile, err = os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "1.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Str("page", "board.html").
				Msg("Failed getting board page")
			return fmt.Errorf("failed opening /%s/board.html: %s", board.Dir, err.Error())
		}

		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		if err = serverutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"webroot":      criticalCfg.WebRoot,
			"boards":       gcsql.AllBoards,
			"sections":     gcsql.AllSections,
			"threads":      threads,
			"board":        board,
			"board_config": config.GetBoardConfig(board.Dir),
		}, boardPageFile, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Str("page", "board.html").
				Msg("Failed building board")
			return fmt.Errorf("Failed building /%s/: %s", board.Dir, err.Error())
		}
		return nil
	}

	// Create the archive pages.
	boardCfg := config.GetBoardConfig(board.Dir)
	threadPages = paginate(boardCfg.ThreadsPerPage, threads)
	board.NumPages = len(threadPages)

	// Create array of page wrapper objects, and open the file.
	var pagesArr boardCatalog

	catalogJSONFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		gcutil.LogError(err).
			Str("subject", "catalog.json").
			Str("boardDir", board.Dir).
			Msg("Failed opening catalog.json")
		return fmt.Errorf("failed opening /%s/catalog.json: %s", board.Dir, err.Error())
	}
	defer catalogJSONFile.Close()

	currentBoardPage := board.CurrentPage
	for _, pageThreads := range threadPages {
		board.CurrentPage++
		var currentPageFilepath string
		pageFilename := strconv.Itoa(board.CurrentPage) + ".html"
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
			"threads":      pageThreads,
			"board":        board,
			"board_config": config.GetBoardConfig(board.Dir),
			"posts": []interface{}{
				gcsql.Post{BoardID: board.ID},
			},
		}, currentPageFile, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Msg("Failed building boardpage")
			return fmt.Errorf("Failed building /%s/ boardpage: %s", board.Dir, err.Error())
		}

		// Collect up threads for this page.
		pageMap := make(map[string]interface{})
		pageMap["page"] = board.CurrentPage
		pageMap["threads"] = pageThreads
		pagesArr = append(pagesArr, pageMap)
	}
	board.CurrentPage = currentBoardPage

	var catalogJSON []byte
	if catalogJSON, err = json.Marshal(pagesArr); err != nil {
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
		for b, id := range which {
			boards = append(boards, gcsql.Board{})
			if err = boards[b].PopulateData(id); err != nil {
				gcutil.LogError(err).
					Int("boardid", id).
					Msg("Unable to get board information")
				return fmt.Errorf("Error getting board information (ID: %d): %s", id, err.Error())
			}
		}
	}
	if len(boards) == 0 {
		return nil
	}

	for b := range boards {
		board := &boards[b]
		if err = buildBoard(board, false, true); err != nil {
			gcutil.LogError(err).
				Str("boardDir", board.Dir).
				Msg("Failed building board")
			return fmt.Errorf("Error building /%s/: %s", board.Dir, err.Error())
		}
		if verbose {
			fmt.Printf("Built /%s/ successfully\n", board.Dir)
		}
	}
	return nil
}

// BuildCatalog builds the catalog for a board with a given id
func BuildCatalog(boardID int) string {
	err := gctemplates.InitTemplates("catalog")
	if err != nil {
		return err.Error()
	}

	var board gcsql.Board
	if err = board.PopulateData(boardID); err != nil {
		gcutil.LogError(err).
			Int("boardid", boardID).
			Msg("Unable to get board information")
		return fmt.Sprintf("Error getting board information (ID: %d)", boardID)
	}
	criticalCfg := config.GetSystemCriticalConfig()
	catalogPath := path.Join(criticalCfg.DocumentRoot, board.Dir, "catalog.html")
	catalogFile, err := os.OpenFile(catalogPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		gcutil.LogError(err).
			Str("boardDir", board.Dir).
			Msg("Failed opening catalog.html")
		return fmt.Sprintf("Failed opening /%s/catalog.html: %s<br/>", board.Dir, err.Error())
	}

	threadOPs, err := gcsql.GetTopPosts(boardID)
	// threadOPs, err := getPostArr(map[string]interface{}{
	// 	"boardid":           boardID,
	// 	"parentid":          0,
	// 	"deleted_timestamp": nilTimestamp,
	// }, "ORDER BY bumped ASC")
	if err != nil {
		gcutil.LogError(err).
			Str("building", "catalog").
			Str("boardDir", board.Dir).Send()
		return fmt.Sprintf("Error building catalog for /%s/: %s<br/>", board.Dir, err.Error())
	}

	if err = serverutil.MinifyTemplate(gctemplates.Catalog, map[string]interface{}{
		"boards":       gcsql.AllBoards,
		"webroot":      criticalCfg.WebRoot,
		"board":        board,
		"board_config": config.GetBoardConfig(board.Dir),
		"sections":     gcsql.AllSections,
		"threads":      threadOPs,
	}, catalogFile, "text/html"); err != nil {
		gcutil.LogError(err).
			Str("building", "catalog").
			Str("boardDir", board.Dir).Send()
		return fmt.Sprintf("Error building catalog for /%s/: %s<br/>", board.Dir, err.Error())
	}
	return fmt.Sprintf("Built catalog for /%s/ successfully", board.Dir)
}

// Build builds the board and its thread files
// if newBoard is true, it adds a row to DBPREFIXboards and fails if it exists
// if force is true, it doesn't fail if the directories exist but does fail if it is a file
func buildBoard(board *gcsql.Board, newBoard, force bool) error {
	var err error
	if board.Dir == "" {
		return ErrNoBoardDir
	}
	if board.Title == "" {
		return ErrNoBoardTitle
	}

	if newBoard {
		board.CreatedOn = time.Now()
		err := gcsql.CreateBoard(board)
		if err != nil {
			return err
		}
	} else if err = board.UpdateID(); err != nil {
		return err
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
			gcutil.LogError(os.ErrExist).
				Str("dirPath", dirPath).Send()
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
			return fmt.Errorf(pathExistsStr, resPath)
		}
		if !resInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, resPath)

		}
	} else if err = os.Mkdir(resPath, 0666); err != nil {
		return fmt.Errorf(genericErrStr, resPath, err.Error())
	}

	if srcInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, srcPath)
		}
		if !srcInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, srcPath)
		}
	} else if err = os.Mkdir(srcPath, 0666); err != nil {
		return fmt.Errorf(genericErrStr, srcPath, err.Error())
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

	BuildBoardPages(board)
	BuildThreads(true, board.ID, 0)
	gcsql.ResetBoardSectionArrays()
	BuildFrontPage()
	if board.EnableCatalog {
		BuildCatalog(board.ID)
	}
	BuildBoardListJSON()
	return nil
}
