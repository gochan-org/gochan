package building

import (
	"context"
	"fmt"
	"net"
	"path"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
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

// Post represents a post in a thread for building (hence why ParentID is used instead of ThreadID)
type Post struct {
	gcsql.Post
	ParentID         int           `json:"resto"`
	BoardID          int           `json:"-"`
	BoardDir         string        `json:"-"`
	IP               net.IP        `json:"-"`
	Filename         string        `json:"tim"`
	OriginalFilename string        `json:"filename"`
	Checksum         string        `json:"md5"`
	Extension        string        `json:"extension"`
	Filesize         int           `json:"fsize"`
	UploadWidth      int           `json:"w"`
	UploadHeight     int           `json:"h"`
	ThumbnailWidth   int           `json:"tn_w"`
	ThumbnailHeight  int           `json:"tn_h"`
	LastModified     string        `json:"last_modified"`
	Country          geoip.Country `json:"-"`
	thread           gcsql.Thread
}

// TitleText returns the text to be used for the title of the page
func (p *Post) TitleText() string {
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

func (p *Post) ThreadPath() string {
	threadID := p.ParentID
	if threadID == 0 {
		threadID = p.ID
	}
	return config.WebPath(p.BoardDir, "res", strconv.Itoa(threadID)+".html")
}

// Timestamp returns the time the post was created.
// Deprecated: Use CreatedOn instead.
func (p *Post) Timestamp() time.Time {
	return p.CreatedOn
}

func (p *Post) WebPath() string {
	return p.ThreadPath() + "#" + strconv.Itoa(p.ID)
}

func (p *Post) ThumbnailPath() string {
	if p.Filename == "" {
		return ""
	}
	thumbnail, _ := uploads.GetThumbnailFilenames(p.Filename)
	return config.WebPath(p.BoardDir, "thumb", thumbnail)
}

func (p *Post) UploadPath() string {
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

func (p *Post) Cyclic() bool {
	return p.thread.Cyclic
}

// Select all from v_building_posts (and queries with the same columns) and call the callback function on each Post
// returned
func QueryPosts(query string, params []any, cb func(*Post) error) error {
	sqlCfg := config.GetSQLConfig()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds)*time.Second)
	defer cancel()

	rows, err := gcsql.QueryContextSQL(ctx, nil, query, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		dest := []any{&post.ID, &post.thread.ID}
		var ip string
		if sqlCfg.DBtype == "mysql" {
			dest = append(dest, &post.IP)
		} else {
			dest = append(dest, &ip)
		}
		var lastBump time.Time
		dest = append(dest,
			&post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.CreatedOn,
			&post.LastModified, &post.ParentID, &lastBump, &post.Message, &post.MessageRaw, &post.BoardID,
			&post.BoardDir, &post.OriginalFilename, &post.Filename, &post.Checksum, &post.Filesize,
			&post.ThumbnailWidth, &post.ThumbnailHeight, &post.UploadWidth, &post.UploadHeight,
			&post.thread.Locked, &post.thread.Stickied, &post.thread.Cyclic, &post.Country.Flag, &post.Country.Name,
			&post.IsDeleted)

		if err = rows.Scan(dest...); err != nil {
			return err
		}
		if sqlCfg.DBtype != "mysql" {
			post.IP = net.ParseIP(ip)
			if post.IP == nil {
				return fmt.Errorf("invalid IP address %q", ip)
			}
		}
		post.IsTopPost = post.ParentID == 0 || post.ParentID == post.ID
		if post.Filename != "" {
			post.Extension = path.Ext(post.Filename)
		}
		if err = cb(&post); err != nil {
			return err
		}
	}
	return rows.Close()
}

func GetBuildablePostsByIP(ip string, limit int) ([]*Post, error) {
	query := `SELECT id, thread_id, ip, name, tripcode, email, subject, created_on, last_modified, parent_id,
		last_bump, message, message_raw, board_id, dir, original_filename, filename, checksum, filesize, tw, th,
		width, height, locked, stickied, cyclical, flag, country, is_deleted
		FROM DBPREFIXv_building_posts WHERE ip = PARAM_ATON ORDER BY id DESC`
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
	}

	var posts []*Post
	err := QueryPosts(query, []any{ip}, func(p *Post) error {
		posts = append(posts, p)
		return nil
	})
	return posts, err
}

func getThreadPosts(thread *gcsql.Thread) ([]*Post, error) {
	const query = `SELECT id, thread_id, ip, name, tripcode, email, subject, created_on, last_modified, parent_id,
		last_bump, message, message_raw, board_id, dir, original_filename, filename, checksum, filesize, tw, th,
		width, height, locked, stickied, cyclical, flag, country, is_deleted
		FROM DBPREFIXv_building_posts WHERE thread_id = ? ORDER BY id ASC`
	var posts []*Post
	err := QueryPosts(query, []any{thread.ID}, func(p *Post) error {
		posts = append(posts, p)
		return nil
	})
	return posts, err
}

func GetRecentPosts(boardid int, limit int) ([]*Post, error) {
	query := `SELECT id, thread_id, ip, name, tripcode, email, subject, created_on, last_modified, parent_id,
		last_bump, message, message_raw, board_id, dir, original_filename, filename, checksum, filesize, tw, th,
		width, height, locked, stickied, cyclical, flag, country, is_deleted
		FROM DBPREFIXv_building_posts`
	var args []any

	if boardid > 0 {
		query += " WHERE board_id = ?"
		args = append(args, boardid)
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(limit)

	var posts []*Post
	err := QueryPosts(query, args, func(post *Post) error {
		if boardid == 0 || post.BoardID == boardid {
			post.Extension = path.Ext(post.Filename)
			posts = append(posts, post)
		}
		return nil
	})
	return posts, err
}
