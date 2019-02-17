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
func buildFrontPage() (html string) {
	initTemplates()
	var recentPostsArr []interface{}

	os.Remove(path.Join(config.DocumentRoot, "index.html"))
	front_file, err := os.OpenFile(path.Join(config.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeFile(front_file)
	if err != nil {
		return handleError(1, "Failed opening front page for writing: "+err.Error()) + "<br />\n"
	}

	// get recent posts
	recentQueryStr := "SELECT `" + config.DBprefix + "posts`.`id`, " +
		"`" + config.DBprefix + "posts`.`parentid`, " +
		"`" + config.DBprefix + "boards`.`dir` AS boardname, " +
		"`" + config.DBprefix + "posts`.`boardid` AS boardid, " +
		"`name`, `tripcode`, `message`, `filename`, `thumb_w`, `thumb_h` " +
		"FROM `" + config.DBprefix + "posts`, `" + config.DBprefix + "boards` " +
		"WHERE `" + config.DBprefix + "posts`.`deleted_timestamp` = ? "
	if !config.RecentPostsWithNoFile {
		recentQueryStr += "AND `" + config.DBprefix + "posts`.`filename` != '' AND `" + config.DBprefix + "posts`.filename != 'deleted' "
	}
	recentQueryStr += "AND `boardid` = `" + config.DBprefix + "boards`.`id` " +
		"ORDER BY `timestamp` DESC LIMIT ?"

	rows, err := querySQL(recentQueryStr, nilTimestamp, config.MaxRecentPosts)
	defer closeRows(rows)
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

	for i := range allBoards {
		board := allBoards[i].(BoardsTable)
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
	board_list_file, err := os.OpenFile(path.Join(config.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeFile(board_list_file)
	if err != nil {
		return handleError(1, "Failed opening board.json for writing: "+err.Error()) + "<br />\n"
	}

	board_list_wrapper := new(BoardJSONWrapper)

	// Our cooldowns are site-wide currently.
	cooldowns_obj := BoardCooldowns{NewThread: config.NewThreadDelay, Reply: config.ReplyDelay, ImageReply: config.ReplyDelay}

	for _, board_int := range allBoards {
		board := board_int.(BoardsTable)
		board_obj := BoardJSON{BoardName: board.Dir, Title: board.Title, WorkSafeBoard: 1,
			ThreadsPerPage: config.ThreadsPerPage, Pages: board.MaxPages, MaxFilesize: board.MaxImageSize,
			MaxMessageLength: board.MaxMessageLength, BumpLimit: 200, ImageLimit: board.NoImagesAfter,
			Cooldowns: cooldowns_obj, Description: board.Description, IsArchived: 0}
		if board.EnableNSFW {
			board_obj.WorkSafeBoard = 0
		}
		board_list_wrapper.Boards = append(board_list_wrapper.Boards, board_obj)
	}

	boardJSON, err := json.Marshal(board_list_wrapper)
	if err != nil {
		return handleError(1, "Failed marshal to JSON: "+err.Error()) + "<br />\n"
	}
	if _, err = board_list_file.Write(boardJSON); err != nil {
		return handleError(1, "Failed writing boards.json file: "+err.Error()) + "<br />\n"
	}
	return "Board list JSON rebuilt successfully.<br />"
}

// buildBoardPages builds the pages for the board archive.
// `board` is a BoardsTable object representing the board to build archive pages for.
// The return value is a string of HTML with debug information from the build process.
func buildBoardPages(board *BoardsTable) (html string) {
	start_time := benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), time.Now(), true)
	var current_page_file *os.File
	var threads []interface{}
	var thread_pages [][]interface{}
	var stickied_threads []interface{}
	var nonstickied_threads []interface{}

	// Check that the board's configured directory is indeed a directory
	results, err := os.Stat(path.Join(config.DocumentRoot, board.Dir))
	if err != nil {
		// Try creating the board's configured directory if it doesn't exist
		err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir), 0777)
		if err != nil {
			html += handleError(1, "Failed creating /"+board.Dir+"/: "+err.Error())
			return
		}
	} else if !results.IsDir() {
		// If the file exists, but is not a folder, notify the user
		html += handleError(1, "Error: /"+board.Dir+"/ exists, but is not a folder.")
		return
	}

	// Get all top level posts for the board.
	op_posts, err := getPostArr(map[string]interface{}{
		"boardid":           board.ID,
		"parentid":          0,
		"deleted_timestamp": nilTimestamp,
	}, " ORDER BY `bumped` DESC")
	if err != nil {
		html += handleError(1, err.Error()) + "<br />"
		op_posts = nil
		return
	}

	// For each top level post, start building a Thread struct
	for _, op := range op_posts {
		var thread Thread
		var posts_in_thread []PostTable

		// Get the number of replies to this thread.
		if err = queryRowSQL("SELECT COUNT(*) FROM `"+config.DBprefix+"posts` WHERE `boardid` = ? AND `parentid` = ? AND `deleted_timestamp` = ?",
			[]interface{}{board.ID, op.ID, nilTimestamp},
			[]interface{}{&thread.NumReplies},
		); err != nil {
			html += err.Error() + "<br />\n"
		}

		// Get the number of image replies in this thread
		if err = queryRowSQL("SELECT COUNT(*) FROM `"+config.DBprefix+"posts` WHERE `boardid` = ? AND `parentid` = ? AND `deleted_timestamp` = ? AND `filesize` <> 0",
			[]interface{}{board.ID, op.ID, nilTimestamp},
			[]interface{}{&thread.NumImages},
		); err != nil {
			html += err.Error() + "<br />\n"
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

		posts_in_thread, err = getPostArr(map[string]interface{}{
			"boardid":           board.ID,
			"parentid":          op.ID,
			"deleted_timestamp": nilTimestamp,
		}, fmt.Sprintf(" ORDER BY `id` DESC LIMIT %d", numRepliesOnBoardPage))
		if err != nil {
			html += err.Error() + "<br />"
		}

		var reversedPosts []PostTable
		for i := len(posts_in_thread); i > 0; i-- {
			reversedPosts = append(reversedPosts, posts_in_thread[i-1])
		}

		if len(posts_in_thread) > 0 {
			// Store the posts to show on board page
			//thread.BoardReplies = posts_in_thread
			thread.BoardReplies = reversedPosts

			// Count number of images on board page
			image_count := 0
			for _, reply := range posts_in_thread {
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
			html += handleError(1, "Failed opening /"+board.Dir+"/board.html: "+err.Error()) + "<br />"
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
		var pages_obj []BoardPageJSON

		catalog_json_file, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		defer closeFile(catalog_json_file)
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
			defer closeFile(current_page_file)
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
					PostTable{BoardID: board.ID},
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
			var page_obj BoardPageJSON
			page_obj.Page = board.CurrentPage

			for _, thread_int := range page_threads {
				thread := thread_int.(Thread)
				post_json := makePostJSON(thread.OP, board.Anonymous)
				var thread_json ThreadJSON
				thread_json.PostJSON = &post_json
				thread_json.Replies = thread.NumReplies
				thread_json.ImagesOnArchive = thread.NumImages
				thread_json.OmittedImages = thread.OmittedImages
				if thread.Stickied {
					if thread.NumReplies > config.StickyRepliesOnBoardPage {
						thread_json.OmittedPosts = thread.NumReplies - config.StickyRepliesOnBoardPage
					}
					thread_json.Sticky = 1
				} else {
					if thread.NumReplies > config.RepliesOnBoardPage {
						thread_json.OmittedPosts = thread.NumReplies - config.RepliesOnBoardPage
					}
				}
				if thread.OP.Locked {
					thread_json.Locked = 1
				}
				page_obj.Threads = append(page_obj.Threads, thread_json)
			}

			pages_obj = append(pages_obj, page_obj)
		}
		board.CurrentPage = currentBoardPage

		catalog_json, err := json.Marshal(pages_obj)
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

// buildBoards builds one or all boards.
// If `all` == true, all boards will have their pages built and `which` is ignored
// Otherwise, the board with the id equal to the value specified as which.
// The return value is a string of HTML with debug information produced by the build process.
// TODO: make this a variadic function (which ...int)
func buildBoards(all bool, which int) (html string) {
	if all {
		boards, _ := getBoardArr(nil, "")
		if len(boards) == 0 {
			return html + "No boards to build.<br />\n"
		}
		for _, board := range boards {
			html += buildBoardPages(&board) + "<br />\n"
			if board.EnableCatalog {
				html += buildCatalog(board.ID) + "<br />\n"
			}

			html += buildThreads(true, board.ID, 0)
		}
	} else {
		boardArr, _ := getBoardArr(map[string]interface{}{"id": which}, "")
		board := boardArr[0]
		html += buildBoardPages(&board) + "<br />\n"
		if board.EnableCatalog {
			html += buildCatalog(board.ID) + "<br />\n"
		}
		html += buildThreads(true, board.ID, 0)
	}

	return
}

func buildCatalog(which int) (html string) {
	board, err := getBoardFromID(which)
	if err != nil {
		html += handleError(1, err.Error())
	}
	catalogPath := path.Join(config.DocumentRoot, board.Dir, "catalog.html")
	catalogFile, err := os.OpenFile(catalogPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		html += handleError(1, "Failed opening /"+board.Dir+"/catalog.html: "+err.Error())
		return
	}
	threadOPs, err := getPostArr(map[string]interface{}{
		"boardid":           which,
		"parentid":          0,
		"deleted_timestamp": nilTimestamp,
	}, "ORDER BY `bumped` ASC")
	if err != nil {
		html += handleError(1, "Error building catalog for /%s/: %s", board.Dir, err.Error())
		return
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
		html += handleError(1, "Error building catalog for /%s/: %s", board.Dir, err.Error())
		return
	}
	html += fmt.Sprintf("Built catalog for /%s/ successfully", board.Dir)
	return
}

// buildThreadPages builds the pages for a thread given by a PostTable object.
func buildThreadPages(op *PostTable) (html string) {
	var replies []PostTable
	var current_page_file *os.File
	board, err := getBoardFromID(op.BoardID)
	if err != nil {
		html += handleError(1, err.Error())
	}

	replies, err = getPostArr(map[string]interface{}{
		"boardid":           op.BoardID,
		"parentid":          op.ID,
		"deleted_timestamp": nilTimestamp,
	}, "ORDER BY `id` ASC")
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
	defer closeFile(threadJSONFile)
	if err != nil {
		html += handleError(1, "Failed opening /%s/res/%d.json: %s", board.Dir, op.ID, err.Error())
		return
	}

	// Create the wrapper object
	thread_json_wrapper := new(ThreadJSONWrapper)

	// Handle the OP, of type *PostTable
	op_post_obj := makePostJSON(*op, board.Anonymous)
	thread_json_wrapper.Posts = append(thread_json_wrapper.Posts, op_post_obj)

	// Iterate through each reply, which are of type PostTable
	for _, reply := range replies {
		postJSON := makePostJSON(reply, board.Anonymous)
		thread_json_wrapper.Posts = append(thread_json_wrapper.Posts, postJSON)
	}
	threadJSON, err := json.Marshal(thread_json_wrapper)
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
// TODO: detect which page will be built and only build that one and the board page
func buildThreads(all bool, boardid, threadid int) (html string) {
	if !all {
		threads, _ := getPostArr(map[string]interface{}{
			"boardid":           boardid,
			"id":                threadid,
			"parentid":          0,
			"deleted_timestamp": nilTimestamp,
		}, "")
		thread := threads[0]
		html += buildThreadPages(&thread) + "<br />\n"
		return
	}

	threads, _ := getPostArr(map[string]interface{}{
		"boardid":           boardid,
		"parentid":          0,
		"deleted_timestamp": nilTimestamp,
	}, "")
	if len(threads) == 0 {
		return
	}

	for _, op := range threads {
		html += buildThreadPages(&op) + "<br />\n"
	}
	return
}
