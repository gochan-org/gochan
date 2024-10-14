package building

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
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
	*gcsql.Board
	Cooldowns config.BoardCooldowns `json:"cooldowns"`
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// BuildBoardPages builds the front pages for the given board, and returns any error it encountered.
func BuildBoardPages(board *gcsql.Board, errEv *zerolog.Event) error {
	if errEv == nil {
		errEv = gcutil.LogError(nil).
			Int("boardID", board.ID).
			Str("boardDir", board.Dir)
		defer errEv.Discard()
	}
	err := gctemplates.InitTemplates(gctemplates.BoardPage)
	if err != nil {
		errEv.Err(err).Caller().Msg("unable to initialize boardpage template")
		return err
	}
	var currentPageFile *os.File
	var catalog boardCatalog
	var catalogThreads []catalogThreadData

	threads, err := board.GetThreads(true, true, true)
	if err != nil {
		errEv.Err(err).Caller().
			Msg("Failed getting board threads")
		return fmt.Errorf("error getting threads for /%s/: %s", board.Dir, err.Error())
	}
	topPosts, err := getBoardTopPosts(board.Dir)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed getting board threads")
		return fmt.Errorf("error getting OP posts for /%s/: %s", board.Dir, err.Error())
	}
	opMap := make(map[int]*Post)
	for _, post := range topPosts {
		post.ParentID = 0
		opMap[post.thread.ID] = post
	}

	postCfg := config.GetBoardConfig(board.Dir).PostConfig
	for _, thread := range threads {
		catalogThread := catalogThreadData{
			Post:     opMap[thread.ID],
			Locked:   boolToInt(thread.Locked),
			Stickied: boolToInt(thread.Stickied),
		}
		errEv.Int("threadID", thread.ID)
		if catalogThread.Images, err = thread.GetReplyFileCount(); err != nil {
			errEv.Err(err).Caller().
				Msg("Failed getting file count")
			return err
		}

		var maxRepliesOnBoardPage int
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
			errEv.Err(err).Caller().Msg("Failed getting reply count")
			return errors.New("Error getting reply count: " + err.Error())
		}

		catalogThread.Posts, err = getThreadPosts(&thread)
		if err != nil {
			errEv.Err(err).Caller().Msg("Failed getting replies")
			return errors.New("Failed getting replies: " + err.Error())
		}
		if len(catalogThread.Posts) == 0 {
			continue
		}
		if len(catalogThread.Posts) > maxRepliesOnBoardPage {
			op := catalogThread.Posts[0]
			replies := catalogThread.Posts[len(catalogThread.Posts)-maxRepliesOnBoardPage:]
			catalogThread.Posts = []*Post{op}
			catalogThread.Posts = append(catalogThread.Posts, replies...)
		}
		catalogThread.uploads, err = thread.GetUploads()
		if err != nil {
			errEv.Err(err).Caller().Msg("Failed getting thread uploads")
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
			catalogThread.OmittedPosts = catalogThread.Replies - len(catalogThread.Posts) + 1
			catalogThread.OmittedImages = len(catalogThread.uploads) - imagesOnBoardPage
		}

		catalogThreads = append(catalogThreads, catalogThread)
	}

	criticalCfg := config.GetSystemCriticalConfig()
	gcutil.DeleteMatchingFiles(path.Join(criticalCfg.DocumentRoot, board.Dir), "\\d.html$")

	// If there are no posts on the board
	var boardPageFile *os.File
	boardConfig := config.GetBoardConfig(board.Dir)
	if len(threads) == 0 {
		catalog.currentPage = 1

		// Open 1.html for writing to the first page.
		boardPageFile, err = os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "1.html"),
			os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
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
			errEv.Err(err).Caller().
				Str("page", "board.html").
				Msg("Failed building board")
			return fmt.Errorf("failed building /%s/: %s", board.Dir, err.Error())
		}

		if err = boardPageFile.Close(); err != nil {
			errEv.Err(err).Caller().
				Str("page", "board.html").
				Msg("Unable to close board file")
			return err
		}
		return nil
	}

	// Create the archive pages.
	catalog.fillPages(boardConfig.ThreadsPerPage, catalogThreads)

	// Create array of page wrapper objects, and open the file.
	var catalogPages boardCatalog

	// catalog JSON file is built with the pages because pages are recorded in the JSON file
	catalogJSONFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, board.Dir, "catalog.json"),
		os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
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
		currentPageFile, err = os.OpenFile(currentPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
		if err != nil {
			errEv.Err(err).Caller().
				Str("page", pageFilename).
				Msg("Failed getting board page")
			continue
		}

		if err = config.TakeOwnershipOfFile(currentPageFile); err != nil {
			errEv.Err(err).Caller().
				Str("page", pageFilename).
				Msg("Unable to update file ownership")
			return errors.New("unable to set board page file ownership")
		}

		// Render the boardpage template
		captchaCfg := config.GetSiteConfig().Captcha
		numThreads := len(threads)
		numPages := numThreads / boardConfig.ThreadsPerPage
		if (numThreads % boardConfig.ThreadsPerPage) > 0 {
			numPages++
		}
		data := map[string]interface{}{
			"boards":      gcsql.AllBoards,
			"sections":    gcsql.AllSections,
			"threads":     page.Threads,
			"numPages":    numPages,
			"currentPage": catalog.currentPage,
			"board":       board,
			"boardConfig": boardConfig,
			"useCaptcha":  captchaCfg.UseCaptcha(),
			"captcha":     captchaCfg,
		}
		if catalog.currentPage > 1 {
			data["prevPage"] = catalog.currentPage - 1
		}
		if catalog.currentPage < numPages {
			data["nextPage"] = catalog.currentPage + 1
		}
		if err = serverutil.MinifyTemplate(gctemplates.BoardPage, data, currentPageFile, "text/html"); err != nil {
			errEv.Err(err).Caller().Send()
			return fmt.Errorf("failed building /%s/ boardpage: %s", board.Dir, err.Error())
		}
		if err = currentPageFile.Close(); err != nil {
			errEv.Err(err).Caller().Send()
			return fmt.Errorf("failed building /%s/ board page", board.Dir)
		}

		// Collect up threads for this page.
		page := catalogPage{}
		page.PageNum = catalog.currentPage
		catalogPages.pages = append(catalogPages.pages, page)
	}

	if err = json.NewEncoder(catalogJSONFile).Encode(catalog.pages); err != nil {
		errEv.Err(err).Caller().Msg("Unable to write catalog JSON to file")
		return errors.New("failed to marshal to catalog JSON")
	}
	if err = catalogJSONFile.Close(); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	return nil
}

// BuildBoards builds the specified board IDs, or all boards if no arguments are passed
// it returns any errors that were encountered
func BuildBoards(verbose bool, which ...int) error {
	var boards []gcsql.Board
	var err error
	errEv := gcutil.LogError(nil)
	defer errEv.Discard()

	if which == nil {
		boards, err = gcsql.GetAllBoards(false)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	} else {
		for _, boardID := range which {
			board, err := gcsql.GetBoardFromID(boardID)
			if err != nil {
				errEv.Err(err).Caller().
					Int("boardid", boardID).
					Msg("Unable to get board information")
				return fmt.Errorf("unable to get board information (ID: %d): %s", boardID, err.Error())
			}
			boards = append(boards, *board)
		}
	}
	if len(boards) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var tmpErr error
	wg.Add(len(boards))
	for b := range boards {
		go func(board *gcsql.Board) {
			tmpErr = buildBoard(board, true)
			if tmpErr == nil && verbose {
				gcutil.LogInfo().Str("board", board.Dir).
					Msg("Built board successfully")
			}
			if err == nil && tmpErr != nil {
				gcutil.LogError(err).Caller().Str("board", board.Dir).Msg("Unable to build board")
				err = tmpErr
			}
			wg.Done()
		}(&boards[b])
	}
	wg.Wait()
	return err
}

// Build builds the board and its thread files
// if force is true, it doesn't fail if the directories exist but does fail if it is a file
func buildBoard(board *gcsql.Board, force bool) error {
	var err error
	errEv := gcutil.LogError(nil).
		Str("boardDir", board.Dir).
		Int("boardID", board.ID)
	defer errEv.Discard()
	if board.Dir == "" {
		errEv.Err(ErrNoBoardDir).Caller().Send()
		return ErrNoBoardDir
	}
	if board.Title == "" {
		errEv.Err(ErrNoBoardTitle).Caller().Send()
		return ErrNoBoardTitle
	}

	oldPosts, err := board.DeleteOldThreads()
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to delete old threads")
		return err
	}
	boardDir := path.Join(config.GetSystemCriticalConfig().DocumentRoot, board.Dir)

	if err = os.MkdirAll(boardDir, config.DirFileMode); err != nil && !errors.Is(err, fs.ErrNotExist) {
		errEv.Err(err).Caller().Msg("Unable to create board directory")
		return err
	}

	for _, postID := range oldPosts {
		post, err := gcsql.GetPostFromID(postID, false)
		if err != nil {
			errEv.Err(err).Caller().
				Int("postID", postID).
				Msg("Unable to get post")
			return err
		}
		upload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().
				Int("postID", postID).
				Msg("Unable to get post uploads")
			return err
		}
		var filePath string
		if upload != nil {
			filePath = path.Join(boardDir, "src", upload.Filename)
			if err = os.Remove(filePath); err != nil {
				errEv.Err(err).Caller().
					Int("postID", postID).
					Str("upload", filePath).Send()
				return err
			}
			thumbPath, catalogThumbPath := uploads.GetThumbnailFilenames(
				path.Join(boardDir, "thumb", upload.Filename))
			if err = os.Remove(thumbPath); err != nil {
				errEv.Err(err).Caller().
					Int("postID", postID).
					Str("thumbnail", thumbPath).Send()
				return err
			}
			if post.IsTopPost && board.EnableCatalog {
				if err = os.Remove(catalogThumbPath); err != nil {
					errEv.Err(err).Caller().
						Int("postID", postID).
						Str("catalogThumbPath", catalogThumbPath).Send()
					return err
				}
			}
		}

		if err = post.UnlinkUploads(false); err != nil {
			errEv.Err(err).Caller().
				Int("postID", postID).Send()
			return err
		}
		if post.IsTopPost {
			filePath = path.Join(boardDir, "res", strconv.Itoa(post.ID)+".html")
			if err = os.Remove(filePath); err != nil {
				errEv.Err(err).Caller().
					Int("postID", postID).
					Str("threadFile", filePath).Send()
				return err
			}
		}
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
			errEv.Err(os.ErrExist).Caller().
				Str("dirPath", dirPath).Send()
			return fmt.Errorf(pathExistsStr, dirPath)
		}
		if !dirInfo.IsDir() {
			errEv.Err(os.ErrExist).Caller().
				Str("dirPath", dirPath).Send()
			return fmt.Errorf(dirIsAFileStr, dirPath)
		}
	} else if err = os.Mkdir(dirPath, config.DirFileMode); err != nil {
		errEv.Err(os.ErrExist).Caller().
			Str("dirPath", dirPath).Send()
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
			errEv.Err(err).Caller().
				Str("resPath", resPath).Send()
			return err
		}
		if !resInfo.IsDir() {
			err = fmt.Errorf(dirIsAFileStr, resPath)
			errEv.Err(err).Caller().
				Str("resPath", resPath).Send()
			return err
		}
	} else if err = os.Mkdir(resPath, config.DirFileMode); err != nil {
		err = fmt.Errorf(genericErrStr, resPath, err.Error())
		errEv.Err(err).Caller().
			Str("resPath", resPath).Send()
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
			errEv.Err(err).Caller().
				Str("srcPath", srcPath).Send()
			return err
		}
		if !srcInfo.IsDir() {
			err = fmt.Errorf(dirIsAFileStr, srcPath)
			errEv.Err(err).Caller().
				Str("srcPath", srcPath).Send()
			return err
		}
	} else if err = os.Mkdir(srcPath, config.DirFileMode); err != nil {
		err = fmt.Errorf(genericErrStr, srcPath, err.Error())
		errEv.Err(err).Caller().
			Str("srcPath", srcPath).Send()
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
	} else if err = os.Mkdir(thumbPath, config.DirFileMode); err != nil {
		errEv.Err(err).Caller().
			Str("thumbPath", thumbPath).Send()
		return fmt.Errorf(genericErrStr, thumbPath, err.Error())
	}
	if config.TakeOwnership(thumbPath); err != nil {
		errEv.Err(err).Caller().
			Str("thumbPath", thumbPath).Send()
		return fmt.Errorf(genericErrStr, thumbPath, err.Error())
	}

	if err = BuildBoardPages(board, errEv); err != nil {
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
			errEv.Err(err).Caller().Send()
			return err
		}
	}
	if err = BuildBoardListJSON(); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	return nil
}

type boardsListJSON struct {
	Boards []boardJSON `json:"boards"`
}

// BuildBoardListJSON generates a JSON file with info about the boards
func BuildBoardListJSON() error {
	boardsJsonPath := path.Join(config.GetSystemCriticalConfig().DocumentRoot, "boards.json")
	boardListFile, err := os.OpenFile(boardsJsonPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
	errEv := gcutil.LogError(nil).Str("building", "boards.json")
	defer errEv.Discard()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("unable to open boards.json for writing: " + err.Error())
	}

	if err = config.TakeOwnershipOfFile(boardListFile); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("unable to update boards.json ownership: " + err.Error())
	}

	boardsListJSONData := boardsListJSON{
		Boards: make([]boardJSON, len(gcsql.AllBoards)),
	}

	for b, board := range gcsql.AllBoards {
		boardsListJSONData.Boards[b] = boardJSON{
			Cooldowns: config.GetBoardConfig(board.Dir).Cooldowns,
		}
		boardsListJSONData.Boards[b].Board = &gcsql.AllBoards[b]
	}

	// TODO: properly check if the board is in a hidden section
	boardJSON, err := json.Marshal(boardsListJSONData)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("Failed to create boards.json " + err.Error())
	}

	if _, err = serverutil.MinifyWriter(boardListFile, boardJSON, "application/json"); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("failed writing boards.json file")
	}
	if err = boardListFile.Close(); err != nil {
		errEv.Err(err).Caller().Send()
		return errors.New("failed closing boards.json")
	}
	return nil
}
