package pre2021

import (
	"time"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

func GetPosts(db *gcsql.GCDB) ([]Post, error) {
	rows, err := db.QuerySQL("SELECT id,boardid,parentid,name,tripcode,email,subject,message,message_raw,password,filename,filename_original,file_checksum,filesize,image_w,image_h,thumb_w,thumb_h,ip,tag,timestamp,autosage,deleted_timestamp,bumped,stickied,locked,reviewed FROM `gc_posts`")
	if err != nil {
		return nil, err
	}
	var posts []Post
	for rows.Next() {
		var post Post
		err = rows.Scan(&post.ID, &post.BoardID, &post.ParentID, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.MessageHTML, &post.MessageText, &post.Password, &post.Filename, &post.FilenameOriginal, &post.FileChecksum, &post.Filesize, &post.ImageW, &post.ImageH, &post.ThumbW, &post.ThumbH, &post.IP, &post.Capcode, &post.Timestamp, &post.Autosage, &post.DeletedTimestamp, &post.Bumped, &post.Stickied, &post.Locked, &post.Reviewed)
		if err != nil {
			return posts, err
		}

		posts = append(posts, post)
	}
	return posts, nil
}

// DBPREFIXannouncements
type Announcement struct {
	ID        uint      // id: bigint
	Subject   string    // subject: varchar
	Message   string    // message: text
	Poster    string    // poster: varchar
	Timestamp time.Time // timestamp: timestamp
}

// DBPREFIXappeals
type BanAppeal struct {
	ID            int        // id: bigint
	Ban           int        // ban: int
	Message       string     // message: text
	Denied        bool       // denied: tinyint
	Timestamp     *time.Time // timestamp: timestamp
	StaffResponse string     // staff_response: text
}

// DBPREFIXposts
type Post struct {
	ID               int       // id: bigint
	BoardID          int       // boardid: int
	ParentID         int       // parentid: int
	Name             string    // name: varchar
	Tripcode         string    // tripcode: varchar
	Email            string    // email: varchar
	Subject          string    // subject: varchar
	MessageHTML      string    // message: text
	MessageText      string    // message_raw: text
	Password         string    // password: varchar
	Filename         string    // filename: varchar
	FilenameOriginal string    // filename_original: varchar
	FileChecksum     string    // file_checksum: varchar
	Filesize         int       // filesize: int
	ImageW           int       // image_w: smallint
	ImageH           int       // image_h: smallint
	ThumbW           int       // thumb_w: smallint
	ThumbH           int       // thumb_h: smallint
	IP               string    // ip: varchar
	Capcode          string    // tag: varchar
	Timestamp        time.Time // timestamp: timestamp
	Autosage         bool      // autosage: tinyint
	DeletedTimestamp time.Time // deleted_timestamp: timestamp
	Bumped           time.Time // bumped: timestamp
	Stickied         bool      // stickied: tinyint
	Locked           bool      // locked: tinyint
	Reviewed         bool      // reviewed: tinyint
}
