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
		return err
	}

	var recentPostsArr []RecentPost

	for rows.Next() {
		recentPost := new(RecentPost)
		if err = rows.Scan(
			&recentPost.PostID, &recentPost.ParentID, &recentPost.BoardName, &recentPost.BoardID,
			&recentPost.Name, &recentPost.Tripcode, &recentPost.Message, &recentPost.Filename, &recentPost.ThumbW, &recentPost.ThumbH,
		); err != nil {
			return err
		}
		recentPostsArr = append(recentPostsArr, recentPost)
	}

	return recentPostsArr
}
