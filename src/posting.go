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
	lastPost    PostTable
	allSections []interface{}
	allBoards   []interface{}
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
	start_time := benchmarkTimer("buildBoard"+strconv.Itoa(board.ID), time.Now(), true)
	var current_page_file *os.File
	var threads []interface{}
	var thread_pages [][]interface{}
	var stickied_threads []interface{}
	var nonstickied_threads []interface{}

	defer func() {
		// Recover and print, log error (if there is one)
		/* if errmsg, panicked := recover().(error); panicked {
			handleError(0, "Recovered from panic: "+errmsg.Error())
		} */
	}()

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
		thread.IName = "thread"

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
		thread_pages = paginate(config.ThreadsPerPage_img, threads)
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

// buildThreads builds thread(s) given a boardid, or if all = false, also given a threadid.
func buildThreads(all bool, boardid, threadid int) (html string) {
	// TODO: detect which page will be built and only build that one and the board page
	// if all is set to true, ignore which, otherwise, which = build only specified boardid
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

	//thread_pages := paginate(config.PostsPerThreadPage, replies)
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
		html += handleError(1, "Failed building /%s/res/%d threadpage: ", board.Dir, op.ID, err.Error()) + "<br />\n"
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

	success_text := fmt.Sprintf("Built /%s/%d successfully", board.Dir, op.ID)
	html += success_text + "<br />\n"
	println(2, success_text)

	for page_num, page_posts := range thread_pages {
		op.CurrentPage = page_num + 1
		current_page_filepath := path.Join(config.DocumentRoot, board.Dir, "res", strconv.Itoa(op.ID)+"p"+strconv.Itoa(op.CurrentPage)+".html")
		current_page_file, err = os.OpenFile(current_page_filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			html += handleError(1, "Failed opening "+current_page_filepath+": "+err.Error()) + "<br />\n"
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
			html += handleError(1, "Failed building /%s/%d: %s", board.Dir, op.ID, err.Error())
			return
		}

		success_text := fmt.Sprintf("Built /%s/%dp%d successfully", board.Dir, op.ID, op.CurrentPage)
		html += success_text + "<br />\n"
		println(2, success_text)
	}
	return
}

func buildFrontPage() (html string) {
	initTemplates()
	var front_arr []interface{}
	var recent_posts_arr []interface{}

	os.Remove(path.Join(config.DocumentRoot, "index.html"))
	front_file, err := os.OpenFile(path.Join(config.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	defer closeFile(front_file)
	if err != nil {
		return handleError(1, "Failed opening front page for writing: "+err.Error()) + "<br />\n"
	}

	// get front pages
	rows, err := querySQL("SELECT * FROM `" + config.DBprefix + "frontpage`")
	defer closeRows(rows)
	if err != nil {
		return handleError(1, "Failed getting front page rows: "+err.Error())
	}

	for rows.Next() {
		frontpage := new(FrontTable)
		frontpage.IName = "front page"
		if err = rows.Scan(&frontpage.ID, &frontpage.Page, &frontpage.Order, &frontpage.Subject,
			&frontpage.Message, &frontpage.Timestamp, &frontpage.Poster, &frontpage.Email); err != nil {
			return handleError(1, err.Error())
		}
		front_arr = append(front_arr, frontpage)
	}

	// get recent posts
	rows, err = querySQL(
		"SELECT `"+config.DBprefix+"posts`.`id`, "+
			"`"+config.DBprefix+"posts`.`parentid`, "+
			"`"+config.DBprefix+"boards`.`dir` AS boardname, "+
			"`"+config.DBprefix+"posts`.`boardid` AS boardid, "+
			"`name`, `tripcode`, `message`, `filename`, `thumb_w`, `thumb_h` "+
			"FROM `"+config.DBprefix+"posts`, `"+config.DBprefix+"boards` "+
			"WHERE `"+config.DBprefix+"posts`.`deleted_timestamp` = ? "+
			"AND `boardid` = `"+config.DBprefix+"boards`.`id` "+
			"ORDER BY `timestamp` DESC LIMIT ?",
		nilTimestamp, config.MaxRecentPosts,
	)
	defer closeRows(rows)
	if err != nil {
		return handleError(1, err.Error())
	}

	for rows.Next() {
		recent_post := new(RecentPost)
		err = rows.Scan(&recent_post.PostID, &recent_post.ParentID, &recent_post.BoardName, &recent_post.BoardID, &recent_post.Name, &recent_post.Tripcode, &recent_post.Message, &recent_post.Filename, &recent_post.ThumbW, &recent_post.ThumbH)
		if err != nil {
			return handleError(1, "Failed getting list of recent posts for front page: "+err.Error())
		}
		recent_posts_arr = append(recent_posts_arr, recent_post)
	}

	if err = front_page_tmpl.Execute(front_file, map[string]interface{}{
		"config":       config,
		"fronts":       front_arr,
		"boards":       allBoards,
		"sections":     allSections,
		"recent_posts": recent_posts_arr,
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
		return handleError(1, "Failed marshal to JSON: "+err.Error()) + "<br />\n"
	}
	if _, err = board_list_file.Write(boardJSON); err != nil {
		return handleError(1, "Failed writing boards.json file: "+err.Error()) + "<br />\n"
	}
	return "Board list JSON rebuilt successfully.<br />"
}

// bumps the given thread on the given board and returns true if there were no errors
func bumpThread(postID, boardID int) error {
	_, err := execSQL("UPDATE `"+config.DBprefix+"posts` SET `bumped` = ? WHERE `id` = ? AND `boardid` = ?",
		time.Now(), postID, boardID,
	)

	return err
}

// Checks check poster's name/tripcode/file checksum (from PostTable post) for banned status
// returns true if the user is banned
func checkBannedStatus(post *PostTable, writer http.ResponseWriter) ([]interface{}, error) {
	var isExpired bool
	var ban_entry BanlistTable
	var interfaces []interface{}
	// var count int
	// var search string
	err := queryRowSQL("SELECT `ip`, `name`, `tripcode`, `message`, `boards`, `timestamp`, `expires`, `appeal_at` FROM `"+config.DBprefix+"banlist` WHERE `ip` = ?",
		[]interface{}{&post.IP},
		[]interface{}{&ban_entry.IP, &ban_entry.Name, &ban_entry.Tripcode, &ban_entry.Message, &ban_entry.Boards, &ban_entry.Timestamp, &ban_entry.Expires, &ban_entry.AppealAt},
	)
	if err == sql.ErrNoRows {
		// the user isn't banned
		// We don't need to return err because it isn't necessary
		return interfaces, nil
	} else if err != nil {
		handleError(1, "Error checking banned status: "+err.Error())
		return interfaces, err
	}
	isExpired = ban_entry.Expires.After(time.Now()) == false
	if isExpired {
		// if it is expired, send a message saying that it's expired, but still post
		println(1, "expired")
		return interfaces, nil
	}
	// the user's IP is in the banlist. Check if the ban has expired
	if getSpecificSQLDateTime(ban_entry.Expires) == "0001-01-01 00:00:00" || ban_entry.Expires.After(time.Now()) {
		// for some funky reason, Go's MySQL driver seems to not like getting a supposedly nil timestamp as an ACTUAL nil timestamp
		// so we're just going to wing it and cheat. Of course if they change that, we're kind of hosed.

		return []interface{}{config, ban_entry}, nil
	}
	return interfaces, nil
}

func sinceLastPost(post *PostTable) int {
	var lastPostTime time.Time
	if err := queryRowSQL("SELECT `timestamp` FROM `"+config.DBprefix+"posts` WHERE `ip` = '?' ORDER BY `timestamp` DESC LIMIT 1",
		[]interface{}{post.IP},
		[]interface{}{&lastPostTime},
	); err == sql.ErrNoRows {
		// no posts by that IP.
		return -1
	}
	return int(time.Since(lastPostTime).Seconds())
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
	vidInfo := make(map[string]int)

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

func parseName(name string) map[string]string {
	parsed := make(map[string]string)
	if !strings.Contains(name, "#") {
		parsed["name"] = name
		parsed["tripcode"] = ""
	} else if strings.Index(name, "#") == 0 {
		parsed["tripcode"] = tripcode.Tripcode(name[1:])
	} else if strings.Index(name, "#") > 0 {
		postNameArr := strings.SplitN(name, "#", 2)
		parsed["name"] = postNameArr[0]
		parsed["tripcode"] = tripcode.Tripcode(postNameArr[1])
	}
	return parsed
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

	result, err := execSQL(insertString+insertValues,
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
func makePost(writer http.ResponseWriter, request *http.Request) {
	startTime := benchmarkTimer("makePost", time.Now(), true)
	var maxMessageLength int
	var post PostTable
	domain := request.Host
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
	parsedName := parseName(formName)
	post.Name = parsedName["name"]
	post.Tripcode = parsedName["tripcode"]

	nameCookie = post.Name + post.Tripcode
	formEmail = request.FormValue("postemail")
	http.SetCookie(writer, &http.Cookie{Name: "email", Value: formEmail, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})

	if !strings.Contains(formEmail, "noko") && !strings.Contains(formEmail, "sage") {
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

	if err := queryRowSQL("SELECT `max_message_length` from `"+config.DBprefix+"boards` WHERE `id` = ?",
		[]interface{}{post.BoardID},
		[]interface{}{&maxMessageLength},
	); err != nil {
		serveErrorPage(writer, handleError(0, "Error getting board info: "+err.Error()))
		return
	}

	if len(post.MessageText) > maxMessageLength {
		serveErrorPage(writer, "Post body is too long")
		return
	}
	post.MessageHTML = formatMessage(post.MessageText)
	post.Password = md5Sum(request.FormValue("postpassword"))

	// Reverse escapes
	nameCookie = strings.Replace(formName, "&amp;", "&", -1)
	nameCookie = strings.Replace(nameCookie, "\\&#39;", "'", -1)
	nameCookie = strings.Replace(url.QueryEscape(nameCookie), "+", "%20", -1)

	// add name and email cookies that will expire in a year (31536000 seconds)
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: nameCookie, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})
	http.SetCookie(writer, &http.Cookie{Name: "password", Value: request.FormValue("postpassword"), Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})

	post.IP = getRealIP(request)
	post.Timestamp = time.Now()
	post.PosterAuthority = getStaffRank(request)
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
		serveErrorPage(writer, "Your post looks like spam.")
		accessLog.Print("Akismet recommended discarding post from: " + post.IP)
		return
	case "spam":
		serveErrorPage(writer, "Your post looks like spam.")
		accessLog.Print("Akismet suggested post is spam from " + post.IP)
		return
	default:
	}

	file, handler, err := request.FormFile("imagefile")
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	if err != nil || handler.Size == 0 {
		// no file was uploaded
		post.Filename = ""
		accessLog.Print("Receiving post from " + request.RemoteAddr + ", referred from: " + request.Referer())
	} else {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			serveErrorPage(writer, handleError(1, "Couldn't read file: "+err.Error()))
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
				serveErrorPage(writer, "No boards have been created yet")
				return
			}
			_boardDir, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
			boardDir := _boardDir[0].Dir
			filePath := path.Join(config.DocumentRoot, "/"+boardDir+"/src/", post.Filename)
			thumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "t."+thumbFiletype, -1))
			catalogThumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "c."+thumbFiletype, -1))

			if err := ioutil.WriteFile(filePath, data, 0777); err != nil {
				handleError(0, "Couldn't write file \""+post.Filename+"\""+err.Error())
				serveErrorPage(writer, "Couldn't write file \""+post.FilenameOriginal+"\"")
				return
			}

			// Calculate image checksum
			post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

			var allowsVids bool
			if err = queryRowSQL("SELECT `embeds_allowed` FROM `"+config.DBprefix+"boards` WHERE `id` = ? LIMIT 1",
				[]interface{}{post.BoardID},
				[]interface{}{&allowsVids},
			); err != nil {
				serveErrorPage(writer, handleError(1, "Couldn't get board info: "+err.Error()))
				return
			}

			if filetype == "webm" {
				if !allowsVids || !config.AllowVideoUploads {
					serveErrorPage(writer, "Video uploading is not currently enabled for this board.")
					os.Remove(filePath)
					return
				}

				accessLog.Print("Receiving post with video: " + handler.Filename + " from " + request.RemoteAddr + ", referrer: " + request.Referer())
				if post.ParentID == 0 {
					err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth)
					if err != nil {
						serveErrorPage(writer, handleError(1, err.Error()))
						return
					}
				} else {
					err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth_reply)
					if err != nil {
						serveErrorPage(writer, handleError(1, err.Error()))
						return
					}
				}

				if err := createVideoThumbnail(filePath, catalogThumbPath, config.ThumbWidth_catalog); err != nil {
					serveErrorPage(writer, handleError(1, err.Error()))
					return
				}

				outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
				if err != nil {
					serveErrorPage(writer, handleError(1, "Error getting video info: "+err.Error()))
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
					os.Remove(filePath)
					handleError(1, "Couldn't open uploaded file \""+post.Filename+"\""+err.Error())
					serveErrorPage(writer, "Upload filetype not supported")
					return
				} else {
					// Get image filesize
					stat, err := os.Stat(filePath)
					if err != nil {
						serveErrorPage(writer, handleError(1, "Couldn't get image filesize: "+err.Error()))
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
						if _, err := os.Stat(path.Join(config.DocumentRoot, "spoiler.png")); err != nil {
							serveErrorPage(writer, "missing /spoiler.png")
							return
						} else {
							err = syscall.Symlink(path.Join(config.DocumentRoot, "spoiler.png"), thumbPath)
							if err != nil {
								serveErrorPage(writer, err.Error())
								return
							}
						}
					} else if config.ThumbWidth >= post.ImageW && config.ThumbHeight >= post.ImageH {
						// If image fits in thumbnail size, symlink thumbnail to original
						post.ThumbW = img.Bounds().Max.X
						post.ThumbH = img.Bounds().Max.Y
						if err := syscall.Symlink(filePath, thumbPath); err != nil {
							serveErrorPage(writer, err.Error())
							return
						}
					} else {
						var thumbnail image.Image
						var catalogThumbnail image.Image
						if post.ParentID == 0 {
							// If this is a new thread, generate thumbnail and catalog thumbnail
							thumbnail = createImageThumbnail(img, "op")
							catalogThumbnail = createImageThumbnail(img, "catalog")
							if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
								serveErrorPage(writer, handleError(1, "Couldn't generate catalog thumbnail: "+err.Error()))
								return
							}
						} else {
							thumbnail = createImageThumbnail(img, "reply")
						}
						if err = imaging.Save(thumbnail, thumbPath); err != nil {
							serveErrorPage(writer, handleError(1, "Couldn't save thumbnail: "+err.Error()))
							return
						}
					}
				}
			}
		}
	}

	if strings.TrimSpace(post.MessageText) == "" && post.Filename == "" {
		serveErrorPage(writer, "Post must contain a message if no image is uploaded.")
		return
	}

	postDelay := sinceLastPost(&post)
	if postDelay > -1 {
		if post.ParentID == 0 && postDelay < config.NewThreadDelay {
			serveErrorPage(writer, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && postDelay < config.ReplyDelay {
			serveErrorPage(writer, "Please wait before making a reply.")
			return
		}
	}

	isBanned, err := checkBannedStatus(&post, writer)
	if err != nil {
		handleError(1, "Error in checkBannedStatus: "+err.Error())
		serveErrorPage(writer, err.Error())
		return
	}

	if len(isBanned) > 0 {
		var banpage_buffer bytes.Buffer
		var banpage_html string
		banpage_buffer.Write([]byte(""))
		if err = banpage_tmpl.Execute(&banpage_buffer, map[string]interface{}{
			"bans": isBanned,
		}); err != nil {
			fmt.Fprintf(writer, banpage_html+handleError(1, err.Error())+"\n</body>\n</html>")
			return
		}
		fmt.Fprintf(writer, banpage_buffer.String())
		return
	}

	sanitizePost(&post)
	result, err := insertPost(post, emailCommand != "sage")
	if err != nil {
		serveErrorPage(writer, handleError(1, err.Error()))
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
			http.Redirect(writer, request, "/"+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, "/"+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, "/"+boards[post.BoardID-1].Dir+"/", http.StatusFound)
	}
	benchmarkTimer("makePost", startTime, false)
}

func formatMessage(message string) string {
	message = bbcompiler.Compile(message)
	// prepare each line to be formatted
	postLines := strings.Split(message, "<br>")
	for i, line := range postLines {
		trimmedLine := strings.TrimSpace(line)
		//lineWords := regexp.MustCompile("\\s").Split(trimmedLine, -1)
		lineWords := strings.Split(trimmedLine, " ")
		isGreentext := false // if true, append </span> to end of line
		for w, word := range lineWords {
			if strings.LastIndex(word, gt+gt) == 0 {
				//word is a backlink
				if _, err := strconv.Atoi(word[8:]); err == nil {
					// the link is in fact, a valid int
					var boardDir string
					var linkParent int

					if err = queryRowSQL("SELECT `dir`,`parentid` FROM "+config.DBprefix+"posts,"+config.DBprefix+"boards WHERE "+config.DBprefix+"posts.id = ?",
						[]interface{}{word[8:]},
						[]interface{}{&boardDir, &linkParent},
					); err != nil {
						handleError(1, customError(err))
					}

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
	return strings.Join(postLines, "<br />")
}
