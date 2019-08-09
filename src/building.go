// functions for post, thread, board, and page building

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"
)

// build front page using templates/front.html
func buildFrontPage() string {
	err := initTemplates("front")
	if err != nil {
		return err.Error()
	}
	var recentPostsArr []interface{}

	os.Remove(path.Join(config.DocumentRoot, "index.html"))
	front_file, err := os.OpenFile(path.Join(config.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(front_file)
	if err != nil {
		return handleError(1, "Failed opening front page for writing: "+err.Error()) + "<br />\n"
	}

	// get recent posts
	recentQueryStr := "SELECT " +
		config.DBprefix + "posts.id, " +
		config.DBprefix + "posts.parentid, " +
		config.DBprefix + "boards.dir as boardname, " +
		config.DBprefix + "posts.boardid as boardid, " +
		config.DBprefix + "posts.name, " +
		config.DBprefix + "posts.tripcode, " +
		config.DBprefix + "posts.message, " +
		config.DBprefix + "posts.filename, " +
		config.DBprefix + "posts.thumb_w, " +
		config.DBprefix + "posts.thumb_h " +
		"FROM " + config.DBprefix + "posts, " + config.DBprefix + "boards " +
		"WHERE " + config.DBprefix + "posts.deleted_timestamp = ? "

	if !config.RecentPostsWithNoFile {
		recentQueryStr += "AND " + config.DBprefix + "posts.filename != '' AND " + config.DBprefix + "posts.filename != 'deleted' "
	}
	recentQueryStr += "AND boardid = " + config.DBprefix + "boards.id " +
		"ORDER BY timestamp DESC LIMIT ?"

	rows, err := querySQL(recentQueryStr, nilTimestamp, config.MaxRecentPosts)
	defer closeHandle(rows)
	if err != nil {
		return handleError(1, err.Error())
	}

	for rows.Next() {
		recentPost := new(RecentPost)
		if err = rows.Scan(
			&recentPost.PostID, &recentPost.ParentID, &recentPost.BoardName, &recentPost.BoardID,
			&recentPost.Name, &recentPost.Tripcode, &recentPost.Message, &recentPost.Filename, &recentPost.ThumbW, &recentPost.ThumbH,
		); err != nil {
			return handleError(1, "Failed getting list of recent posts for front page: "+err.Error())
		}
		recentPostsArr = append(recentPostsArr, recentPost)
	}

	for _, board := range allBoards {
		if board.Section == 0 {
			board.Section = 1
		}
	}

	if err = front_page_tmpl.Execute(front_file, map[string]interface{}{
		"config":       config,
		"sections":     allSections,
		"boards":       allBoards,
		"recent_posts": recentPostsArr,
	}); err != nil {
		return handleError(1, "Failed executing front page template: "+err.Error())
	}
	return "Front page rebuilt successfully."
}

func buildBoardListJSON() (html string) {
	boardListFile, err := os.OpenFile(path.Join(config.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(boardListFile)
	if err != nil {
		return handleError(1, "Failed opening board.json for writing: "+err.Error()) + "<br />\n"
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
		return handleError(1, "Failed marshal to JSON: "+err.Error()) + "<br />\n"
	}
	if _, err = boardListFile.Write(boardJSON); err != nil {
		return handleError(1, "Failed writing boards.json file: "+err.Error()) + "<br />\n"
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
	start_time := benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), time.Now(), true)
	var current_page_file *os.File
	var threads []interface{}
	var thread_pages [][]interface{}
	var stickied_threads []interface{}
	var nonstickied_threads []interface{}

	// Get all top level posts for the board.
	op_posts, err := getPostArr(map[string]interface{}{
		"boardid":           board.ID,
		"parentid":          0,
		"deleted_timestamp": nilTimestamp,
	}, " ORDER BY bumped DESC")
	if err != nil {
		html += handleError(1, "Error getting OP posts for /%s/: %s", board.Dir, err.Error()) + "<br />\n"
		op_posts = nil
		return
	}

	// For each top level post, start building a Thread struct
	for _, op := range op_posts {
		var thread Thread
		var postsInThread []Post

		// Get the number of replies to this thread.
		queryStr := "SELECT COUNT(*) FROM " + config.DBprefix + "posts WHERE boardid = ? AND parentid = ? AND deleted_timestamp = ?"

		if err = queryRowSQL(queryStr,
			[]interface{}{board.ID, op.ID, nilTimestamp},
			[]interface{}{&thread.NumReplies},
		); err != nil {
			html += handleError(1,
				"Error getting replies to /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />\n"
		}

		// Get the number of image replies in this thread
		queryStr += " AND filesize <> 0"
		if err = queryRowSQL(queryStr,
			[]interface{}{board.ID, op.ID, op.DeletedTimestamp},
			[]interface{}{&thread.NumImages},
		); err != nil {
			html += handleError(1,
				"Error getting number of image replies to /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />\n"
		}

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

		postsInThread, err = getPostArr(map[string]interface{}{
			"boardid":           board.ID,
			"parentid":          op.ID,
			"deleted_timestamp": nilTimestamp,
		}, fmt.Sprintf(" ORDER BY id DESC LIMIT %d", numRepliesOnBoardPage))
		if err != nil {
			html += handleError(1,
				"Error getting posts in /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />\n"
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
			image_count := 0
			for _, reply := range postsInThread {
				if reply.Filesize != 0 {
					image_count++
				}
			}
			// Then calculate number of omitted images.
			thread.OmittedImages = thread.NumImages - image_count
		}

		// Add thread struct to appropriate list
		if op.Stickied {
			stickied_threads = append(stickied_threads, thread)
		} else {
			nonstickied_threads = append(nonstickied_threads, thread)
		}
	}

	num, _ := deleteMatchingFiles(path.Join(config.DocumentRoot, board.Dir), "\\d.html$")
	printf(2, "Number of files deleted: %d\n", num)
	// Order the threads, stickied threads first, then nonstickied threads.
	threads = append(stickied_threads, nonstickied_threads...)

	// If there are no posts on the board
	if len(threads) == 0 {
		board.CurrentPage = 1
		// Open board.html for writing to the first page.
		board_page_file, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "board.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			html += handleError(1,
				"Failed opening /%s/board.html: %s", board.Dir, err.Error()) + "<br />"
			return
		}

		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		if err = img_boardpage_tmpl.Execute(board_page_file, map[string]interface{}{
			"config":   config,
			"boards":   allBoards,
			"sections": allSections,
			"threads":  threads,
			"board":    board,
		}); err != nil {
			html += handleError(1, "Failed building /"+board.Dir+"/: "+err.Error()) + "<br />"
			return
		}

		html += "/" + board.Dir + "/ built successfully, no threads to build.\n"
		benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), start_time, false)
		return
	} else {
		// Create the archive pages.
		thread_pages = paginate(config.ThreadsPerPage, threads)
		board.NumPages = len(thread_pages) - 1

		// Create array of page wrapper objects, and open the file.
		pagesArr := make([]map[string]interface{}, board.NumPages)

		catalog_json_file, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		defer closeHandle(catalog_json_file)
		if err != nil {
			html += handleError(1, "Failed opening /"+board.Dir+"/catalog.json: "+err.Error())
			return
		}

		currentBoardPage := board.CurrentPage
		for _, page_threads := range thread_pages {
			board.CurrentPage++
			var current_page_filepath string
			pageFilename := strconv.Itoa(board.CurrentPage) + ".html"
			current_page_filepath = path.Join(config.DocumentRoot, board.Dir, pageFilename)
			current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
			defer closeHandle(current_page_file)
			if err != nil {
				html += handleError(1, "Failed opening board page: "+err.Error()) + "<br />"
				continue
			}

			// Render the boardpage template, don't forget config
			if err = img_boardpage_tmpl.Execute(current_page_file, map[string]interface{}{
				"config":   config,
				"boards":   allBoards,
				"sections": allSections,
				"threads":  page_threads,
				"board":    board,
				"posts": []interface{}{
					Post{BoardID: board.ID},
				},
			}); err != nil {
				html += handleError(1, "Failed building /"+board.Dir+"/ boardpage: "+err.Error()) + "<br />"
				return
			}

			if board.CurrentPage == 1 {
				boardPage := path.Join(config.DocumentRoot, board.Dir, "board.html")
				os.Remove(boardPage)
				if err = syscall.Symlink(current_page_filepath, boardPage); !os.IsExist(err) && err != nil {
					html += handleError(1, "Failed building /"+board.Dir+"/: "+err.Error()) + "<br />"
				}
			}

			// Collect up threads for this page.
			pageMap := make(map[string]interface{})
			pageMap["page"] = board.CurrentPage
			pageMap["threads"] = page_threads
			pagesArr = append(pagesArr, pageMap)
		}
		board.CurrentPage = currentBoardPage

		catalog_json, err := json.Marshal(pagesArr)
		if err != nil {
			html += handleError(1, "Failed to marshal to JSON: "+err.Error()) + "<br />"
			return
		}
		if _, err = catalog_json_file.Write(catalog_json); err != nil {
			html += handleError(1, "Failed writing /"+board.Dir+"/catalog.json: "+err.Error()) + "<br />"
			return
		}
		html += "/" + board.Dir + "/ built successfully.\n"
	}
	benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), start_time, false)
	return
}

// buildBoards builds the specified board IDs, or all boards if no arguments are passed
// The return value is a string of HTML with debug information produced by the build process.
func buildBoards(which ...int) (html string) {
	var boards []Board

	if which == nil {
		boards = allBoards
	} else {
		for _, b := range which {
			board, err := getBoardFromID(b)
			if err != nil {
				html += handleError(0, err.Error()) + "<br />\n"
				continue
			}
			boards = append(boards, *board)
		}
	}

	if len(boards) == 0 {
		return html + "No boards to build.<br />\n"
	}
	for _, board := range boards {
		boardPath := path.Join(config.DocumentRoot, board.Dir)
		if err := os.Mkdir(boardPath, 0666); err != nil && !os.IsExist(err) {
			html += handleError(0, "Error creating board directories: %s\n", err.Error()) + "<br />\n"
		}
		if err := os.Mkdir(path.Join(boardPath, "res"), 0666); err != nil && !os.IsExist(err) {
			html += handleError(0, "Error creating board directories: %s\n", err.Error()) + "<br />\n"
		}
		if err := os.Mkdir(path.Join(boardPath, "src"), 0666); err != nil && !os.IsExist(err) {
			html += handleError(0, "Error creating board directories: %s\n", err.Error()) + "<br />\n"
		}
		if err := os.Mkdir(path.Join(boardPath, "thumb"), 0666); err != nil && !os.IsExist(err) {
			html += handleError(0, "Error creating board directories: %s\n", err.Error()) + "<br />\n"
		}

		if board.EnableCatalog {
			html += buildCatalog(board.ID) + "<br />\n"
		}
		html += buildBoardPages(&board) + "<br />\n" +
			buildThreads(true, board.ID, 0) + "<br />\n"
	}
	return
}

func buildCatalog(which int) string {
	err := initTemplates("catalog")
	if err != nil {
		return err.Error()
	}
	board, err := getBoardFromID(which)
	if err != nil {
		return handleError(1, err.Error())
	}
	catalogPath := path.Join(config.DocumentRoot, board.Dir, "catalog.html")
	catalogFile, err := os.OpenFile(catalogPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return handleError(1, "Failed opening /%s/catalog.html: %s", board.Dir, err.Error())
	}
	threadOPs, err := getPostArr(map[string]interface{}{
		"boardid":           which,
		"parentid":          0,
		"deleted_timestamp": nilTimestamp,
	}, "ORDER BY bumped ASC")
	if err != nil {
		return handleError(1, "Error building catalog for /%s/: %s", board.Dir, err.Error())
	}
	var threadInterfaces []interface{}
	for _, thread := range threadOPs {
		threadInterfaces = append(threadInterfaces, thread)
	}
	threadPages := paginate(config.PostsPerThreadPage, threadInterfaces)

	if err = catalog_tmpl.Execute(catalogFile, map[string]interface{}{
		"boards":      allBoards,
		"config":      config,
		"board":       board,
		"sections":    allSections,
		"threadPages": threadPages,
	}); err != nil {
		return handleError(1, "Error building catalog for /%s/: %s", board.Dir, err.Error())
	}
	return fmt.Sprintf("Built catalog for /%s/ successfully", board.Dir)
}

// buildThreadPages builds the pages for a thread given by a Post object.
func buildThreadPages(op *Post) (html string) {
	err := initTemplates("threadpage")
	if err != nil {
		return err.Error()
	}

	var replies []Post
	var current_page_file *os.File
	var board *Board
	if board, err = getBoardFromID(op.BoardID); err != nil {
		html += handleError(1, err.Error())
	}

	replies, err = getPostArr(map[string]interface{}{
		"boardid":           op.BoardID,
		"parentid":          op.ID,
		"deleted_timestamp": nilTimestamp,
	}, "ORDER BY id ASC")
	if err != nil {
		html += handleError(1, "Error building thread "+strconv.Itoa(op.ID)+":"+err.Error())
		return
	}
	os.Remove(path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html"))

	var repliesInterface []interface{}
	for _, reply := range replies {
		repliesInterface = append(repliesInterface, reply)
	}

	thread_pages := paginate(config.PostsPerThreadPage, repliesInterface)
	deleteMatchingFiles(path.Join(config.DocumentRoot, board.Dir, "res"), "^"+strconv.Itoa(op.ID)+"p")

	op.NumPages = len(thread_pages)

	current_page_filepath := path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".html")
	current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		html += handleError(1, "Failed opening "+current_page_filepath+": "+err.Error())
		return
	}

	// render main page
	if err = img_threadpage_tmpl.Execute(current_page_file, map[string]interface{}{
		"config":   config,
		"boards":   allBoards,
		"board":    board,
		"sections": allSections,
		"posts":    replies,
		"op":       op,
	}); err != nil {
		html += handleError(1, "Failed building /%s/res/%d threadpage: %s", board.Dir, op.ID, err.Error()) + "<br />\n"
		return
	}

	// Put together the thread JSON
	threadJSONFile, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+".json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeHandle(threadJSONFile)
	if err != nil {
		html += handleError(1, "Failed opening /%s/res/%d.json: %s", board.Dir, op.ID, err.Error())
		return
	}

	threadMap := make(map[string][]Post)

	// Handle the OP, of type *Post
	threadMap["posts"] = []Post{*op}

	// Iterate through each reply, which are of type Post
	for _, reply := range replies {
		threadMap["posts"] = append(threadMap["posts"], reply)
	}
	threadJSON, err := json.Marshal(threadMap)
	if err != nil {
		html += handleError(1, "Failed to marshal to JSON: %s", err.Error()) + "<br />"
		return
	}

	if _, err = threadJSONFile.Write(threadJSON); err != nil {
		html += handleError(1, "Failed writing /%s/res/%d.json: %s", board.Dir, op.ID, err.Error()) + "<br />"
		return
	}

	html += fmt.Sprintf("Built /%s/%d successfully", board.Dir, op.ID)

	for page_num, page_posts := range thread_pages {
		op.CurrentPage = page_num + 1
		current_page_filepath := path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+"p"+strconv.Itoa(op.CurrentPage)+".html")
		current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			html += handleError(1, "<br />Failed opening "+current_page_filepath+": "+err.Error()) + "<br />\n"
			return
		}

		if err = img_threadpage_tmpl.Execute(current_page_file, map[string]interface{}{
			"config":   config,
			"boards":   allBoards,
			"board":    board,
			"sections": allSections,
			"posts":    page_posts,
			"op":       op,
		}); err != nil {
			html += handleError(1, "<br />Failed building /%s/%d: %s", board.Dir, op.ID, err.Error())
			return
		}

		html += fmt.Sprintf("<br />Built /%s/%dp%d successfully", board.Dir, op.ID, op.CurrentPage)
	}
	return
}

// buildThreads builds thread(s) given a boardid, or if all = false, also given a threadid.
// if all is set to true, ignore which, otherwise, which = build only specified boardid
// TODO: make it variadic
func buildThreads(all bool, boardid, threadid int) (html string) {
	var threads []Post
	var err error

	queryMap := map[string]interface{}{
		"boardid":           boardid,
		"parentid":          0,
		"deleted_timestamp": nilTimestamp,
	}
	if !all {
		queryMap["id"] = threadid
	}
	if threads, err = getPostArr(queryMap, ""); err != nil {
		return handleError(0, err.Error()) + "<br />\n"
	}
	if len(threads) == 0 {
		return "No threads to build<br />\n"
	}

	for _, op := range threads {
		html += buildThreadPages(&op) + "<br />\n"
	}
	return
}
