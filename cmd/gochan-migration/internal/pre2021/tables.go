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

// DBPREFIXbanlist
type BanInfo struct {
	ID          uint      // id: bigint
	AllowRead   bool      // allow_read: tinyint
	IP          string    // ip: varchar
	Name        string    // name: varchar
	NameIsRegex bool      // name_is_regex: tinyint
	Filename    string    // filename: varchar
	Checksum    string    // file_checksum: varchar
	Boards      string    // boards: varchar
	Staff       string    // staff: varchar
	Timestamp   time.Time // timestamp: timestamp
	Expires     time.Time // expires: timestamp
	Permaban    bool      // permaban: tinyint
	Reason      string    // reason: varchar
	Type        int       // type: smallint
	StaffNote   string    // staff_note: varchar
	AppealAt    time.Time // appeal_at: timestamp
	CanAppeal   bool      // can_appeal: tinyint
}

// DBPREFIXboards
type Board struct {
	ID               int       // id: bigint
	ListOrder        int       // list_order: tinyint
	Dir              string    // dir: varchar
	Type             int       // type: tinyint
	UploadType       int       // upload_type: tinyint
	Title            string    // title: varchar
	Subtitle         string    // subtitle: varchar
	Description      string    // description: varchar
	Section          int       // section: int
	MaxFilesize      int       // max_file_size: int
	MaxPages         int       // max_pages: tinyint
	DefaultStyle     string    // default_style: varchar
	Locked           bool      // locked: tinyint
	CreatedOn        time.Time // created_on: timestamp
	Anonymous        string    // anonymous: varchar
	ForcedAnon       bool      // forced_anon: tinyint
	MaxAge           int       // max_age: int
	AutosageAfter    int       // autosage_after: int
	NoImagesAfter    int       // no_images_after: int
	MaxMessageLength int       // max_message_length: int
	EmbedsAllowed    bool      // embeds_allowed: tinyint
	RedirectToThread bool      // redirect_to_thread: tinyint
	RequireFile      bool      // require_file: tinyint
	EnableCatalog    bool      // enable_catalog: tinyint
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

// DBPREFIXreports
type Report struct {
	ID        int       // id: bigint
	Board     string    // board: varchar
	PostID    int       // postid: int
	Timestamp time.Time // timestamp: timestamp
	IP        string    // ip: varchar
	Reason    string    // reason: varchar
	Cleared   bool      // cleared: tinyint
	IsTemp    bool      // istemp: tinyint
}

// DBPREFIXsections
type BoardSection struct {
	ID           int    // id: bigint
	ListOrder    int    // list_order: int
	Hidden       bool   // hidden: tinyint
	Name         string // name: varchar
	Abbreviation string // abbreviation: varchar
}
