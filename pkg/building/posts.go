package building

import (
	"fmt"
	"html/template"
	"net"
	"path"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
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
	t.stickied as stickied,
	flag, country
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
	IP               net.IP        `json:"-"`
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
	Country          geoip.Country `json:"-"`
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
	thumbnail, _ := uploads.GetThumbnailFilenames(p.Filename)
	return config.WebPath(p.BoardDir, "thumb", thumbnail)
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

func QueryPosts(query string, params []any, cb func(Post) error) error {
	rows, err := gcsql.QuerySQL(query, params...)
	if err != nil {
		return err
	}
	defer rows.Close()
	dbType := config.GetSystemCriticalConfig().DBtype

	for rows.Next() {
		var post Post
		dest := []any{&post.ID, &post.thread.ID}
		var ip string
		if dbType == "mysql" {
			dest = append(dest, &post.IP)
		} else {
			dest = append(dest, &ip)
		}
		var lastBump time.Time
		dest = append(dest,
			&post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
			&post.LastModified, &post.ParentID, &lastBump, &post.Message, &post.MessageRaw, &post.BoardDir,
			&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
			&post.thread.Locked, &post.thread.Stickied, &post.Country.Flag, &post.Country.Name)

		if err = rows.Scan(dest...); err != nil {
			return err
		}
		if dbType != "mysql" {
			post.IP = net.ParseIP(ip)
			if post.IP == nil {
				return fmt.Errorf("invalid IP address %q", ip)
			}
		}
		post.IsTopPost = post.ParentID == 0 || post.ParentID == post.ID
		if post.Filename != "" {
			post.Extension = path.Ext(post.Filename)
		}
		if err = cb(post); err != nil {
			return err
		}
	}
	return rows.Close()
}

func GetBuildablePost(id int, _ int) (*Post, error) {
	const query = postQueryBase + " AND DBPREFIXposts.id = ?"

	var post Post
	var lastBump time.Time
	var ip string
	out := []any{&post.ID, &post.thread.ID}
	dbType := config.GetSystemCriticalConfig().DBtype
	if dbType == "mysql" {
		out = append(out, &post.IP)
	} else {
		out = append(out, &ip)
	}
	out = append(out, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Timestamp,
		&post.LastModified, &post.ParentID, lastBump, &post.Message, &post.MessageRaw, &post.BoardID, &post.BoardDir,
		&post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
		&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
		&post.thread.Locked, &post.thread.Stickied, &post.Country.Flag, &post.Country.Name)

	err := gcsql.QueryRowSQL(query, []any{id}, out)
	if err != nil {
		return nil, err
	}
	if dbType != "mysql" {
		post.IP = net.ParseIP(ip)
		if post.IP == nil {
			return nil, fmt.Errorf("invalid post IP address %q", ip)
		}
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

	var posts []Post
	err := QueryPosts(query, []any{ip}, func(p Post) error {
		posts = append(posts, p)
		return nil
	})
	return posts, err
}

func getThreadPosts(thread *gcsql.Thread) ([]Post, error) {
	const query = postQueryBase + " AND DBPREFIXposts.thread_id = ? ORDER BY DBPREFIXposts.id ASC"
	var posts []Post
	err := QueryPosts(query, []any{thread.ID}, func(p Post) error {
		posts = append(posts, p)
		return nil
	})
	return posts, err
}

func GetRecentPosts(boardid int, limit int) ([]Post, error) {
	query := postQueryBase
	args := []any{}

	if boardid > 0 {
		query += " WHERE t.board_id = ?"
		args = append(args, boardid)
	}

	query += " ORDER BY DBPREFIXposts.id DESC LIMIT " + strconv.Itoa(limit)

	var posts []Post
	err := QueryPosts(query, args, func(post Post) error {
		if boardid == 0 || post.BoardID == boardid {
			post.Extension = path.Ext(post.Filename)
			posts = append(posts, post)
		}
		return nil
	})
	return posts, err
}
