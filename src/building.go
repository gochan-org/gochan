// functions for post, thread, board, and page building

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"syscall"
	"text/template"

	"github.com/tdewolff/minify"
	minifyHTML "github.com/tdewolff/minify/html"
	minifyJS "github.com/tdewolff/minify/js"
	minifyJSON "github.com/tdewolff/minify/json"
)

var minifier *minify.M

func initMinifier() {
	if !config.MinifyHTML && !config.MinifyJS {
		return
	}
	minifier = minify.New()
	if config.MinifyHTML {
		minifier.AddFunc("text/html", minifyHTML.Minify)
	}
	if config.MinifyJS {
		minifier.AddFunc("text/javascript", minifyJS.Minify)
		minifier.AddFunc("application/json", minifyJSON.Minify)
	}
}

func canMinify(mediaType string) bool {
	return (mediaType == "text/html" && config.MinifyHTML) || ((mediaType == "application/json" || mediaType == "text/javascript") && config.MinifyJS)
}

func minifyTemplate(tmpl *template.Template, data interface{}, writer io.Writer, mediaType string) error {
	if !canMinify(mediaType) {
		return tmpl.Execute(writer, data)
	}

	minWriter := minifier.Writer(mediaType, writer)
	defer closeHandle(minWriter)
	return tmpl.Execute(minWriter, data)
}

func minifyWriter(writer io.Writer, data []byte, mediaType string) (int, error) {
	if !canMinify(mediaType) {
		return writer.Write(data)
	}

	minWriter := minifier.Writer(mediaType, writer)
	defer closeHandle(minWriter)
	return minWriter.Write(data)
}

// build front page using templates/front.html
func buildFrontPage() string {
	err := initTemplates("front")
	if err != nil {
		return gclog.Print(lErrorLog, "Error loading front page template: ", err.Error())
	}
	os.Remove(path.Join(config.DocumentRoot, "index.html"))
	frontFile, err := os.OpenFile(path.Join(config.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(frontFile)
	if err != nil {
		return gclog.Print(lErrorLog, "Failed opening front page for writing: ", err.Error()) + "<br />"
	}

	var recentPostsArr []RecentPost
	recentPostsArr, err = GetRecentPostsGlobal(config.MaxRecentPosts, !config.RecentPostsWithNoFile)
	if err == nil {
		return gclog.Print(lErrorLog, "Failed loading recent posts: "+err.Error()) + "<br />"
	}

	for _, board := range allBoards {
		if board.Section == 0 {
			board.Section = 1
		}
	}

	if err = minifyTemplate(frontPageTmpl, map[string]interface{}{
		"config":       config,
		"sections":     allSections,
		"boards":       allBoards,
		"recent_posts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		return gclog.Print(lErrorLog, "Failed executing front page template: "+err.Error()) + "<br />"
	}
	return "Front page rebuilt successfully."
}

func buildBoardListJSON() (html string) {
	boardListFile, err := os.OpenFile(path.Join(config.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(boardListFile)
	if err != nil {
		return gclog.Print(lErrorLog, "Failed opening boards.json for writing: ", err.Error()) + "<br />"
	}

	boardsMap := map[string][]Board{
		"boards": []Board{},
	}

	// Our cooldowns are site-wide currently.
	cooldowns := BoardCooldowns{NewThread: config.NewThreadDelay, Reply: config.ReplyDelay, ImageReply: config.ReplyDelay}

	for _, board := range allBoards {
		board.Cooldowns = cooldowns
		boardsMap["boards"] = append(boardsMap["boards"], board)
	}

	boardJSON, err := json.Marshal(boardsMap)
	if err != nil {
		return gclog.Print(lErrorLog, "Failed to create boards.json: ", err.Error()) + "<br />"
	}

	if _, err = minifyWriter(boardListFile, boardJSON, "application/json"); err != nil {
		return gclog.Print(lErrorLog, "Failed writing boards.json file: ", err.Error()) + "<br />"
	}
	return "Board list JSON rebuilt successfully.<br />"
}

// buildBoardPages builds the pages for the board archive.
// `board` is a Board object representing the board to build archive pages for.
// The return value is a string of HTML with debug information from the build process.
func buildBoardPages(board *Board) (html string) {
	err := initTemplates("boardpage")
	if err != nil {
		return err.Error()
	}
	var currentPageFile *os.File
	var threads []interface{}
	var threadPages [][]interface{}
	var stickiedThreads []interface{}
	var nonStickiedThreads []interface{}
	var opPosts []Post

	// Get all top level posts for the board.
	if opPosts, err = GetTopPosts(board.ID, true); err != nil {
		return html + gclog.Printf(lErrorLog, "Error getting OP posts for /%s/: %s", board.Dir, err.Error()) + "<br />"
	}

	// For each top level post, start building a Thread struct
	for _, op := range opPosts {
		var thread Thread
		var postsInThread []Post

		var replyCount, err = GetReplyCount(op.ID)
		if err == nil {
			return html + gclog.Printf(lErrorLog,
				"Error getting replies to /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />"
		}
		thread.NumReplies = replyCount

		fileCount, err := GetReplyFileCount(op.ID)
		if err == nil {
			return html + gclog.Printf(lErrorLog,
				"Error getting file count to /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />"
		}
		thread.NumImages = fileCount

		thread.OP = op

		var numRepliesOnBoardPage int

		if op.Stickied {
			// If the thread is stickied, limit replies on the archive page to the
			// configured value for stickied threads.
			numRepliesOnBoardPage = config.StickyRepliesOnBoardPage
		} else {
			// Otherwise, limit the replies to the configured value for normal threads.
			numRepliesOnBoardPage = config.RepliesOnBoardPage
		}

		postsInThread, err = GetExistingRepliesLimitedRev(op.ID, numRepliesOnBoardPage)
		if err != nil {
			return html + gclog.Printf(lErrorLog,
				"Error getting posts in /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />"
		}

		var reversedPosts []Post
		for i := len(postsInThread); i > 0; i-- {
			reversedPosts = append(reversedPosts, postsInThread[i-1])
		}

		if len(postsInThread) > 0 {
			// Store the posts to show on board page
			//thread.BoardReplies = postsInThread
			thread.BoardReplies = reversedPosts

			// Count number of images on board page
			imageCount := 0
			for _, reply := range postsInThread {
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

	deleteMatchingFiles(path.Join(config.DocumentRoot, board.Dir), "\\d.html$")
	// Order the threads, stickied threads first, then nonstickied threads.
	threads = append(stickiedThreads, nonStickiedThreads...)

	// If there are no posts on the board
	if len(threads) == 0 {
		board.CurrentPage = 1
		// Open board.html for writing to the first page.
		boardPageFile, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "board.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			return html + gclog.Printf(lErrorLog,
				"Failed opening /%s/board.html: %s",
				board.Dir, err.Error()) + "<br />"
		}

		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		if err = minifyTemplate(boardpageTmpl, map[string]interface{}{
			"config":   config,
			"boards":   allBoards,
			"sections": allSections,
			"threads":  threads,
			"board":    board,
		}, boardPageFile, "text/html"); err != nil {
			return html + gclog.Printf(lErrorLog,
				"Failed building /%s/: %s",
				board.Dir, err.Error()) + "<br />"
		}
		html += "/" + board.Dir + "/ built successfully.\n"
		return
	}

	// Create the archive pages.
	threadPages = paginate(config.ThreadsPerPage, threads)
	board.NumPages = len(threadPages)

	// Create array of page wrapper objects, and open the file.
	pagesArr := make([]map[string]interface{}, board.NumPages)

	catalogJSONFile, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(catalogJSONFile)
	if err != nil {
		return gclog.Printf(lErrorLog,
			"Failed opening /%s/catalog.json: %s",
			board.Dir, err.Error()) + "<br />"
	}

	currentBoardPage := board.CurrentPage
	for _, pageThreads := range threadPages {
		board.CurrentPage++
		var currentPageFilepath string
		pageFilename := strconv.Itoa(board.CurrentPage) + ".html"
		currentPageFilepath = path.Join(config.DocumentRoot, board.Dir, pageFilename)
		currentPageFile, err = os.OpenFile(currentPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		defer closeHandle(currentPageFile)
		if err != nil {
			html += gclog.Printf(lErrorLog,
				"Failed opening /%s/%s: %s",
				board.Dir, pageFilename, err.Error()) + "<br />"
			continue
		}

		// Render the boardpage template
		if err = minifyTemplate(boardpageTmpl, map[string]interface{}{
			"config":   config,
			"boards":   allBoards,
			"sections": allSections,
			"threads":  pageThreads,
			"board":    board,
			"posts": []interface{}{
				Post{BoardID: board.ID},
			},
		}, currentPageFile, "text/html"); err != nil {
			return html + gclog.Printf(lErrorLog,
				"Failed building /%s/ boardpage: %s", board.Dir, err.Error()) + "<br />"
		}

		if board.CurrentPage == 1 {
			boardPage := path.Join(config.DocumentRoot, board.Dir, "board.html")
			os.Remove(boardPage)
			if err = syscall.Symlink(currentPageFilepath, boardPage); !os.IsExist(err) && err != nil {
				html += gclog.Printf(lErrorLog, "Failed building /%s/: %s",
					board.Dir, err.Error())
			}
		}

		// Collect up threads for this page.
		pageMap := make(map[string]interface{})
		pageMap["page"] = board.CurrentPage
		pageMap["threads"] = pageThreads
		pagesArr = append(pagesArr, pageMap)
	}
	board.CurrentPage = currentBoardPage

	catalogJSON, err := json.Marshal(pagesArr)
	if err != nil {
		return html + gclog.Print(lErrorLog, "Failed to marshal to JSON: ", err.Error()) + "<br />"
	}
	if _, err = catalogJSONFile.Write(catalogJSON); err != nil {
		return html + gclog.Printf(lErrorLog,
			"Failed writing /%s/catalog.json: %s", board.Dir, err.Error()) + "<br />"
	}
	html += "/" + board.Dir + "/ built successfully."
	return
}

// buildBoards builds the specified board IDs, or all boards if no arguments are passed
// The return value is a string of HTML with debug information produced by the build process.
func buildBoards(which ...int) (html string) {
	var boards []Board
	var err error
	if which == nil {
		boards = allBoards
	} else {
		for b, id := range which {
			boards = append(boards, Board{})
			if err = boards[b].PopulateData(id, ""); err != nil {
				return gclog.Printf(lErrorLog, "Error getting board information (ID: %d)", id)
			}
		}
	}
	if len(boards) == 0 {
		return "No boards to build."
	}

	for _, board := range boards {
		if err = board.Build(false, true); err != nil {
			return gclog.Printf(lErrorLog,
				"Error building /%s/: %s", board.Dir, err.Error()) + "<br />"
		}
		html += "Built /" + board.Dir + "/ successfully."
	}
	return
}

func buildJS() string {
	// minify gochan.js (if enabled)
	gochanMinJSPath := path.Join(config.DocumentRoot, "javascript", "gochan.min.js")
	gochanMinJSFile, err := os.OpenFile(gochanMinJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer closeHandle(gochanMinJSFile)
	if err != nil {
		return gclog.Printf(lErrorLog, "Error opening %q for writing: %s",
			gochanMinJSPath, err.Error()) + "<br />"
	}
	gochanJSPath := path.Join(config.DocumentRoot, "javascript", "gochan.js")
	gochanJSBytes, err := ioutil.ReadFile(gochanJSPath)
	if err != nil {
		return gclog.Printf(lErrorLog, "Error opening %q for writing: %s",
			gochanJSPath, err.Error()) + "<br />"
	}
	if _, err := minifyWriter(gochanMinJSFile, gochanJSBytes, "text/javascript"); err != nil {
		config.UseMinifiedGochanJS = false
		return gclog.Printf(lErrorLog, "Error minifying %q: %s:",
			gochanMinJSPath, err.Error()) + "<br />"
	}
	config.UseMinifiedGochanJS = true

	// build consts.js from template
	if err = initTemplates("js"); err != nil {
		return gclog.Print(lErrorLog, "Error loading consts.js template: ", err.Error())
	}
	constsJSPath := path.Join(config.DocumentRoot, "javascript", "consts.js")
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer closeHandle(constsJSFile)
	if err != nil {
		return gclog.Printf(lErrorLog, "Error opening %q for writing: %s",
			constsJSPath, err.Error()) + "<br />"
	}

	if err = minifyTemplate(jsTmpl, config, constsJSFile, "text/javascript"); err != nil {
		return gclog.Printf(lErrorLog, "Error building %q: %s",
			constsJSPath, err.Error()) + "<br />"
	}
	return "Built gochan.min.js and consts.js successfully.<br />"
}

func buildCatalog(which int) string {
	err := initTemplates("catalog")
	if err != nil {
		return err.Error()
	}

	var board Board
	if err = board.PopulateData(which, ""); err != nil {
		return gclog.Printf(lErrorLog, "Error getting board information (ID: %d)", which)
	}

	catalogPath := path.Join(config.DocumentRoot, board.Dir, "catalog.html")
	catalogFile, err := os.OpenFile(catalogPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return gclog.Printf(lErrorLog,
			"Failed opening /%s/catalog.html: %s", board.Dir, err.Error()) + "<br />"
	}

	threadOPs, err := GetTopPosts(which, false)
	if err != nil {
		return gclog.Printf(lErrorLog,
			"Error building catalog for /%s/: %s", board.Dir, err.Error()) + "<br />"
	}

	var threadInterfaces []interface{}
	for _, thread := range threadOPs {
		threadInterfaces = append(threadInterfaces, thread)
	}

	if err = minifyTemplate(catalogTmpl, map[string]interface{}{
		"boards":   allBoards,
		"config":   config,
		"board":    board,
		"sections": allSections,
	}, catalogFile, "text/html"); err != nil {
		return gclog.Printf(lErrorLog,
			"Error building catalog for /%s/: %s", board.Dir, err.Error()) + "<br />"
	}
	return fmt.Sprintf("Built catalog for /%s/ successfully", board.Dir)
}

// buildThreadPages builds the pages for a thread given by a Post object.
func buildThreadPages(op *Post) error {
	err := initTemplates("threadpage")
	if err != nil {
		return err
	}

	var replies []Post
	var threadPageFile *os.File
	var board Board
	if err = board.PopulateData(op.BoardID, ""); err != nil {
		return err
	}

	replies, err = GetExistingReplies(op.ID)
	if err != nil {
		return fmt.Errorf("Error building thread %d: %s", op.ID, err.Error())
	}
	os.Remove(path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html"))

	var repliesInterface []interface{}
	for _, reply := range replies {
		repliesInterface = append(repliesInterface, reply)
	}

	threadPageFilepath := path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html")
	threadPageFile, err = os.OpenFile(threadPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return fmt.Errorf("Failed opening /%s/res/%d.html: %s", board.Dir, op.ID, err.Error())
	}

	// render thread page
	if err = minifyTemplate(threadpageTmpl, map[string]interface{}{
		"config":   config,
		"boards":   allBoards,
		"board":    board,
		"sections": allSections,
		"posts":    replies,
		"op":       op,
	}, threadPageFile, "text/html"); err != nil {
		return fmt.Errorf("Failed building /%s/res/%d threadpage: %s", board.Dir, op.ID, err.Error())
	}

	// Put together the thread JSON
	threadJSONFile, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(threadJSONFile)
	if err != nil {
		return fmt.Errorf("Failed opening /%s/res/%d.json: %s", board.Dir, op.ID, err.Error())
	}

	threadMap := make(map[string][]Post)

	// Handle the OP, of type *Post
	threadMap["posts"] = []Post{*op}

	// Iterate through each reply, which are of type Post
	threadMap["posts"] = append(threadMap["posts"], replies...)
	threadJSON, err := json.Marshal(threadMap)
	if err != nil {
		return fmt.Errorf("Failed to marshal to JSON: %s", err.Error())
	}

	if _, err = threadJSONFile.Write(threadJSON); err != nil {
		return fmt.Errorf("Failed writing /%s/res/%d.json: %s", board.Dir, op.ID, err.Error())
	}

	return nil
}

// buildThreads builds thread(s) given a boardid, or if all = false, also given a threadid.
// if all is set to true, ignore which, otherwise, which = build only specified boardid
// TODO: make it variadic
func buildThreads(all bool, boardid, threadid int) error {
	var threads []Post
	var err error
	if all {
		threads, err = GetTopPostsNoSort(boardid)
	} else {
		var post Post
		post, err = GetSpecificTopPost(threadid)
		threads = []Post{post}
	}
	if err != nil {
		return err
	}

	for _, op := range threads {
		if err = buildThreadPages(&op); err != nil {
			return err
		}
	}
	return nil
}
