package building

import (
	"encoding/json"
	"errors"
	"html/template"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

const (
	postQueryBase = `SELECT DBPREFIXposts.id AS postid, thread_id AS threadid, ip, name, tripcode, email, subject, created_on, created_on as last_modified,
	(SELECT id FROM DBPREFIXposts WHERE thread_id = threadid AND is_top_post) AS parent_id,
	message, message_raw,
	(SELECT board_id FROM DBPREFIXthreads where id = threadid LIMIT 1) AS boardid,
	(SELECT dir FROM DBPREFIXboards WHERE id = boardid LIMIT 1) AS dir,
	coalesce(DBPREFIXfiles.original_filename,'') as original_filename,
	coalesce(DBPREFIXfiles.filename,'') AS filename,
	coalesce(DBPREFIXfiles.checksum,'') AS checksum,
	coalesce(DBPREFIXfiles.file_size,0) AS filesize,
	coalesce(DBPREFIXfiles.thumbnail_width,0) AS tw,
	coalesce(DBPREFIXfiles.thumbnail_height,0) AS th,
	coalesce(DBPREFIXfiles.width,0) AS width,
	coalesce(DBPREFIXfiles.height,0) AS height
	FROM DBPREFIXposts
	LEFT JOIN DBPREFIXfiles ON DBPREFIXfiles.post_id = DBPREFIXposts.id WHERE is_deleted = FALSE`
)

func truncateString(msg string, limit int, ellipsis bool) string {
	if len(msg) > limit {
		if ellipsis {
			return msg[:limit] + "..."
		}
		return msg[:limit]
	}
	return msg
}

type Post struct {
	ID               int           `json:"no"`
	ParentID         int           `json:"resto"`
	IsTopPost        bool          `json:"-"`
	BoardID          int           `json:"-"`
	BoardDir         string        `json:"-"`
	IP               string        `json:"-"`
	Name             string        `json:"name"`
	Tripcode         string        `json:"trip"`
	Email            string        `json:"email"`
	Subject          string        `json:"sub"`
	MessageRaw       string        `json:"com"`
	Message          template.HTML `json:"-"`
	Filename         string        `json:"tim"`
	OriginalFilename string        `json:"filename"`
	Checksum         string        `json:"md5"`
	Extension        string        `json:"extension"`
	Filesize         int           `json:"fsize"`
	UploadWidth      int           `json:"w"`
	UploadHeight     int           `json:"h"`
	ThumbnailWidth   int           `json:"tn_w"`
	ThumbnailHeight  int           `json:"tn_h"`
	Capcode          string        `json:"capcode"`
	Timestamp        time.Time     `json:"time"`
	LastModified     string        `json:"last_modified"`
}

func (p Post) TitleText() string {
	title := "/" + p.BoardDir + "/ - "
	if p.Subject != "" {
		title += truncateString(p.Subject, 20, true)
	} else if p.Message != "" {
		title += truncateString(bbcodeTagRE.ReplaceAllString(p.MessageRaw, ""), 20, true)
	} else {
		title += "#" + strconv.Itoa(p.ID)
	}
	return title
}

func (p Post) WebPath() string {
	threadID := p.ParentID
	if threadID == 0 {
		threadID = p.ID
	}
	return config.WebPath(p.BoardDir, "res", strconv.Itoa(threadID)+".html#"+strconv.Itoa(p.ID))
}

func (p Post) ThumbnailPath() string {
	if p.Filename == "" {
		return ""
	}
	return config.WebPath(p.BoardDir, "thumb", gcutil.GetThumbnailPath("reply", p.Filename))
}

func (p Post) UploadPath() string {
	if p.Filename == "" {
		return ""
	}
	return config.WebPath(p.BoardDir, "src", p.Filename)

}

func GetBuildablePost(id int, boardid int) (*Post, error) {
	const query = postQueryBase + " AND DBPREFIXposts.id = ?"
	var post Post
	var threadID int
	err := gcsql.QueryRowSQL(query, []interface{}{id}, []interface{}{
		&post.ID, &threadID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
		&post.LastModified, &post.ParentID, &post.Message, &post.MessageRaw, &post.BoardID, &post.BoardDir,
		&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
		&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
	})
	if err != nil {
		return nil, err
	}
	post.IsTopPost = post.ParentID == 0
	post.Extension = path.Ext(post.Filename)
	return &post, nil
}

func getBoardTopPosts(boardID int) ([]Post, error) {
	const query = "SELECT * FROM (" + postQueryBase + " AND is_top_post) p WHERE boardid = ?"
	rows, err := gcsql.QuerySQL(query, boardID)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		var threadID int
		err = rows.Scan(
			&post.ID, &threadID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &post.Message, &post.MessageRaw, &post.BoardID, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
		)
		if err != nil {
			return nil, err
		}
		post.IsTopPost = post.ParentID == 0 || post.ParentID == post.ID
		posts = append(posts, post)
	}
	return posts, nil
}

func getThreadPosts(thread *gcsql.Thread) ([]Post, error) {
	const query = postQueryBase + " AND DBPREFIXposts.thread_id = ?"
	rows, err := gcsql.QuerySQL(query, thread.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		var threadID int
		err = rows.Scan(
			&post.ID, &threadID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &post.Message, &post.MessageRaw, &post.BoardID, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
		)
		if err != nil {
			return nil, err
		}
		post.IsTopPost = post.ParentID == 0 || post.ParentID == post.ID
		posts = append(posts, post)
	}
	return posts, nil
}

func GetRecentPosts(boardid int, limit int) ([]Post, error) {
	query := "SELECT * FROM (" + postQueryBase + ") posts"
	var args []interface{} = []interface{}{}

	if boardid > 0 {
		query += " WHERE boardid = ?"
		args = append(args, boardid)
	}

	rows, err := gcsql.QuerySQL(query+" ORDER BY postid DESC LIMIT "+strconv.Itoa(limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		var threadID int
		err = rows.Scan(
			&post.ID, &threadID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &post.Message, &post.MessageRaw, &post.BoardID, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
		)
		if err != nil {
			return nil, err
		}
		if boardid == 0 || post.BoardID == boardid {
			post.Extension = path.Ext(post.Filename)
			posts = append(posts, post)
		}
	}
	return posts, nil
}

// BuildBoardListJSON generates a JSON file with info about the boards
func BuildBoardListJSON() error {
	boardListFile, err := os.OpenFile(config.WebPath("boards.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		gcutil.LogError(err).
			Str("building", "boardsList").Send()
		return errors.New("Failed opening boards.json for writing: " + err.Error())
	}
	defer boardListFile.Close()

	boardsMap := map[string][]gcsql.Board{
		"boards": {},
	}

	// TODO: properly check if the board is in a hidden section
	boardsMap["boards"] = gcsql.AllBoards
	boardJSON, err := json.Marshal(boardsMap)
	if err != nil {
		gcutil.LogError(err).Str("building", "boards.json").Send()
		return errors.New("Failed to create boards.json: " + err.Error())
	}

	if _, err = serverutil.MinifyWriter(boardListFile, boardJSON, "application/json"); err != nil {
		gcutil.LogError(err).Str("building", "boards.json").Send()
		return errors.New("Failed writing boards.json file: " + err.Error())
	}
	return nil
}