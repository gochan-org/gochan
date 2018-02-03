// functions for handling posting, uploading, and post/thread/board page building

package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aquilax/tripcode"
	"github.com/disintegration/imaging"
)

const (
	whitespaceMatch = "[\000-\040]"
	gt              = "&gt;"
	yearInSeconds   = 31536000
)

var (
	last_post    PostTable
	all_sections []interface{}
	all_boards   []interface{}
)

// buildBoards builds one or all boards. If all == true, all boards will have their pages built.
// If all == false, the board with the id equal to the value specified as which.
// The return value is a string of HTML with debug information produced by the build process.
func buildBoards(all bool, which int) (html string) {
	// if all is set to true, ignore which, otherwise, which = build only specified boardid
	if !all {
		boardArr, _ := getBoardArr(map[string]interface{}{"id": which}, "")
		board := boardArr[0]
		html += buildBoardPages(&board) + "<br />\n"
		html += buildThreads(true, board.ID, 0)
		return
	}
	boards, _ := getBoardArr(nil, "")
	if len(boards) == 0 {
		return html + "No boards to build.<br />\n"
	}

	for _, board := range boards {
		html += buildBoardPages(&board) + "<br />\n"
		html += buildThreads(true, board.ID, 0)
	}
	return
}

// buildBoardPages builds the pages for the board archive. board is a BoardsTable object representing the board to
// 	build archive pages for. The return value is a string of HTML with debug information from the build process.
func buildBoardPages(board *BoardsTable) (html string) {
	//	start_time := benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), time.Now(), true)
	var boardinfo_i []interface{}
	var current_page_file *os.File
	var threads []interface{}
	var thread_pages [][]interface{}
	var stickied_threads []interface{}
	var nonstickied_threads []interface{}
	var errortext string

	defer func() {
		// This function cleans up after we return. If there was an error, it prints on the log and the console.
		if uhoh, ok := recover().(error); ok {
			errorLog.Print("buildBoardPages failed: " + uhoh.Error())
			println(0, "buildBoardPages failed: "+uhoh.Error())
		}
		if current_page_file != nil {
			current_page_file.Close()
		}
	}()

	// Check that the board's configured directory is indeed a directory
	results, err := os.Stat(path.Join(config.DocumentRoot, board.Dir))
	if err != nil {
		// Try creating the board's configured directory if it doesn't exist
		err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir), 0777)
		if err != nil {
			errortext = "Failed creating /" + board.Dir + "/: " + err.Error()
			html += errortext + "<br />\n"
			errorLog.Println(errortext)
			println(1, errortext)
		}
	} else if !results.IsDir() {
		// If the file exists, but is not a folder, notify the user
		errortext = "Error: /" + board.Dir + "/ exists, but is not a folder."
		html += errortext + "<br />\n"
		errorLog.Println(errortext)
		println(1, errortext)
	}

	// Get all top level posts for the board.
	op_posts, err := getPostArr(map[string]interface{}{
		"boardid":           board.ID,
		"parentid":          0,
		"deleted_timestamp": nil_timestamp,
	}, " ORDER BY `bumped` DESC")
	if err != nil {
		html += err.Error() + "<br />"
		op_posts = nil //make([]interface{}, 0)
		return
	}

	// For each top level post, start building a Thread struct
	for _, op_post_i := range op_posts {
		var thread Thread
		var posts_in_thread []interface{}
		op_post := op_post_i.(PostTable)
		thread.IName = "thread"

		// Store the OP post for this thread
		//op_post := op_post_i.(PostTable)

		// Get the number of replies to this thread.
		stmt, err := db.Prepare("SELECT COUNT(*) FROM `" + config.DBprefix + "posts` WHERE `boardid` = ? AND `parentid` = ? AND `deleted_timestamp` = ?")
		if err != nil {
			html += err.Error() + "<br />\n"
		}
		defer func() {
			if stmt != nil {
				stmt.Close()
			}
		}()
		err = stmt.QueryRow(board.ID, op_post.ID, nil_timestamp).Scan(&thread.NumReplies)
		if err != nil {
			html += err.Error() + "<br />\n"
		}

		// Get the number of image replies in this thread
		stmt, err = db.Prepare("SELECT COUNT(*) FROM `" + config.DBprefix + "posts` WHERE `boardid` = ? AND `parentid` = ? AND `deleted_timestamp` = ? AND `filesize` <> 0")
		if err != nil {
			html += err.Error() + "<br />\n"
		}
		err = stmt.QueryRow(board.ID, op_post.ID, nil_timestamp).Scan(&thread.NumImages)
		if err != nil {
			html += err.Error() + "<br />\n"
		}

		thread.OP = op_post

		var numRepliesOnBoardPage int

		if op_post.Stickied {
			// If the thread is stickied, limit replies on the archive page to the
			// configured value for stickied threads.
			numRepliesOnBoardPage = config.StickyRepliesOnBoardPage
		} else {
			// Otherwise, limit the replies to the configured value for normal threads.
			numRepliesOnBoardPage = config.RepliesOnBoardPage
		}

		posts_in_thread, err = getPostArr(map[string]interface{}{
			"boardid":           board.ID,
			"parentid":          op_post.ID,
			"deleted_timestamp": nil_timestamp,
		}, fmt.Sprintf(" ORDER BY `id` DESC LIMIT %d", numRepliesOnBoardPage))
		if err != nil {
			html += err.Error() + "<br />"
		}

		posts_in_thread = reverse(posts_in_thread)

		if len(posts_in_thread) > 0 {
			// Store the posts to show on board page
			thread.BoardReplies = posts_in_thread

			// Count number of images on board page
			image_count := 0
			for _, reply := range posts_in_thread {
				if reply.(PostTable).Filesize != 0 {
					image_count++
				}
			}
			// Then calculate number of omitted images.
			thread.OmittedImages = thread.NumImages - image_count
		}

		// Add thread struct to appropriate list
		if op_post.Stickied {
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
		board.CurrentPage = 0
		boardinfo_i = nil
		boardinfo_i = append(boardinfo_i, board)

		// Open board.html for writing to the first page.
		printf(1, "Current page: %s/%d\n", board.Dir, board.CurrentPage)
		board_page_file, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "board.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			errortext = "Failed opening /" + board.Dir + "/board.html: " + err.Error()
			html += errortext + "<br />\n"
			errorLog.Println(errortext)
			println(1, errortext)
			return
		}

		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		err = renderTemplate(img_boardpage_tmpl, "boardpage", board_page_file,
			&Wrapper{IName: "boards", Data: all_boards},
			&Wrapper{IName: "sections", Data: all_sections},
			&Wrapper{IName: "threads", Data: threads},
			&Wrapper{IName: "boardinfo", Data: boardinfo_i},
		)
		if err != nil {
			errortext = "Failed building /" + board.Dir + "/: " + err.Error()
			html += errortext + "<br />\n"
			errorLog.Print(errortext)
			println(1, errortext)
			return
		}
		html += "/" + board.Dir + "/ built successfully, no threads to build.\n"
		//benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), start_time, false)
		return
	} else {
		// Create the archive pages.
		thread_pages = paginate(config.ThreadsPerPage_img, threads)

		board.NumPages = len(thread_pages) - 1

		// Create array of page wrapper objects, and open the file.
		var pages_obj []BoardPageJSON

		catalog_json_file, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		defer func() {
			if catalog_json_file != nil {
				catalog_json_file.Close()
			}
		}()

		if err != nil {
			errortext = "Failed opening /" + board.Dir + "/catalog.json: " + err.Error()
			html += errortext + "<br />\n"
			println(1, errortext)
			errorLog.Print(errortext)
			return
		}

		for page_num, page_threads := range thread_pages {
			// Package up board info for the template to use.
			board.CurrentPage = page_num
			boardinfo_i = nil
			boardinfo_i = append(boardinfo_i, board)

			// Write to board.html for the first page.
			var current_page_filepath string
			if board.CurrentPage == 0 {
				current_page_filepath = path.Join(config.DocumentRoot, board.Dir, "board.html")
			} else {
				current_page_filepath = path.Join(config.DocumentRoot, board.Dir, strconv.Itoa(page_num)+".html")
			}

			current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
			if err != nil {
				errortext = "Failed opening board page: " + err.Error()
				html += errortext + "<br />\n"
				errorLog.Println(errortext)
				println(1, errortext)
				continue
			}
			// Render the boardpage template, given boards, sections, threads, and board info
			err = renderTemplate(img_boardpage_tmpl, "boardpage", current_page_file,
				&Wrapper{IName: "boards", Data: all_boards},
				&Wrapper{IName: "sections", Data: all_sections},
				&Wrapper{IName: "threads", Data: page_threads},
				&Wrapper{IName: "boardinfo", Data: boardinfo_i},
			)
			if err != nil {
				errortext = "Failed building /" + board.Dir + "/: " + err.Error()
				html += errortext + "<br />\n"
				errorLog.Print(errortext)
				println(1, errortext)
				return
			}

			// Clean up page's file
			current_page_file.Close()

			// Collect up threads for this page.
			var page_obj BoardPageJSON
			page_obj.Page = page_num

			for _, thread_int := range page_threads {
				thread := thread_int.(Thread)
				post_json := makePostJSON(thread.OP.(PostTable), board.Anonymous)
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
				if thread.OP.(PostTable).Locked {
					thread_json.Locked = 1
				}
				page_obj.Threads = append(page_obj.Threads, thread_json)
			}

			pages_obj = append(pages_obj, page_obj)
		}

		catalog_json, err := json.Marshal(pages_obj)

		if err != nil {
			errortext = "Failed to marshal to JSON: " + err.Error()
			errorLog.Println(errortext)
			println(1, errortext)
			html += errortext + "<br />\n"
			return
		}

		_, err = catalog_json_file.Write(catalog_json)

		if err != nil {
			errortext = "Failed writing /" + board.Dir + "/catalog.json: " + err.Error()
			errorLog.Println(errortext)
			println(1, errortext)
			html += errortext + "<br />\n"
			return
		}

		html += "/" + board.Dir + "/ built successfully.\n"
	}

	//benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), start_time, false)
	return
}

// buildThreads builds thread(s) given a boardid, or if all = false, also given a threadid.
func buildThreads(all bool, boardid, threadid int) (html string) {
	// TODO: detect which page will be built and only build that one and the board page
	// if all is set to true, ignore which, otherwise, which = build only specified boardid
	if !all {
		_thread, _ := getPostArr(map[string]interface{}{
			"boardid":           boardid,
			"id":                threadid,
			"parentid":          0,
			"deleted_timestamp": nil_timestamp,
		}, "")
		//thread := _thread[0].(PostTable)
		thread := _thread[0]
		thread_struct := thread.(PostTable)
		html += buildThreadPages(&thread_struct) + "<br />\n"
		return
	}

	threads, _ := getPostArr(map[string]interface{}{
		"boardid":           boardid,
		"parentid":          0,
		"deleted_timestamp": nil_timestamp,
	}, "")
	if len(threads) == 0 {
		return
	}
	for _, op_i := range threads {
		op := op_i.(PostTable)
		html += buildThreadPages(&op) + "<br />\n"
	}
	return
}

// buildThreadPages builds the pages for a thread given by a PostTable object.
func buildThreadPages(op *PostTable) (html string) {
	var boardDir string
	var anonymous string
	var replies []interface{}
	var current_page_file *os.File
	var errortext string

	stmt, err := db.Prepare("SELECT `dir`,`anonymous` FROM `" + config.DBprefix + "boards` WHERE `id` = ?")
	if err != nil {
		// possibly a syntax error? This normally shouldn't happen so this might be removed
		errortext = err.Error()
		html += errortext + "<br />\n"
		errorLog.Println(errortext)
		println(1, errortext)
		return
	}

	err = stmt.QueryRow(op.BoardID).Scan(&boardDir, &anonymous)
	if err != nil {
		errortext = "Failed getting board directory and board's anonymous setting from post: " + err.Error()
		html += errortext + "<br />\n"
		errorLog.Println(errortext)
		println(1, errortext)
		return
	}

	replies, err = getPostArr(map[string]interface{}{
		"boardid":           op.BoardID,
		"parentid":          op.ID,
		"deleted_timestamp": nil_timestamp,
	}, "ORDER BY `ID` ASC")
	if err != nil {
		errortext = "Error building thread " + strconv.Itoa(op.ID) + ":" + err.Error()
		html += errortext
		errorLog.Println(errortext)
		println(1, errortext)
		return
	}
	os.Remove(path.Join(config.DocumentRoot, boardDir, "res", strconv.Itoa(op.ID)+".html"))

	var repliesInterface []interface{}
	for _, reply := range replies {
		repliesInterface = append(repliesInterface, reply)
	}

	//thread_pages := paginate(config.PostsPerThreadPage, replies)
	thread_pages := paginate(config.PostsPerThreadPage, repliesInterface)
	for i, _ := range thread_pages {
		thread_pages[i] = append([]interface{}{op}, thread_pages[i]...)
	}
	deleteMatchingFiles(path.Join(config.DocumentRoot, boardDir, "res"), "^"+strconv.Itoa(op.ID)+"p")

	op.NumPages = len(thread_pages)

	current_page_filepath := path.Join(config.DocumentRoot, boardDir, "res", strconv.Itoa(op.ID)+".html")
	current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		errortext = "Failed opening " + current_page_filepath + ": " + err.Error()
		html += errortext + "<br />\n"
		println(1, errortext)
		errorLog.Println(errortext)
		return
	}
	// render main page
	err = renderTemplate(img_threadpage_tmpl, "threadpage", current_page_file,
		&Wrapper{IName: "boards_", Data: all_boards},
		&Wrapper{IName: "sections_w", Data: all_sections},
		&Wrapper{IName: "posts_w", Data: append([]interface{}{op}, replies...)},
	)

	if err != nil {
		fmt.Sprintf("Failed building /%s/res/%d.html: %s", boardDir, op.ID, err.Error())
		html += errortext + "<br />\n"
		println(1, errortext)
		errorLog.Print(errortext)
		return
	}

	// Put together the thread JSON
	threadJSONFile, err := os.OpenFile(path.Join(config.DocumentRoot, boardDir, "res", strconv.Itoa(op.ID)+".json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer func() {
		if threadJSONFile != nil {
			threadJSONFile.Close()
		}
	}()
	if err != nil {
		errortext = fmt.Sprintf("Failed opening /%s/res/%d.json: %s", boardDir, op.ID, err.Error())
		html += errortext + "<br />\n"
		println(1, errortext)
		errorLog.Print(errortext)
		return
	}
	// Create the wrapper object
	thread_json_wrapper := new(ThreadJSONWrapper)

	// Handle the OP, of type *PostTable
	op_post_obj := makePostJSON(*op, anonymous)

	thread_json_wrapper.Posts = append(thread_json_wrapper.Posts, op_post_obj)

	// Iterate through each reply, which are of type PostTable
	for _, post_int := range replies {
		post := post_int.(PostTable)
		post_obj := makePostJSON(post, anonymous)
		thread_json_wrapper.Posts = append(thread_json_wrapper.Posts, post_obj)
	}
	threadJSON, err := json.Marshal(thread_json_wrapper)

	if err != nil {
		errortext = "Failed to marshal to JSON: " + err.Error()
		errorLog.Println(errortext)
		println(1, errortext)
		html += errortext + "<br />\n"
		return
	}

	_, err = threadJSONFile.Write(threadJSON)
	if err != nil {
		errortext = fmt.Sprintf("Failed writing /%s/res/%d.json: %s", boardDir, op.ID, err.Error())
		errorLog.Println(errortext)
		println(1, errortext)
		html += errortext + "<br />\n"
		return
	}

	success_text := fmt.Sprintf("Built /%s/%d successfully", boardDir, op.ID)
	html += success_text + "<br />\n"
	println(2, success_text)

	for page_num, page_posts := range thread_pages {
		op.CurrentPage = page_num

		current_page_filepath := path.Join(config.DocumentRoot, boardDir, "res", strconv.Itoa(op.ID)+"p"+strconv.Itoa(op.CurrentPage+1)+".html")
		current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			errortext = "Failed opening " + current_page_filepath + ": " + err.Error()
			html += errortext + "<br />\n"
			println(1, errortext)
			errorLog.Println(errortext)
			return
		}

		// var postsSanitized []interface{}
		// for i := 0; i < len(page_posts); i++ {
		// 	postsSanitized = append(postsSanitized, sanitizePost(page_posts[i].(PostTable)))
		// }

		err = renderTemplate(img_threadpage_tmpl, "threadpage", current_page_file,
			&Wrapper{IName: "boards_", Data: all_boards},
			&Wrapper{IName: "sections_w", Data: all_sections},
			//&Wrapper{IName: "posts_w", Data: postsSanitized},
			&Wrapper{IName: "posts_w", Data: page_posts},
		)
		if err != nil {
			errortext = fmt.Sprintf("Failed building /%s/%d: %s", boardDir, op.ID, err.Error())
			html += errortext + "<br />\n"
			println(1, errortext)
			errorLog.Print(errortext)
			return
		}
		success_text := fmt.Sprintf("Built /%s/%dp%d successfully", boardDir, op.ID, op.CurrentPage+1)
		html += success_text + "<br />\n"
		println(2, success_text)
	}
	return
}

func buildFrontPage() (html string) {
	initTemplates()
	var front_arr []interface{}
	var recent_posts_arr []interface{}

	var errortext string
	os.Remove(path.Join(config.DocumentRoot, "index.html"))
	front_file, err := os.OpenFile(path.Join(config.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer func() {
		if front_file != nil {
			front_file.Close()
		}
	}()
	if err != nil {
		errortext = "Failed opening front page for writing: " + err.Error()
		errorLog.Println(errortext)
		return errortext + "<br />\n"
	}

	// get front pages
	rows, err := db.Query("SELECT * FROM `" + config.DBprefix + "frontpage`")
	if err != nil {
		errortext = "Failed getting front page rows: " + err.Error()
		errorLog.Print(errortext)
		return errortext + "<br />"
	}
	for rows.Next() {
		frontpage := new(FrontTable)
		frontpage.IName = "front page"
		err = rows.Scan(&frontpage.ID, &frontpage.Page, &frontpage.Order, &frontpage.Subject, &frontpage.Message, &frontpage.Timestamp, &frontpage.Poster, &frontpage.Email)
		if err != nil {
			errorLog.Print(err.Error())
			println(1, err.Error())
			return err.Error()
		}
		front_arr = append(front_arr, frontpage)
	}

	// get recent posts
	stmt, err := db.Prepare(
		"SELECT `" + config.DBprefix + "posts`.`id`, " +
			"`" + config.DBprefix + "posts`.`parentid`, " +
			"`" + config.DBprefix + "boards`.`dir` AS boardname, " +
			"`" + config.DBprefix + "posts`.`boardid` AS boardid, " +
			"`name`, `tripcode`, `message`, `filename`, `thumb_w`, `thumb_h` " +
			"FROM `" + config.DBprefix + "posts`, `" + config.DBprefix + "boards` " +
			"WHERE `" + config.DBprefix + "posts`.`deleted_timestamp` = ? " +
			"AND `boardid` = `" + config.DBprefix + "boards`.`id` " +
			"ORDER BY `timestamp` DESC LIMIT ?")
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()
	if err != nil {
		errorLog.Print(err.Error())
		println(1, err.Error())
		return err.Error() + "<br />\n"
	}
	rows, err = stmt.Query(nil_timestamp, config.MaxRecentPosts)
	if err != nil {
		errortext = "Failed getting list of recent posts for front page: " + err.Error()
		errorLog.Print(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	for rows.Next() {
		recent_post := new(RecentPost)
		err = rows.Scan(&recent_post.PostID, &recent_post.ParentID, &recent_post.BoardName, &recent_post.BoardID, &recent_post.Name, &recent_post.Tripcode, &recent_post.Message, &recent_post.Filename, &recent_post.ThumbW, &recent_post.ThumbH)
		if err != nil {
			errortext = "Failed getting list of recent posts for front page: " + err.Error()
			errorLog.Print(errortext)
			println(1, errortext)
			return errortext + "<br />\n"
		}
		recent_posts_arr = append(recent_posts_arr, recent_post)
	}

	err = renderTemplate(front_page_tmpl, "frontpage", front_file,
		&Wrapper{IName: "fronts", Data: front_arr},
		&Wrapper{IName: "boards", Data: all_boards},
		&Wrapper{IName: "sections", Data: all_sections},
		&Wrapper{IName: "recent posts", Data: recent_posts_arr},
	)
	if err != nil {
		errortext = "Failed executing front page template: " + err.Error()
		errorLog.Println(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	return "Front page rebuilt successfully.<br />"
}

func buildBoardListJSON() (html string) {
	var errortext string
	board_list_file, err := os.OpenFile(path.Join(config.DocumentRoot, "boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer func() {
		if board_list_file != nil {
			board_list_file.Close()
		}
	}()
	if err != nil {
		errortext = "Failed opening board.json for writing: " + err.Error()
		errorLog.Println(errortext)
		return errortext + "<br />\n"
	}

	board_list_wrapper := new(BoardJSONWrapper)

	// Our cooldowns are site-wide currently.
	cooldowns_obj := BoardCooldowns{NewThread: config.NewThreadDelay, Reply: config.ReplyDelay, ImageReply: config.ReplyDelay}

	for _, board_int := range all_boards {
		board := board_int.(BoardsTable)
		board_obj := BoardJSON{BoardName: board.Dir, Title: board.Title, WorkSafeBoard: 1,
			ThreadsPerPage: config.ThreadsPerPage_img, Pages: board.MaxPages, MaxFilesize: board.MaxImageSize,
			MaxMessageLength: board.MaxMessageLength, BumpLimit: 200, ImageLimit: board.NoImagesAfter,
			Cooldowns: cooldowns_obj, Description: board.Description, IsArchived: 0}
		if board.EnableNSFW {
			board_obj.WorkSafeBoard = 0
		}
		board_list_wrapper.Boards = append(board_list_wrapper.Boards, board_obj)
	}

	boardJSON, err := json.Marshal(board_list_wrapper)

	if err != nil {
		errortext = "Failed marshal to JSON: " + err.Error()
		errorLog.Println(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	_, err = board_list_file.Write(boardJSON)

	if err != nil {
		errortext = "Failed writing boards.json file: " + err.Error()
		errorLog.Println(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	return "Board list JSON rebuilt successfully.<br />"
}

// bumps the given thread on the given board and returns true if it exists
// or false on error
func bumpThread(postID, boardID int) error {
	stmt, err := db.Prepare("UPDATE `" + config.DBprefix + "posts` SET `bumped` = ? WHERE `id` = ? AND `boardid` = ?")
	if err != nil {
		return err
	}
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()

	_, err = stmt.Exec(time.Now(), postID, boardID)
	if err != nil {
		return err
	}
	return nil
}

// Checks check poster's name/tripcode/file checksum (from PostTable post) for banned status
// returns true if the user is banned
func checkBannedStatus(post *PostTable, writer *http.ResponseWriter) ([]interface{}, error) {
	var is_expired bool
	var ban_entry BanlistTable
	var interfaces []interface{}
	// var count int
	// var search string
	stmt, err := db.Prepare("SELECT `ip`, `name`, `tripcode`, `message`, `boards`, `timestamp`, `expires`, `appeal_at` FROM `" + config.DBprefix + "banlist` WHERE `ip` = ?")
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()
	if err != nil {
		errortext := "Error checking banned status: " + err.Error()
		println(1, errortext)
		errorLog.Print(errortext)
		return interfaces, nil
	}
	err = stmt.QueryRow(&post.IP).Scan(&ban_entry.IP, &ban_entry.Name, &ban_entry.Tripcode, &ban_entry.Message, &ban_entry.Boards, &ban_entry.Timestamp, &ban_entry.Expires, &ban_entry.AppealAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// the user isn't banned
			// We don't need to return err because it isn't necessary
			return interfaces, nil

		} else {
			// something went wrong
			return interfaces, err
		}
	} else {
		is_expired = ban_entry.Expires.After(time.Now()) == false
		if is_expired {
			// if it is expired, send a message saying that it's expired, but still post
			println(1, "expired")
			return interfaces, nil

		}
		// the user's IP is in the banlist. Check if the ban has expired
		if getSpecificSQLDateTime(ban_entry.Expires) == "0001-01-01 00:00:00" || ban_entry.Expires.After(time.Now()) {
			// for some funky reason, Go's MySQL driver seems to not like getting a supposedly nil timestamp as an ACTUAL nil timestamp
			// so we're just going to wing it and cheat. Of course if they change that, we're kind of hosed.

			var interfaces []interface{}
			interfaces = append(interfaces, config)
			interfaces = append(interfaces, ban_entry)
			return interfaces, nil
		}
		return interfaces, nil
	}
	return interfaces, nil
}

func sinceLastPost(post *PostTable) int {
	var oldpost PostTable
	err := db.QueryRow("SELECT `timestamp` FROM `" + config.DBprefix + "posts` WHERE `ip` = '" + post.IP + "' ORDER BY `timestamp` DESC LIMIT 1").Scan(&oldpost.Timestamp)

	since := time.Since(oldpost.Timestamp)
	if err == sql.ErrNoRows {
		// no posts by that IP.
		return -1
	} else {
		return int(since.Seconds())
	}
	return -1
}

func createImageThumbnail(image_obj image.Image, size string) image.Image {
	var thumb_width int
	var thumb_height int

	switch size {
	case "op":
		thumb_width = config.ThumbWidth
		thumb_height = config.ThumbHeight
	case "reply":
		thumb_width = config.ThumbWidth_reply
		thumb_height = config.ThumbHeight_reply
	case "catalog":
		thumb_width = config.ThumbWidth_catalog
		thumb_height = config.ThumbHeight_catalog
	}
	old_rect := image_obj.Bounds()
	if thumb_width >= old_rect.Max.X && thumb_height >= old_rect.Max.Y {
		return image_obj
	}

	thumb_w, thumb_h := getThumbnailSize(old_rect.Max.X, old_rect.Max.Y, size)
	image_obj = imaging.Resize(image_obj, thumb_w, thumb_h, imaging.CatmullRom) // resize to 600x400 px using CatmullRom cubic filter
	return image_obj
}

func createVideoThumbnail(video, thumb string, size int) error {
	sizeStr := strconv.Itoa(size)
	outputBytes, err := exec.Command("ffmpeg", "-y", "-itsoffset", "-1", "-i", video, "-vframes", "1", "-filter:v", "scale='min("+sizeStr+"\\, "+sizeStr+"):-1'", thumb).CombinedOutput()
	println(2, "ffmpeg output: \n"+string(outputBytes))
	if err != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		if len(outputStringArr) > 1 {
			outputString := outputStringArr[len(outputStringArr)-2]
			err = errors.New(outputString)
		}
	}
	return err
}

func getVideoInfo(path string) (map[string]int, error) {
	var vidInfo map[string]int
	outputBytes, err := exec.Command("ffprobe", "-v quiet", "-show_format", "-show_streams", path).CombinedOutput()
	if err == nil && outputBytes != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		for _, line := range outputStringArr {
			lineArr := strings.Split(line, "=")
			if len(lineArr) < 2 {
				continue
			}

			if lineArr[0] == "width" || lineArr[0] == "height" || lineArr[0] == "size" {
				value, _ := strconv.Atoi(lineArr[1])
				vidInfo[lineArr[0]] = value
			}
		}
	}

	return vidInfo, err
}

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now)) + strconv.Itoa(int(rand.Intn(98)+1))
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(w int, h int, size string) (new_w int, new_h int) {
	var thumb_width int
	var thumb_height int

	switch {
	case size == "op":
		thumb_width = config.ThumbWidth
		thumb_height = config.ThumbHeight
	case size == "reply":
		thumb_width = config.ThumbWidth_reply
		thumb_height = config.ThumbHeight_reply
	case size == "catalog":
		thumb_width = config.ThumbWidth_catalog
		thumb_height = config.ThumbHeight_catalog
	}
	if w == h {
		new_w = thumb_width
		new_h = thumb_height
	} else {
		var percent float32
		if w > h {
			percent = float32(thumb_width) / float32(w)
		} else {
			percent = float32(thumb_height) / float32(h)
		}
		new_w = int(float32(w) * percent)
		new_h = int(float32(h) * percent)
	}
	return
}

// inserts prepared post object into the SQL table so that it can be rendered
func insertPost(post PostTable, bump bool) (sql.Result, error) {
	var result sql.Result
	insertString := "INSERT INTO " + config.DBprefix + "posts (`boardid`, `parentid`, `name`, `tripcode`, `email`, `subject`, `message`, `message_raw`, `password`, `filename`, `filename_original`, `file_checksum`, `filesize`, `image_w`, `image_h`, `thumb_w`, `thumb_h`, `ip`, `tag`, `timestamp`, `autosage`, `poster_authority`, `deleted_timestamp`,`bumped`,`stickied`, `locked`, `reviewed`, `sillytag`) "

	insertValues := "VALUES("
	numColumns := 28 // number of columns in the post table minus `id`
	for i := 0; i < numColumns-1; i++ {
		insertValues += "?, "
	}
	insertValues += " ? )"

	stmt, err := db.Prepare(insertString + insertValues)
	if err != nil {
		return nil, err
	}
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()
	result, err = stmt.Exec(
		post.BoardID, post.ParentID, post.Name, post.Tripcode,
		post.Email, post.Subject, post.MessageHTML, post.MessageText,
		post.Password, post.Filename, post.FilenameOriginal,
		post.FileChecksum, post.Filesize, post.ImageW, post.ImageH,
		post.ThumbW, post.ThumbH, post.IP, post.Tag, post.Timestamp,
		post.Autosage, post.PosterAuthority, post.DeletedTimestamp,
		post.Bumped, post.Stickied, post.Locked, post.Reviewed, post.Sillytag,
	)
	if err != nil {
		return result, err
	}

	// Bump parent post if requested.
	if post.ParentID != 0 && bump {
		err = bumpThread(post.ParentID, post.BoardID)
		if err != nil {
			return nil, err
		}
	}
	return result, err
}

// called when a user accesses /post. Parse form data, then insert and build
func makePost(w http.ResponseWriter, r *http.Request, data interface{}) {
	startTime := benchmarkTimer("makePost", time.Now(), true)
	request = *r
	writer = w
	var maxMessageLength int
	var errorText string
	var post PostTable
	domain := r.Host
	var formName string
	var nameCookie string
	var formEmail string

	// fix new cookie domain for when you use a port number
	chopPortNumRegex := regexp.MustCompile("(.+|\\w+):(\\d+)$")
	domain = chopPortNumRegex.Split(domain, -1)[0]

	post.IName = "post"
	post.ParentID, _ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID, _ = strconv.Atoi(request.FormValue("boardid"))

	var emailCommand string
	formName = request.FormValue("postname")

	if strings.Index(formName, "#") == -1 {
		post.Name = formName
	} else if strings.Index(formName, "#") == 0 {
		post.Tripcode = tripcode.Tripcode(formName[1:])
	} else if strings.Index(formName, "#") > 0 {
		postNameArr := strings.SplitN(formName, "#", 2)
		post.Name = postNameArr[0]
		post.Tripcode = tripcode.Tripcode(postNameArr[1])
	}

	nameCookie = post.Name + post.Tripcode
	formEmail = request.FormValue("postemail")
	http.SetCookie(writer, &http.Cookie{Name: "email", Value: formEmail, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})

	if strings.Index(formEmail, "noko") == -1 && strings.Index(formEmail, "sage") == -1 {
		post.Email = formEmail
	} else if strings.Index(formEmail, "#") > 1 {
		formEmailArr := strings.SplitN(formEmail, "#", 2)
		post.Email = formEmailArr[0]
		emailCommand = formEmailArr[1]
	} else if formEmail == "noko" || formEmail == "sage" {
		emailCommand = formEmail
		post.Email = ""
	}

	post.Subject = request.FormValue("postsubject")
	post.MessageText = strings.Trim(request.FormValue("postmsg"), "\r\n")

	stmt, err := db.Prepare("SELECT `max_message_length` from `" + config.DBprefix + "boards` WHERE `id` = ?")
	if err != nil {
		serveErrorPage(w, "Error getting board info.")
		errorLog.Print("Error getting board info: " + err.Error())
	}
	defer func() {
		if stmt != nil {
			stmt.Close()
		}
	}()
	err = stmt.QueryRow(post.BoardID).Scan(&maxMessageLength)

	if err != nil {
		serveErrorPage(w, "Requested board does not exist.")
		errorLog.Print("requested board does not exist. Error: " + err.Error())
	}

	if len(post.MessageText) > maxMessageLength {
		serveErrorPage(w, "Post body is too long")
		return
	}
	post.MessageHTML = html.EscapeString(post.MessageText)
	formatMessage(&post)

	post.Password = md5Sum(request.FormValue("postpassword"))

	// Reverse escapes
	nameCookie = strings.Replace(formName, "&amp;", "&", -1)
	nameCookie = strings.Replace(nameCookie, "\\&#39;", "'", -1)
	nameCookie = strings.Replace(url.QueryEscape(nameCookie), "+", "%20", -1)

	// add name and email cookies that will expire in a year (31536000 seconds)
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: nameCookie, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})
	http.SetCookie(writer, &http.Cookie{Name: "password", Value: request.FormValue("postpassword"), Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})

	post.IP = getRealIP(&request)
	post.Timestamp = time.Now()
	post.PosterAuthority = getStaffRank()
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !validReferrer(request) {
		accessLog.Print("Rejected post from possible spambot @ " + post.IP)
		//TODO: insert post into temporary post table and add to report list
		// or maybe not
		return
	}

	switch checkPostForSpam(post.IP, request.Header["User-Agent"][0], request.Referer(),
		post.Name, post.Email, post.MessageText) {
	case "discard":
		accessLog.Print("Akismet recommended discarding post from: " + post.IP)
		return
	case "spam":
		accessLog.Print("Akismet suggested post is spam from " + post.IP)
		return
	default:
	}

	file, handler, err := request.FormFile("imagefile")
	if err != nil {
		// no file was uploaded
		post.Filename = ""
		accessLog.Print("Receiving post from " + request.RemoteAddr + ", referred from: " + request.Referer())
	} else {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			serveErrorPage(w, "Couldn't read file: "+err.Error())
		} else {
			post.FilenameOriginal = html.EscapeString(handler.Filename)
			filetype := getFileExtension(post.FilenameOriginal)
			thumbFiletype := filetype

			if thumbFiletype == "gif" || thumbFiletype == "webm" {
				thumbFiletype = "jpg"
			}

			post.Filename = getNewFilename() + "." + getFileExtension(post.FilenameOriginal)
			boardArr, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
			if len(boardArr) == 0 {
				serveErrorPage(w, "No boards have been created yet")
			}
			_boardDir, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
			boardDir := _boardDir[0].Dir
			filePath := path.Join(config.DocumentRoot, "/"+boardDir+"/src/", post.Filename)

			thumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "t."+thumbFiletype, -1))

			catalogThumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "c."+thumbFiletype, -1))

			err := ioutil.WriteFile(filePath, data, 0777)
			if err != nil {
				errorText = "Couldn't write file \"" + post.Filename + "\"" + err.Error()
				println(0, errorText)
				errorLog.Println(errorText)
				serveErrorPage(w, "Couldn't write file \""+post.FilenameOriginal+"\"")
				return
			}

			// Calculate image checksum
			post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

			var allowsVids bool
			vidStmt, err := db.Prepare("SELECT `embeds_allowed` FROM `" + config.DBprefix + "boards` WHERE `id` = ? LIMIT 1")
			if err != nil {
				errortext := err.Error()
				errorLog.Println(errorText)
				serveErrorPage(w, "Couldn't get board info: "+errorText)
				println(1, errortext)
				return
			}

			defer func() {
				if vidStmt != nil {
					vidStmt.Close()
				}
			}()

			err = vidStmt.QueryRow(post.BoardID).Scan(&allowsVids)
			if err != nil {
				errortext := err.Error()
				errorLog.Println(errorText)
				serveErrorPage(w, "Couldn't get board info: "+errorText)
				println(1, errortext)
				return
			}

			if filetype == "webm" {
				if !allowsVids || !config.AllowVideoUploads {
					serveErrorPage(w, "Video uploading is not currently enabled for this board.")
					os.Remove(filePath)
					return
				}

				accessLog.Print("Receiving post with video: " + handler.Filename + " from " + request.RemoteAddr + ", referrer: " + request.Referer())
				if post.ParentID == 0 {
					err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth)
					if err != nil {
						serveErrorPage(w, err.Error())
						printf(1, err.Error())
						return
					}
				} else {
					err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth_reply)
					if err != nil {
						serveErrorPage(w, err.Error())
						printf(1, err.Error())
						return
					}
				}

				err := createVideoThumbnail(filePath, catalogThumbPath, config.ThumbWidth_catalog)
				if err != nil {
					serveErrorPage(w, err.Error())
					printf(1, err.Error())
					return
				}

				outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
				if err != nil {
					errortext := "Error getting video info: " + err.Error()
					serveErrorPage(w, errortext)
					printf(1, errortext)
					return
				}
				if err == nil && outputBytes != nil {
					outputStringArr := strings.Split(string(outputBytes), "\n")
					for _, line := range outputStringArr {
						lineArr := strings.Split(line, "=")
						if len(lineArr) < 2 {
							continue
						}
						value, _ := strconv.Atoi(lineArr[1])
						switch lineArr[0] {
						case "width":
							post.ImageW = value
						case "height":
							post.ImageH = value
						case "size":
							post.Filesize = value
						}
					}
					if post.ParentID == 0 {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "op")
					} else {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "reply")
					}
				}

			} else {
				// Attempt to load uploaded file with imaging library
				img, err := imaging.Open(filePath)
				if err != nil {
					errorText = "Couldn't open uploaded file \"" + post.Filename + "\"" + err.Error()
					errorLog.Println(errorText)
					println(1, errorText)
					os.Remove(filePath)
					serveErrorPage(w, "Upload filetype not supported")
					return
				} else {
					// Get image filesize
					stat, err := os.Stat(filePath)
					if err != nil {
						errortext := "Couldn't get image filesize: " + err.Error()
						errorLog.Println(errortext)
						println(1, errortext)
						serveErrorPage(w, errortext)
						return
					} else {
						post.Filesize = int(stat.Size())
					}

					// Get image width and height, as well as thumbnail width and height
					post.ImageW = img.Bounds().Max.X
					post.ImageH = img.Bounds().Max.Y
					if post.ParentID == 0 {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "op")
					} else {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "reply")
					}

					accessLog.Print("Receiving post with image: " + handler.Filename + " from " + request.RemoteAddr + ", referrer: " + request.Referer())

					if request.FormValue("spoiler") == "on" {
						// If spoiler is enabled, symlink thumbnail to spoiler image
						_, err := os.Stat(path.Join(config.DocumentRoot, "spoiler.png"))
						if err != nil {
							serveErrorPage(w, "missing /spoiler.png")
							return
						} else {
							err = syscall.Symlink(path.Join(config.DocumentRoot, "spoiler.png"), thumbPath)
							if err != nil {
								serveErrorPage(w, err.Error())
								return
							}
						}
					} else if config.ThumbWidth >= post.ImageW && config.ThumbHeight >= post.ImageH {
						// If image fits in thumbnail size, symlink thumbnail to original
						post.ThumbW = img.Bounds().Max.X
						post.ThumbH = img.Bounds().Max.Y
						err := syscall.Symlink(filePath, thumbPath)
						if err != nil {
							serveErrorPage(w, err.Error())
							return
						}
					} else {
						var thumbnail image.Image
						var catalogThumbnail image.Image
						if post.ParentID == 0 {
							// If this is a new thread, generate thumbnail and catalog thumbnail
							thumbnail = createImageThumbnail(img, "op")
							catalogThumbnail = createImageThumbnail(img, "catalog")
							println(1, catalogThumbPath)
							err = imaging.Save(catalogThumbnail, catalogThumbPath)
							if err != nil {
								errorLog.Println("Couldn't generate catalog thumbnail: " + err.Error())
								serveErrorPage(w, "Couldn't generate catalog thumbnail: "+err.Error())
								return
							}
						} else {
							thumbnail = createImageThumbnail(img, "reply")
						}
						err = imaging.Save(thumbnail, thumbPath)
						if err != nil {
							errortext := "Couldn't save thumbnail: " + err.Error()
							println(0, errortext)
							errorLog.Println(errortext)
							serveErrorPage(w, errortext)
							return
						}
					}
				}
			}
		}
	}
	if post.FilenameOriginal != "" {
		//post.FilenameOriginal = html.EscapeString(post.FilenameOriginal)
	}

	if strings.TrimSpace(post.MessageText) == "" && post.Filename == "" {
		serveErrorPage(w, "Post must contain a message if no image is uploaded.")
		return
	}

	postDelay := sinceLastPost(&post)
	if postDelay > -1 {
		if post.ParentID == 0 && postDelay < config.NewThreadDelay {
			serveErrorPage(w, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && postDelay < config.ReplyDelay {
			serveErrorPage(w, "Please wait before making a reply.")
			return
		}
	}

	isBanned, err := checkBannedStatus(&post, &w)
	if err != nil {
		errorText = "Error in checkBannedStatus: " + err.Error()
		serveErrorPage(w, err.Error())
		errorLog.Println(errorText)
		println(1, errorText)
		return
	}

	if len(isBanned) > 0 {
		var banpage_buffer bytes.Buffer
		var banpage_html string
		banpage_buffer.Write([]byte(""))
		err = renderTemplate(banpage_tmpl, "banpage", &banpage_buffer, &Wrapper{IName: "bans", Data: isBanned})
		if err != nil {
			fmt.Fprintf(writer, banpage_html+err.Error()+"\n</body>\n</html>")
			println(1, err.Error())
			errorLog.Println(err.Error())
			return
		}
		fmt.Fprintf(w, banpage_buffer.String())
		return
	}

	post = sanitizePost(post)
	result, err := insertPost(post, emailCommand != "sage")
	if err != nil {
		serveErrorPage(w, err.Error())
		return
	}
	postid, _ := result.LastInsertId()
	post.ID = int(postid)

	boards, _ := getBoardArr(nil, "")
	// rebuild the board page
	buildBoards(false, post.BoardID)

	buildFrontPage()

	if emailCommand == "noko" {
		if post.ParentID == 0 {
			http.Redirect(writer, &request, "/"+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, &request, "/"+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, &request, "/"+boards[post.BoardID-1].Dir+"/", http.StatusFound)
	}
	benchmarkTimer("makePost", startTime, false)
}

func formatMessage(post *PostTable) {
	message := post.MessageHTML

	// prepare each line to be formatted
	postLines := strings.Split(message, "\\r\\n")
	for i, line := range postLines {
		trimmedLine := strings.TrimSpace(line)
		lineWords := strings.Split(trimmedLine, " ")
		isGreentext := false // if true, append </span> to end of line
		for w, word := range lineWords {
			if strings.LastIndex(word, gt+gt) == 0 {
				//word is a backlink
				_, err := strconv.Atoi(word[8:])
				if err == nil {
					// the link is in fact, a valid int
					var boardDir string
					var linkParent int
					stmt, _ := db.Prepare("SELECT `dir`,`parentid` FROM " + config.DBprefix + "posts," + config.DBprefix + "boards WHERE " + config.DBprefix + "posts.id = ?")
					stmt.QueryRow(word[8:]).Scan(&boardDir, &linkParent)
					// get post board dir

					if boardDir == "" {
						lineWords[w] = "<a href=\"javascript:;\"><strike>" + word + "</strike></a>"
					} else if linkParent == 0 {
						lineWords[w] = "<a href=\"/" + boardDir + "/res/" + word[8:] + ".html\">" + word + "</a>"
					} else {
						lineWords[w] = "<a href=\"/" + boardDir + "/res/" + strconv.Itoa(linkParent) + ".html#" + word[8:] + "\">" + word + "</a>"
					}
				}
			} else if strings.Index(word, gt) == 0 && w == 0 {
				// word is at the beginning of a line, and is greentext
				isGreentext = true
				lineWords[w] = "<span class=\"greentext\">" + word
			}
		}
		line = strings.Join(lineWords, " ")
		if isGreentext {
			line += "</span>"
		}
		postLines[i] = line
	}
	post.MessageHTML = strings.Join(postLines, "<br />")
}
