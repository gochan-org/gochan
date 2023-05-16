package building

import (
	"html/template"
	"path"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	postQueryBase = `SELECT DBPREFIXposts.id, DBPREFIXposts.thread_id, ip, name, tripcode, email, subject, created_on, created_on as last_modified,
	p.id AS parent_id, t.last_bump as last_bump,
	message, message_raw,
	(SELECT dir FROM DBPREFIXboards WHERE id = t.board_id LIMIT 1) AS dir,
	coalesce(DBPREFIXfiles.original_filename,'') as original_filename,
	coalesce(DBPREFIXfiles.filename,'') AS filename,
	coalesce(DBPREFIXfiles.checksum,'') AS checksum,
	coalesce(DBPREFIXfiles.file_size,0) AS filesize,
	coalesce(DBPREFIXfiles.thumbnail_width,0) AS tw,
	coalesce(DBPREFIXfiles.thumbnail_height,0) AS th,
	coalesce(DBPREFIXfiles.width,0) AS width,
	coalesce(DBPREFIXfiles.height,0) AS height,
	t.locked as locked,
	t.stickied as stickied
	FROM DBPREFIXposts
	LEFT JOIN DBPREFIXfiles ON DBPREFIXfiles.post_id = DBPREFIXposts.id AND is_deleted = FALSE
	LEFT JOIN (
		SELECT id, board_id, last_bump, locked, stickied FROM DBPREFIXthreads
	) t ON t.id = DBPREFIXposts.thread_id
	INNER JOIN (
		SELECT id, thread_id FROM DBPREFIXposts WHERE is_top_post
	) p on p.thread_id = DBPREFIXposts.thread_id
	WHERE is_deleted = FALSE `
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
	thread           gcsql.Thread
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

func (p Post) ThreadPath() string {
	threadID := p.ParentID
	if threadID == 0 {
		threadID = p.ID
	}
	return config.WebPath(p.BoardDir, "res", strconv.Itoa(threadID)+".html")
}

func (p Post) WebPath() string {
	return p.ThreadPath() + "#" + strconv.Itoa(p.ID)
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

func (p *Post) Locked() bool {
	return p.thread.Locked
}

func (p *Post) Stickied() bool {
	return p.thread.Stickied
}

func GetBuildablePost(id int, boardid int) (*Post, error) {
	const query = postQueryBase + " AND DBPREFIXposts.id = ?"
	var post Post
	var lastBump time.Time
	err := gcsql.QueryRowSQL(query, []interface{}{id}, []interface{}{
		&post.ID, &post.thread.ID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
		&post.LastModified, &post.ParentID, lastBump, &post.Message, &post.MessageRaw, &post.BoardID, &post.BoardDir,
		&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
		&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
		&post.thread.Locked, &post.thread.Stickied,
	})
	if err != nil {
		return nil, err
	}
	post.IsTopPost = post.ParentID == 0
	post.Extension = path.Ext(post.Filename)
	return &post, nil
}

func GetBuildablePostsByIP(ip string, limit int) ([]Post, error) {
	query := postQueryBase + " AND DBPREFIXposts.ip = ? ORDER BY DBPREFIXposts.id DESC"
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
	}
	rows, err := gcsql.QuerySQL(query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	var lastBump time.Time
	for rows.Next() {
		var post Post
		if err = rows.Scan(
			&post.ID, &post.thread.ID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &lastBump, &post.Message, &post.MessageRaw, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
			&post.thread.Locked, &post.thread.Stickied,
		); err != nil {
			return nil, err
		}
		post.IsTopPost = post.ParentID == 0
		post.Extension = path.Ext(post.Filename)
		posts = append(posts, post)
	}
	return posts, nil
}

func getThreadPosts(thread *gcsql.Thread) ([]Post, error) {
	const query = postQueryBase + " AND DBPREFIXposts.thread_id = ? ORDER BY DBPREFIXposts.id ASC"
	rows, err := gcsql.QuerySQL(query, thread.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	var lastBump time.Time
	for rows.Next() {
		var post Post
		err = rows.Scan(
			&post.ID, &post.thread.ID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &lastBump, &post.Message, &post.MessageRaw, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
			&post.thread.Locked, &post.thread.Stickied,
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
	query := postQueryBase
	var args []interface{} = []interface{}{}

	if boardid > 0 {
		query += " WHERE t.board_id = ?"
		args = append(args, boardid)
	}

	query += " ORDER BY DBPREFIXposts.id DESC LIMIT " + strconv.Itoa(limit)
	rows, err := gcsql.QuerySQL(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	var lastBump time.Time
	for rows.Next() {
		var post Post
		var threadID int
		err = rows.Scan(
			&post.ID, &threadID, &post.IP, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &lastBump, &post.Message, &post.MessageRaw, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
			&post.thread.Locked, &post.thread.Stickied,
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
