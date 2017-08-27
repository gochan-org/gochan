// functions for handling posting, uploading, and post/thread/board page building

package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/nyarla/go-crypt"
	"html"
	"image"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	whitespace_match = "[\000-\040]"
	gt = "&gt;"
)

var (
	last_post    PostTable
	all_sections []interface{}
	all_boards   []interface{}
)

func generateTripCode(input string) string {
	re := regexp.MustCompile("[^\\.-z]") // remove every ASCII character before . and after z

	input += "   " // padding
	input = strings.Replace(input, "&amp;", "&", -1)
	input = strings.Replace(input, "\\&#39;", "'", -1)

	salt := string(re.ReplaceAllLiteral([]byte(input), []byte(".")))
	salt = byteByByteReplace(salt[1:3], ":;<=>?@[\\]^_`", "ABCDEFGabcdef") // stole-I MEAN BORROWED from Kusaba X
	return crypt.Crypt(input, salt)[3:]
}

// buildBoards builds one or all boards. If all == true, all boards will have their pages built.
// If all == false, the board with the id equal to the value specified as which.
// The return value is a string of HTML with debug information produced by the build process.
func buildBoards(all bool, which int) (html string) {
	// if all is set to true, ignore which, otherwise, which = build only specified boardid
	if !all {
		_board, _ := getBoardArr("`id` = " + strconv.Itoa(which))
		board := _board[0]
		html += buildBoardPages(&board) + "<br />\n"
		html += buildThreads(true, board.ID, 0)
		return
	}
	boards, _ := getBoardArr("")
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
	var interfaces []interface{}
	var threads []interface{}
	var thread_pages [][]interface{}
	var stickied_threads []interface{}
	var nonstickied_threads []interface{}
	var errortext string

	defer func() {
		// This function cleans up after we return. If there was an error, it prints on the log and the console.
		if uhoh, ok := recover().(error); ok {
			error_log.Print("buildBoardPages failed: " + uhoh.Error())
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
			error_log.Println(errortext)
			println(1, errortext)
		}
	} else if !results.IsDir() {
		// If the file exists, but is not a folder, notify the user
		errortext = "Error: /" + board.Dir + "/ exists, but is not a folder."
		html += errortext + "<br />\n"
		error_log.Println(errortext)
		println(1, errortext)
	}

	// Get all top level posts for the board.
	op_posts, err := getPostArr("SELECT * FROM " + config.DBprefix + "posts WHERE `boardid` = " +
		strconv.Itoa(board.ID) + " AND `parentid` = 0 AND `deleted_timestamp` = '" + nil_timestamp + "' ORDER BY `bumped` DESC")
	if err != nil {
		html += err.Error() + "<br />"
		op_posts = make([]interface{}, 0)
		return
	}

	// For each top level post, start building a Thread struct
	for _, op_post_i := range op_posts {
		var thread Thread
		var posts_in_thread []interface{}

		thread.IName = "thread"

		// Store the OP post for this thread
		op_post := op_post_i.(PostTable)

		if op_post.Stickied {
			// If the thread is stickied, limit replies on the archive page to the
			// 	configured value for stickied threads.
			posts_in_thread, err = getPostArr("SELECT * FROM (SELECT * FROM " + config.DBprefix + "posts WHERE `boardid` = " + strconv.Itoa(board.ID) + " AND `parentid` = " + strconv.Itoa(op_post.ID) + " AND `deleted_timestamp` = '" + nil_timestamp + "' ORDER BY `id` DESC LIMIT " + strconv.Itoa(config.StickyRepliesOnBoardPage) + ") AS posts ORDER BY id ASC")
			if err != nil {
				html += err.Error() + "<br />"
			}
			// Get the number of replies to this thread.
			err = db.QueryRow("SELECT COUNT(*) FROM `" + config.DBprefix + "posts` WHERE `boardid` = " + strconv.Itoa(board.ID) + " AND `parentid` = " + strconv.Itoa(op_post.ID) + " AND `deleted_timestamp` = '" + nil_timestamp + "'").Scan(&thread.NumReplies)
			if err != nil {
				html += err.Error() + "<br />"
			}
			thread.OP = op_post_i
			if len(posts_in_thread) > 0 {
				thread.BoardReplies = posts_in_thread
			}
			stickied_threads = append(stickied_threads, thread)
		} else {
			// Otherwise, limit the replies to the configured value for normal threads.
			posts_in_thread, err = getPostArr("SELECT * FROM (SELECT * FROM " + config.DBprefix + "posts WHERE `boardid` = " + strconv.Itoa(board.ID) + " AND `parentid` = " + strconv.Itoa(op_post.ID) + " AND `deleted_timestamp` = '" + nil_timestamp + "' ORDER BY `id` DESC LIMIT " + strconv.Itoa(config.RepliesOnBoardPage) + ") AS posts ORDER BY id ASC")
			if err != nil {
				html += err.Error() + "<br />"
			}
			// Get the number of replies to this thread.
			err = db.QueryRow("SELECT COUNT(*) FROM `" + config.DBprefix + "posts` WHERE `boardid` = " + strconv.Itoa(board.ID) + " AND `parentid` = " + strconv.Itoa(op_post.ID) + " AND `deleted_timestamp` = '" + nil_timestamp + "'").Scan(&thread.NumReplies)
			if err != nil {
				html += err.Error() + "<br />"
			}
			thread.OP = op_post_i
			if len(posts_in_thread) > 0 {
				thread.BoardReplies = posts_in_thread
			}
			nonstickied_threads = append(nonstickied_threads, thread)
		}
	}

	num, _ := deleteMatchingFiles(path.Join(config.DocumentRoot, board.Dir), "\\d.html$")
	printf(2, "Number of files deleted: %d\n", num)
	// Order the threads, stickied threads first, then nonstickied threads.
	threads = append(stickied_threads, nonstickied_threads...)
	if len(threads) == 0 {
		board.CurrentPage = 0
		boardinfo_i = nil
		boardinfo_i = append(boardinfo_i, board)

		// Package up boards, sections, threads, the boardinfo for the template to use.
		interfaces = nil
		interfaces = append(interfaces, config,
			&Wrapper{IName: "boards", Data: all_boards},
			&Wrapper{IName: "sections", Data: all_sections},
			&Wrapper{IName: "threads", Data: threads},
			&Wrapper{IName: "boardinfo", Data: boardinfo_i})
		wrapped := &Wrapper{IName: "boardpage", Data: interfaces}

		// Write to board.html for the first page.
		printf(1, "Current page: %s/%d\n", board.Dir, board.CurrentPage)
		board_page_file, err := os.OpenFile(path.Join(config.DocumentRoot, board.Dir, "board.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			errortext = "Failed opening /" + board.Dir + "/board.html: " + err.Error()
			html += errortext + "<br />\n"
			error_log.Println(errortext)
			println(1, errortext)
			return
		}

		// Run the template, pointing it to the file, and passing in the data required.
		err = img_boardpage_tmpl.Execute(board_page_file, wrapped)
		if err != nil {
			errortext = "Failed building /" + board.Dir + "/: " + err.Error()
			html += errortext + "<br />\n"
			error_log.Print(errortext)
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
		for page_num, page_threads := range thread_pages {
			// Package up board info for the template to use.
			board.CurrentPage = page_num
			boardinfo_i = nil
			boardinfo_i = append(boardinfo_i, board)

			// Package up boards, sections, threads, the boardinfo for the template to use.
			interfaces = nil
			interfaces = append(interfaces, config,
				&Wrapper{IName: "boards", Data: all_boards},
				&Wrapper{IName: "sections", Data: all_sections},
				&Wrapper{IName: "threads", Data: page_threads},
				&Wrapper{IName: "boardinfo", Data: boardinfo_i})
			wrapped := &Wrapper{IName: "boardpage", Data: interfaces}

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
				error_log.Println(errortext)
				println(1, errortext)
				continue
			}

			// Run the template, pointing it to the file, and passing in the data required.
			err = img_boardpage_tmpl.Execute(current_page_file, wrapped)
			if err != nil {
				errortext = "Failed building /" + board.Dir + "/: " + err.Error()
				html += errortext + "<br />\n"
				error_log.Print(errortext)
				println(1, errortext)
				return
			}
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
		_thread, _ := getPostArr("SELECT * FROM " + config.DBprefix + "posts WHERE `boardid` = " + strconv.Itoa(boardid) + " AND `id` = " + strconv.Itoa(threadid) + " AND `parentid` = 0 AND `deleted_timestamp` = '" + nil_timestamp + "'")
		thread := _thread[0]
		thread_struct := thread.(PostTable)
		html += buildThreadPages(&thread_struct) + "<br />\n"
		return
	}
	threads, _ := getPostArr("SELECT * FROM " + config.DBprefix + "posts WHERE `boardid` = " + strconv.Itoa(boardid) + " AND `parentid` = 0 AND `deleted_timestamp` = '" + nil_timestamp + "'")
	if len(threads) == 0 {
		return
	}
	for _, op := range threads {
		op_struct := op.(PostTable)
		html += buildThreadPages(&op_struct) + "<br />\n"
	}
	return
}

// buildThreadPages builds the pages for a thread given by a PostTable object.
func buildThreadPages(op *PostTable) (html string) {
	var board_dir string
	var replies []interface{}
	var interfaces []interface{}
	var page []interface{}
	var current_page_file *os.File
	var errortext string

	err := db.QueryRow("SELECT `dir` FROM `" + config.DBprefix + "boards` WHERE `id` = '" + strconv.Itoa(op.BoardID) + "';").Scan(&board_dir)
	if err != nil {
		errortext = "Failed getting board directory from post: " + err.Error()
		html += errortext + "<br />\n"
		error_log.Println(errortext)
		println(1, errortext)
		return
	}

	replies, err = getPostArr("SELECT * FROM " + config.DBprefix + "posts WHERE `boardid` = " + strconv.Itoa(op.BoardID) + " AND `parentid` = " + strconv.Itoa(op.ID) + " AND `deleted_timestamp` = '" + nil_timestamp + "'")
	if err != nil {
		errortext = "Error building thread " + strconv.Itoa(op.ID) + ":" + err.Error()
		html += errortext
		error_log.Println(errortext)
		println(1, errortext)
		return
	}
	os.Remove(path.Join(config.DocumentRoot, board_dir, "res", strconv.Itoa(op.ID)+".html"))

	thread_pages := paginate(config.PostsPerThreadPage, replies)
	for i, _ := range thread_pages {
		thread_pages[i] = append([]interface{}{op}, thread_pages[i]...)
	}
	deleteMatchingFiles(path.Join(config.DocumentRoot, board_dir, "res"), "^"+strconv.Itoa(op.ID)+"p")

	op.NumPages = len(thread_pages)
	// build main page
	page = append([]interface{}{op}, replies...)
	interfaces = append(interfaces, config,
		&Wrapper{IName: "boards_", Data: all_boards},
		&Wrapper{IName: "sections_w", Data: all_sections},
		&Wrapper{IName: "posts_w", Data: page})
	wrapped := &Wrapper{IName: "threadpage", Data: interfaces}

	current_page_filepath := path.Join(config.DocumentRoot, board_dir, "res", strconv.Itoa(op.ID)+".html")
	current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		errortext = "Failed opening " + current_page_filepath + ": " + err.Error()
		html += errortext + "<br />\n"
		println(1, errortext)
		error_log.Println(errortext)
		return
	}
	err = img_threadpage_tmpl.Execute(current_page_file, wrapped)
	if err != nil {
		errortext = "Failed building /" + board_dir + "/res/" + strconv.Itoa(op.ID) + ".html: " + err.Error()
		html += errortext + "<br />\n"
		println(1, errortext)
		error_log.Print(errortext)
		return
	}

	// Put together the thread JSON
	thread_json_file, err := os.OpenFile(path.Join(config.DocumentRoot, board_dir, "res", strconv.Itoa(op.ID)+".json"), os.O_CREATE | os.O_RDWR | os.O_TRUNC, 0777)
	defer func()  {
		if thread_json_file != nil {
			thread_json_file.Close()
		}
	}()
	if err != nil {
		errortext = "Failed opening /" + board_dir + "/res/" + strconv.Itoa(op.ID) + ".json: " + err.Error()
		html += errortext + "<br />\n"
		println(1, errortext)
		error_log.Print(errortext)
		return
	}
	// Create the wrapper object
	thread_json_wrapper := new(ThreadJSONWrapper)

	// Handle the OP, of type *PostTable
	var filename string
	var fileext string
	var orig_filename string

	// Separate out the extension from the filename
	if op.Filename != "deleted" && op.Filename != "" {
		ext_start := strings.LastIndex(op.Filename, ".")
		fileext = op.Filename[ext_start:]

		orig_ext_start := strings.LastIndex(op.FilenameOriginal, fileext)
		orig_filename = op.FilenameOriginal[:orig_ext_start]
		filename = op.Filename[:ext_start]
	}

	op_post_obj := PostJSON { ID: op.ID, ParentID: op.ParentID, Subject: op.Subject, Message: op.MessageHTML,
		Name: op.Name, Tripcode: "!" + op.Tripcode, Timestamp: op.Timestamp.Unix(), Bumped: op.Bumped.Unix(),
		ThumbWidth: op.ThumbW, ThumbHeight: op.ThumbH, ImageWidth: op.ImageW, ImageHeight: op.ImageH,
		FileSize: op.Filesize, OrigFilename: orig_filename, Extension: fileext, Filename: filename, FileChecksum: op.FileChecksum}

	thread_json_wrapper.Posts = append(thread_json_wrapper.Posts, op_post_obj)

	// Iterate through each reply, which are of type PostTable
	for _, post_int := range replies {
		post := post_int.(PostTable)

		// Separate out the extension from the filenames
		if post.Filename != "deleted" && post.Filename != "" {
			ext_start := strings.LastIndex(post.Filename, ".")
			fileext = post.Filename[ext_start:]

			orig_ext_start := strings.LastIndex(post.FilenameOriginal, fileext)
			orig_filename = post.FilenameOriginal[:orig_ext_start]
			filename = post.Filename[:ext_start]
		} else {
			filename = ""
			orig_filename = ""
			fileext = ""
		}

		post_obj := PostJSON { ID: post.ID, ParentID: post.ParentID, Subject: post.Subject, Message: post.MessageHTML,
		 	Name: post.Name, Tripcode: "!" + post.Tripcode, Timestamp: post.Timestamp.Unix(), Bumped: post.Bumped.Unix(),
			ThumbWidth: post.ThumbW, ThumbHeight: post.ThumbH, ImageWidth: post.ImageW, ImageHeight: post.ImageH,
			FileSize: post.Filesize, OrigFilename: orig_filename, Extension: fileext, Filename: filename, FileChecksum: post.FileChecksum}

		thread_json_wrapper.Posts = append(thread_json_wrapper.Posts, post_obj)
	}
	thread_json, err := json.Marshal(thread_json_wrapper)

	if err != nil {
		errortext = "Failed to marshal to JSON: " + err.Error()
		error_log.Println(errortext)
		println(1, errortext)
		html += errortext + "<br />\n"
		return
	}

	_, err = thread_json_file.Write(thread_json)

	if err != nil {
		errortext = "Failed writing /" + board_dir + "/res/" + strconv.Itoa(op.ID) + ".json: " + err.Error()
		error_log.Println(errortext)
		println(1, errortext)
		html += errortext + "<br />\n"
		return
	}

	success_text := "Built /" + board_dir + "/" + strconv.Itoa(op.ID) + " successfully"
	html += success_text + "<br />\n"
	println(2, success_text)

	for page_num, page_posts := range thread_pages {
		op.CurrentPage = page_num
		interfaces = nil
		interfaces = append(interfaces, config,
			&Wrapper{IName: "boards_", Data: all_boards},
			&Wrapper{IName: "sections_w", Data: all_sections},
			&Wrapper{IName: "posts_w", Data: page_posts})

		wrapped := &Wrapper{IName: "threadpage", Data: interfaces}

		var current_page_filepath string
		current_page_filepath = path.Join(config.DocumentRoot, board_dir, "res", strconv.Itoa(op.ID)+"p"+strconv.Itoa(op.CurrentPage+1)+".html")

		current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			errortext = "Failed opening " + current_page_filepath + ": " + err.Error()
			html += errortext + "<br />\n"
			println(1, errortext)
			error_log.Println(errortext)
			return
		}

		err = img_threadpage_tmpl.Execute(current_page_file, wrapped)
		if err != nil {
			errortext = "Failed building /" + board_dir + "/" + strconv.Itoa(op.ID) + ": " + err.Error()
			html += errortext + "<br />\n"
			println(1, errortext)
			error_log.Print(errortext)
			return
		}
		success_text := "Built /" + board_dir + "/" + strconv.Itoa(op.ID) + "p" + strconv.Itoa(op.CurrentPage+1) + " successfully"
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
		error_log.Println(errortext)
		return errortext + "<br />\n"
	}

	// get front pages
	rows, err := db.Query("SELECT * FROM `" + config.DBprefix + "frontpage`;")
	if err != nil {
		errortext = "Failed getting front page rows: " + err.Error()
		error_log.Print(errortext)
		return errortext + "<br />"
	}
	for rows.Next() {
		frontpage := new(FrontTable)
		frontpage.IName = "front page"
		err = rows.Scan(&frontpage.ID, &frontpage.Page, &frontpage.Order, &frontpage.Subject, &frontpage.Message, &frontpage.Timestamp, &frontpage.Poster, &frontpage.Email)
		if err != nil {
			error_log.Print(err.Error())
			println(1, err.Error())
			return err.Error()
		}
		front_arr = append(front_arr, frontpage)
	}

	// get recent posts
	rows, err = db.Query("SELECT `" + config.DBprefix + "posts`.`id`, " +
		"`" + config.DBprefix + "posts`.`parentid`, " +
		"`" + config.DBprefix + "boards`.`dir` AS boardname, " +
		"`" + config.DBprefix + "posts`.`boardid` AS boardid, " +
		"`name`, " +
		"`tripcode`, " +
		"`message`, " +
		"`filename`, " +
		"`thumb_w`, " +
		"`thumb_h` " +
		"FROM `" + config.DBprefix + "posts`, " +
		"`" + config.DBprefix + "boards` " +
		"WHERE `" + config.DBprefix + "posts`.`deleted_timestamp` = \"" + nil_timestamp + "\"" +
		"AND `boardid` = `" + config.DBprefix + "boards`.`id` " +
		"ORDER BY `timestamp` DESC " +
		"LIMIT " + strconv.Itoa(config.MaxRecentPosts))
	if err != nil {
		errortext = "Failed getting list of recent posts for front page: " + err.Error()
		error_log.Print(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	for rows.Next() {
		recent_post := new(RecentPost)
		err = rows.Scan(&recent_post.PostID, &recent_post.ParentID, &recent_post.BoardName, &recent_post.BoardID, &recent_post.Name, &recent_post.Tripcode, &recent_post.Message, &recent_post.Filename, &recent_post.ThumbW, &recent_post.ThumbH)
		if err != nil {
			errortext = "Failed getting list of recent posts for front page: " + err.Error()
			error_log.Print(errortext)
			println(1, errortext)
			return errortext + "<br />\n"
		}
		recent_posts_arr = append(recent_posts_arr, recent_post)
	}

	var interfaces []interface{}
	interfaces = append(interfaces, config,
		&Wrapper{IName: "fronts", Data: front_arr},
		&Wrapper{IName: "boards", Data: all_boards},
		&Wrapper{IName: "sections", Data: all_sections},
		&Wrapper{IName: "recent posts", Data: recent_posts_arr})

	wrapped := &Wrapper{IName: "frontpage", Data: interfaces}
	err = front_page_tmpl.Execute(front_file, wrapped)
	if err != nil {
		errortext = "Failed executing front page template: " + err.Error()
		error_log.Println(errortext)
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
		error_log.Println(errortext)
		return errortext + "<br />\n"
	}

	board_list_wrapper := new(BoardJSONWrapper)

	// Our cooldowns are site-wide currently.
	cooldowns_obj := BoardCooldowns { NewThread: config.NewThreadDelay, Reply: config.ReplyDelay, ImageReply: config.ReplyDelay}

	for _, board_int := range all_boards {
		board := board_int.(BoardsTable)
		board_obj := BoardJSON { BoardName: board.Dir, Title: board.Title, WorkSafeBoard: 1,
			ThreadsPerPage: config.ThreadsPerPage_img, Pages: board.MaxPages, MaxFilesize: board.MaxImageSize,
			MaxMessageLength: board.MaxMessageLength, BumpLimit: 200, ImageLimit: board.NoImagesAfter,
			Cooldowns: cooldowns_obj, Description: board.Description, IsArchived: 0 }
		if(board.EnableNSFW) {
			board_obj.WorkSafeBoard = 0
		}
		board_list_wrapper.Boards = append(board_list_wrapper.Boards, board_obj)
	}

	board_json, err := json.Marshal(board_list_wrapper)

	if err != nil {
		errortext = "Failed marshal to JSON: " + err.Error()
		error_log.Println(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	_, err = board_list_file.Write(board_json)

	if err != nil {
		errortext = "Failed writing boards.json file: " + err.Error()
		error_log.Println(errortext)
		println(1, errortext)
		return errortext + "<br />\n"
	}
	return "Board list JSON rebuilt successfully.<br />"
}

// Checks check poster's name/tripcode/file checksum (from PostTable post) for banned status
// returns true if the user is banned
func checkBannedStatus(post *PostTable, writer *http.ResponseWriter) ([]interface{}, error) {
	var is_expired bool
	var ban_entry BanlistTable
	// var count int
	// var search string
	err := db.QueryRow("SELECT `ip`, `name`, `tripcode`, `message`, `boards`, `timestamp`, `expires`, `appeal_at` FROM `"+config.DBprefix+"banlist` WHERE `ip` = '"+post.IP+"'").Scan(&ban_entry.IP, &ban_entry.Name, &ban_entry.Tripcode, &ban_entry.Message, &ban_entry.Boards, &ban_entry.Timestamp, &ban_entry.Expires, &ban_entry.AppealAt)
	var interfaces []interface{}

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

func createThumbnail(image_obj image.Image, size string) image.Image {
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
	old_rect := image_obj.Bounds()
	if thumb_width >= old_rect.Max.X && thumb_height >= old_rect.Max.Y {
		return image_obj
	}

	thumb_w, thumb_h := getThumbnailSize(old_rect.Max.X, old_rect.Max.Y, size)
	image_obj = imaging.Resize(image_obj, thumb_w, thumb_h, imaging.CatmullRom) // resize to 600x400 px using CatmullRom cubic filter
	return image_obj
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
	post_sql_str := "INSERT INTO `" + config.DBprefix + "posts` (`boardid`,`parentid`,`name`,`tripcode`,`email`,`subject`,`message`,`message_raw`,`password`"
	if post.Filename != "" {
		post_sql_str += ",`filename`,`filename_original`,`file_checksum`,`filesize`,`image_w`,`image_h`,`thumb_w`,`thumb_h`"
	}
	post_sql_str += ",`ip`"
	post_sql_str += ",`timestamp`,`poster_authority`,"
	if post.ParentID == 0 {
		post_sql_str += "`bumped`,"
	}
	post_sql_str += "`stickied`,`locked`) VALUES(" + strconv.Itoa(post.BoardID) + "," + strconv.Itoa(post.ParentID) + ",'" + post.Name + "','" + post.Tripcode + "','" + post.Email + "','" + post.Subject + "','" + post.MessageHTML + "','" + post.MessageText + "','" + post.Password + "'"
	if post.Filename != "" {
		post_sql_str += ",'" + post.Filename + "','" + post.FilenameOriginal + "','" + post.FileChecksum + "'," + strconv.Itoa(int(post.Filesize)) + "," + strconv.Itoa(post.ImageW) + "," + strconv.Itoa(post.ImageH) + "," + strconv.Itoa(post.ThumbW) + "," + strconv.Itoa(post.ThumbH)
	}
	post_sql_str += ",'" + post.IP + "','" + getSpecificSQLDateTime(post.Timestamp) + "'," + strconv.Itoa(post.PosterAuthority) + ","
	if post.ParentID == 0 {
		post_sql_str += "'" + getSpecificSQLDateTime(post.Bumped) + "',"
	}
	if post.Stickied {
		post_sql_str += "1,"
	} else {
		post_sql_str += "0,"
	}
	if post.Locked {
		post_sql_str += "1);"
	} else {
		post_sql_str += "0);"
	}
	result, err := db.Exec(post_sql_str)
	if err != nil {
		return result, err
	}
	if post.ParentID != 0 {
		_, err := db.Exec("UPDATE `" + config.DBprefix + "posts` SET `bumped` = '" + getSpecificSQLDateTime(post.Bumped) + "' WHERE `id` = " + strconv.Itoa(post.ParentID))
		if err != nil {
			return result, err
		}
	}
	return result, err
}

func makePost(w http.ResponseWriter, r *http.Request, data interface{}) {
	start_time := benchmarkTimer("makePost", time.Now(), true)
	request = *r
	writer = w
	var max_message_length int
	var errortext string
	domain := r.Host

	chopPortNumRegex := regexp.MustCompile("(.+|\\w+):(\\d+)$")
	domain = chopPortNumRegex.Split(domain, -1)[0]

	var post PostTable
	post.IName = "post"
	post.ParentID, _ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID, _ = strconv.Atoi(request.FormValue("boardid"))

	var email_command string

	post_name := html.EscapeString(escapeString(request.FormValue("postname")))

	if strings.Index(post_name, "#") == -1 {
		post.Name = post_name
	} else if strings.Index(post_name, "#") == 0 {
		post.Tripcode = generateTripCode(post_name[1:])
	} else if strings.Index(post_name, "#") > 0 {
		post_name_arr := strings.SplitN(post_name, "#", 2)
		post.Name = post_name_arr[0]
		post.Tripcode = generateTripCode(post_name_arr[1])
	}

	post_email := escapeString(request.FormValue("postemail"))
	if strings.Index(post_email, "noko") == -1 && strings.Index(post_email, "sage") == -1 {
		post.Email = html.EscapeString(escapeString(post_email))
	} else if strings.Index(post_email, "#") > 1 {
		post_email_arr := strings.SplitN(post_email, "#", 2)
		post.Email = html.EscapeString(escapeString(post_email_arr[0]))
		email_command = post_email_arr[1]
	} else if post_email == "noko" || post_email == "sage" {
		email_command = post_email
		post.Email = ""
	}
	post.Subject = html.EscapeString(escapeString(request.FormValue("postsubject")))
	post.MessageText = strings.Trim(escapeString(request.FormValue("postmsg")),"\r\n")

	err := db.QueryRow("SELECT `max_message_length` FROM `" + config.DBprefix + "boards` WHERE `id` = " + strconv.Itoa(post.BoardID)).Scan(&max_message_length)
	if err != nil {
		server.ServeErrorPage(w, "Requested board does not exist.")
		error_log.Print("requested board does not exist. Error: " + err.Error())
	}

	if len(post.MessageText) > max_message_length {
		server.ServeErrorPage(w, "Post body is too long")
		return
	}
	post.MessageHTML = html.EscapeString(post.MessageText)
	formatMessage(&post)

	post.Password = md5_sum(request.FormValue("postpassword"))

	// Reverse escapes
	post_name_cookie := strings.Replace(post_name, "&amp;", "&", -1)
	post_name_cookie = strings.Replace(post_name_cookie, "\\&#39;", "'", -1)

	post_name_cookie = strings.Replace(url.QueryEscape(post_name_cookie), "+", "%20", -1)

	http.SetCookie(writer, &http.Cookie{Name: "name", Value: post_name_cookie, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))), MaxAge: 31536000})
	// http.SetCookie(writer, &http.Cookie{Name: "name", Value: post_name_cookie, Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})
	if email_command == "" {
		http.SetCookie(writer, &http.Cookie{Name: "email", Value: post.Email, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))), MaxAge: 31536000})
		// http.SetCookie(writer, &http.Cookie{Name: "email", Value: post.Email, Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})
	} else {
		if email_command == "noko" {
			if post.Email == "" {
				http.SetCookie(writer, &http.Cookie{Name: "email", Value: "noko", Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))), MaxAge: 31536000})
				// http.SetCookie(writer, &http.Cookie{Name: "email", Value:"noko", Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})
			} else {
				http.SetCookie(writer, &http.Cookie{Name: "email", Value: post.Email + "#noko", Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))), MaxAge: 31536000})
				//http.SetCookie(writer, &http.Cookie{Name: "email", Value: post.Email + "#noko", Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})
			}
		}
	}

	http.SetCookie(writer, &http.Cookie{Name: "password", Value: request.FormValue("postpassword"), Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))), MaxAge: 31536000})
	//http.SetCookie(writer, &http.Cookie{Name: "password", Value: request.FormValue("postpassword"), Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})

	post.IP = getRealIP(&request)
	post.Timestamp = time.Now()
	post.PosterAuthority = getStaffRank()
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !validReferrer(request) {
		access_log.Print("Rejected post from possible spambot @ " + post.IP)
		//TODO: insert post into temporary post table and add to report list
		return
	}

	switch checkPostForSpam(post.IP, request.Header["User-Agent"][0], request.Referer(),
	post.Name, post.Email, post.MessageText) {
		case "discard":
			access_log.Print("Akismet recommended discarding post from: " + post.IP)
			return
		case "spam":
			access_log.Print("Akismet suggested post is spam from " + post.IP)
			return
		default:
	}

	file, handler, uploaderr := request.FormFile("imagefile")
	if uploaderr != nil {
		// no file was uploaded
		post.Filename = ""
		access_log.Print("Receiving post from " + request.RemoteAddr + ", referred from: " + request.Referer())

	} else {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			server.ServeErrorPage(w, "Couldn't read file: "+err.Error())
		} else {
			post.FilenameOriginal = html.EscapeString(handler.Filename)
			filetype := getFileExtension(post.FilenameOriginal)
			thumb_filetype := filetype
			if thumb_filetype == "gif" {
				thumb_filetype = "jpg"
			}
			post.FilenameOriginal = escapeString(post.FilenameOriginal)
			post.Filename = getNewFilename() + "." + getFileExtension(post.FilenameOriginal)
			board_arr, _ := getBoardArr("`id` = " + request.FormValue("boardid"))
			if len(board_arr) == 0 {
				server.ServeErrorPage(w, "No boards have been created yet")
			}
			_board_dir, _ := getBoardArr("`id` = " + request.FormValue("boardid"))
			board_dir := _board_dir[0].Dir
			file_path := path.Join(config.DocumentRoot, "/"+board_dir+"/src/", post.Filename)
			thumb_path := path.Join(config.DocumentRoot, "/"+board_dir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "t."+thumb_filetype, -1))
			catalog_thumb_path := path.Join(config.DocumentRoot, "/"+board_dir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "c."+thumb_filetype, -1))

			err := ioutil.WriteFile(file_path, data, 0777)
			if err != nil {
				errortext = "Couldn't write file \"" + post.Filename + "\"" + err.Error()
				println(1, errortext)
				error_log.Println(errortext)
				server.ServeErrorPage(w, "Couldn't write file \""+post.FilenameOriginal+"\"")
				return
			}

			// Calculate image checksum
			post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

			// TODO Remove me: debugging checksums
			fmt.Printf("Uploaded image checksum: %x", md5.Sum(data))

			// Attempt to load uploaded file with imaging library
			img, err := imaging.Open(file_path)
			if err != nil {
				errortext = "Couldn't open uploaded file \"" + post.Filename + "\"" + err.Error()
				error_log.Println(errortext)
				println(1, errortext)
				server.ServeErrorPage(w, "Upload filetype not supported")

				return
			} else {
				// Get image filesize
				stat, err := os.Stat(file_path)
				if err != nil {
					error_log.Println(err.Error())
					println(1, err.Error())
					server.ServeErrorPage(w, err.Error())
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

				access_log.Print("Receiving post with image: " + handler.Filename + " from " + request.RemoteAddr + ", referrer: " + request.Referer())

				if request.FormValue("spoiler") == "on" {
					// If spoiler is enabled, symlink thumbnail to spoiler image
					_, err := os.Stat(path.Join(config.DocumentRoot, "spoiler.png"))
					if err != nil {
						server.ServeErrorPage(w, "missing /spoiler.png")
						return
					} else {
						err = syscall.Symlink(path.Join(config.DocumentRoot, "spoiler.png"), thumb_path)
						if err != nil {
							server.ServeErrorPage(w, err.Error())
							return
						}
					}
				} else if config.ThumbWidth >= post.ImageW && config.ThumbHeight >= post.ImageH {
					// If image fits in thumbnail size, symlink thumbnail to original
					post.ThumbW = img.Bounds().Max.X
					post.ThumbH = img.Bounds().Max.Y
					err := syscall.Symlink(file_path, thumb_path)
					if err != nil {
						server.ServeErrorPage(w, err.Error())
						return
					}
				} else {
					var thumbnail image.Image
					var catalog_thumbnail image.Image
					if post.ParentID == 0 {
						// If this is a new thread, generate thumbnail and catalog thumbnail
						thumbnail = createThumbnail(img, "op")
						catalog_thumbnail = createThumbnail(img, "catalog")
						err = imaging.Save(catalog_thumbnail, catalog_thumb_path)
						if err != nil {
							server.ServeErrorPage(w, err.Error())
							return
						}
					} else {
						thumbnail = createThumbnail(img, "reply")
					}
					err = imaging.Save(thumbnail, thumb_path)
					if err != nil {
						println(1, err.Error())
						error_log.Println(err.Error())
						server.ServeErrorPage(w, err.Error())
						return
					}
				}
			}
		}
	}

	if strings.TrimSpace(post.MessageText) == "" && post.Filename == "" {
		server.ServeErrorPage(w, "Post must contain a message if no image is uploaded.")
		return
	}
	post_delay := sinceLastPost(&post)
	if post_delay > -1 {
		if post.ParentID == 0 && post_delay < config.NewThreadDelay {
			server.ServeErrorPage(w, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && post_delay < config.ReplyDelay {
			server.ServeErrorPage(w, "Please wait before making a reply.")
			return
		}
	}

	isbanned, err := checkBannedStatus(&post, &w)
	if err != nil {
		errortext = "Error in checkBannedStatus: " + err.Error()
		server.ServeErrorPage(w, err.Error())
		error_log.Println(errortext)
		println(1, errortext)
		return
	}

	if len(isbanned) > 0 {
		wrapped := &Wrapper{IName: "bans", Data: isbanned}

		var banpage_buffer bytes.Buffer
		var banpage_html string
		banpage_buffer.Write([]byte(""))

		err = banpage_tmpl.Execute(&banpage_buffer, wrapped)
		if err != nil {
			fmt.Fprintf(writer, banpage_html+err.Error()+"\n</body>\n</html>")
			println(1, err.Error())
			error_log.Println(err.Error())
			return
		}
		fmt.Fprintf(w, banpage_buffer.String())
		return
	}

	result, err := insertPost(post, email_command != "sage")
	if err != nil {
		server.ServeErrorPage(w, err.Error())
		return
	}
	postid, _ := result.LastInsertId()
	post.ID = int(postid)

	boards, _ := getBoardArr("")
	// rebuild the board page
	buildBoards(false, post.BoardID)

	buildFrontPage()

	if email_command == "noko" {
		if post.ParentID == 0 {
			http.Redirect(writer, &request, "/"+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, &request, "/"+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, &request, "/"+boards[post.BoardID-1].Dir+"/", http.StatusFound)
	}
	benchmarkTimer("makePost", start_time, false)
}

func formatMessage(post *PostTable) {
	message := post.MessageHTML

	// prepare each line to be formatted
	post_lines := strings.Split(message,"\\r\\n")
	for i, line := range post_lines {
		trimmed_line := strings.TrimSpace(line)
		line_words := strings.Split(trimmed_line," ")
		is_greentext := false // if true, append </span> to end of line
		for w,word := range line_words {
			if strings.LastIndex(word, gt+gt) == 0 {
				//word is a backlink
				_,err := strconv.Atoi(word[8:])
				if err == nil {
					// the link is in fact, a valid int
					var board_dir string
					var link_parent int
					db.QueryRow("SELECT `dir`,`parentid` FROM " + config.DBprefix + "posts," + config.DBprefix + "boards WHERE " + config.DBprefix + "posts.id = '" + word[8:] + "';").Scan(&board_dir,&link_parent)
					// get post board dir

					if board_dir == "" {
						line_words[w] = "<a href=\"javascript:;\"><strike>" + word + "</strike></a>"
					} else if link_parent == 0 {
						line_words[w] = "<a href=\"/" + board_dir + "/res/" + word[8:] + ".html\">" + word + "</a>"
					} else {
						line_words[w] = "<a href=\"/" + board_dir + "/res/" + strconv.Itoa(link_parent) + ".html#" + word[8:] + "\">" + word + "</a>"
					}
				}
			} else if strings.Index(word, gt) == 0 && w == 0 {
				// word is at the beginning of a line, and is greentext
				is_greentext = true
				line_words[w] = "<span class=\"greentext\">" + word
			}
		}
		line = strings.Join(line_words," ")
		if is_greentext {
			line += "</span>"
		}
		post_lines[i] = line
	}
	post.MessageHTML = strings.Join(post_lines,"<br />")
}
