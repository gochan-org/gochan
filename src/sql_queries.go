package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	libgeo "github.com/nranchev/go-libGeoIP"
	"golang.org/x/crypto/bcrypt"
)

// GetTopPostsNoSort gets the thread ops for a given board.
// Results are unsorted
func GetTopPostsNoSort(boardID int) {
	//TODO
}

// GetTopPosts gets the thread ops for a given board.
// newestFirst sorts the ops by the newest first if true, by newest last if false
func GetTopPosts(boardID int, newestFirst bool) (posts []Post, err error) {
	//TODO sort by bump
}

// GetExistingReplies gets all the reply posts to a given thread, ordered by oldest first.
func GetExistingReplies(topPost int) (posts []Post, err error) {
	//TODO sort by number/date
}

// GetExistingRepliesLimitedRev gets N amount of reply posts to a given thread, ordered by newest first.
func GetExistingRepliesLimitedRev(topPost int, limit int) (posts []Post, err error) {
	//TODO
}

// GetSpecificTopPost gets the information for the top post for a given id.
func GetSpecificTopPost(ID int) (posts []Post, err error) {
	//Currently implemented as GetSpecificPost because getSpecificPost can also be a top post.
	return GetSpecificPost(ID)
}

// GetSpecificPost gets a specific post for a given id.
func GetSpecificPost(ID int) (posts []Post, err error) {
	//TODO
}

// GetAllNondeletedMessageRaw gets all the raw message texts from the database, saved per id
func GetAllNondeletedMessageRaw() (messages []MessagePostContainer, err error) {
	//TODO
}

// SetMessages sets all the non-raw text for a given array of items.
func SetMessages(messages []MessagePostContainer) (err error) {
	//TODO
}

// GetRecentPosts queries and returns the global N most recent posts from the database.
func GetRecentPosts(amount int, onlyWithFile bool) (recentPosts []RecentPost) {
	//TODO: rework so it uses all features/better sql
	//get recent posts
	recentQueryStr := `
	/*
	recentposts = join all non-deleted posts with the post id of their thread and the board it belongs on, sort by date and grab top x posts
	singlefiles = the top file per post id
	
	Left join singlefiles on recentposts where recentposts.selfid = singlefiles.post_id
	Coalesce filenames to "" (if filename = null -> "" else filename)
	
	Query might benefit from [filter on posts with at least one file -> ] filter N most recent -> manually loop N results for file/board/parentthreadid
	*/
	
	Select 
		recentposts.selfid AS id,
		recentposts.toppostid AS parentid,
		recentposts.boardname,
		recentposts.boardid,
		recentposts.name,
		recentposts.tripcode,
		recentposts.message,
		COALESCE(singlefiles.filename, '') as filename,
		singlefiles.thumbnail_width as thumb_w,
		singlefiles.thumbnail_height as thumb_h
	FROM
		(SELECT 
			posts.id AS selfid,
			topposts.id AS toppostid,
			boards.dir AS boardname,
			boards.id AS boardid,
			posts.name,
			posts.tripcode,
			posts.message,
			posts.email,
			 posts.created_on
		FROM
			DBPREFIXposts AS posts
		JOIN DBPREFIXthreads AS threads 
			ON threads.id = posts.thread_id
		JOIN DBPREFIXposts AS topposts 
			ON threads.id = topposts.thread_id
		JOIN DBPREFIXboards AS boards
			ON threads.board_id = boards.id
		WHERE 
			topposts.is_top_post = TRUE AND posts.is_deleted = FALSE
		
		) as recentposts
	LEFT JOIN 
		(SELECT files.post_id, filename, files.thumbnail_width, files.thumbnail_height
		FROM DBPREFIXfiles as files
		JOIN 
			(SELECT post_id, min(file_order) as file_order
			FROM DBPREFIXfiles
			GROUP BY post_id) as topfiles 
			ON files.post_id = topfiles.post_id AND files.file_order = topfiles.file_order
		) AS singlefiles 
		
		ON recentposts.selfid = singlefiles.post_id`
	if onlyWithFile {
		recentQueryStr += "WHERE singlefiles.filename IS NOT NULL"
	}
	recentQueryStr += "ORDER BY recentposts.created_on DESC LIMIT ?"

	rows, err := querySQL(recentQueryStr, amount)
	defer closeHandle(rows)
	if err != nil {
		return nil, err
	}

	var recentPostsArr []RecentPost

	for rows.Next() {
		recentPost := new(RecentPost)
		if err = rows.Scan(
			&recentPost.PostID, &recentPost.ParentID, &recentPost.BoardName, &recentPost.BoardID,
			&recentPost.Name, &recentPost.Tripcode, &recentPost.Message, &recentPost.Filename, &recentPost.ThumbW, &recentPost.ThumbH,
		); err != nil {
			return nil, err
		}
		recentPostsArr = append(recentPostsArr, recentPost)
	}

	return recentPostsArr, nil
}

// GetReplyCount gets the total amount non-deleted of replies in a thread
func GetReplyCount(postID int) (replyCount int, err error) {

}

// GetReplyFileCount gets the amount of files non-deleted posted in total in a thread
func GetReplyFileCount(postID int) (fileCount int, err error) {

}

// GetStaffData returns the data associated with a given username
func GetStaffData(staffName string) (data string, err error) {
	//("SELECT sessiondata FROM DBPREFIXsessions WHERE name = ?",
}

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (name string, err error) {
	//after refactor, check if still used
}

func GetStaffBySession(session string) (*Staff, error) { //TODO not upt to date with old db yet
	staff := new(Staff)
	err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
		[]interface{}{name},
		[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	)
	return staff, err
}

func GetStaffByName(name string) (*Staff, error) { //TODO not upt to date with old db yet
	staff := new(Staff)
	err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
		[]interface{}{name},
		[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	)
	return staff, err
}

func newStaff(username string, password string, rank int) error { //TODO not up to date with old db yet
	_, err := execSQL("INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES(?,?,?)",
		&username, bcryptSum(password), &rank)
	return err
}

func deleteStaff(username string) error { //TODO not up to date with old db yet
	_, err := execSQL("DELETE FROM DBPREFIXstaff WHERE username = ?", username)
	return err
}

func CreateSession(key string, username string) error { //TODO not up to date with old db yet
	//TODO move amount of time to config file
	//TODO also set last login
	return execSQL("INSERT INTO DBPREFIXsessions (name,sessiondata,expires) VALUES(?,?,?)",
		key, username, getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*730))),
	)
}

func PermanentlyRemoveDeletedPosts() error {
	//Remove all deleted posts
	//Remove orphaned threads
	//Make sure cascades are set up properly
}

func OptimizeDatabase() error { //TODO FIX, try to do it entirely within one SQL transaction

	html += "Optimizing all tables in database.<hr />"
	tableRows, tablesErr := querySQL("SHOW TABLES")
	defer closeHandle(tableRows)

	if tablesErr != nil && tablesErr != sql.ErrNoRows {
		return html + "<tr><td>" +
			gclog.Print(lErrorLog, "Error optimizing SQL tables: ", tablesErr.Error()) +
			"</td></tr></table>"
	}
	for tableRows.Next() {
		var table string
		tableRows.Scan(&table)
		if _, err := execSQL("OPTIMIZE TABLE " + table); err != nil {
			return html + "<tr><td>" +
				gclog.Print(lErrorLog, "Error optimizing SQL tables: ", tablesErr.Error()) +
				"</td></tr></table>"
		}
	}
}

// FileBan creates a new ban on a file. If boards = nil, the ban is global.
func FileBan(fileChecksum string, staffName string, expires Time, permaban bool, staffNote string, boardURI string) error {

}

// FileNameBan creates a new ban on a filename. If boards = nil, the ban is global.
func FileNameBan(fileName string, isRegex bool, staffName string, expires Time, permaban bool, staffNote string, boardURI string) error {

}

// UserNameBan creates a new ban on a username. If boards = nil, the ban is global.
func UserNameBan(userName string, isRegex bool, staffName string, expires Time, permaban bool, staffNote string, boardURI string) error {

}

func UserBan(threadBan bool, staffName string, boardURI string, postID int, expires Time, permaban bool,
	staffNote string, message string, canAppeal bool, appealAt Time) error {

}

func GetStaffRankAndBoards(username string) (rank int, boardUris []string, err error) {

}

//GetAllAccouncements gets all announcements, newest first
func GetAllAccouncements() ([]Announcement, error) {
	//("SELECT subject,message,poster,timestamp FROM DBPREFIXannouncements ORDER BY id DESC")
	//rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)
}
