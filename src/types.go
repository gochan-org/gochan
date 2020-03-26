package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/frustra/bbcode"
)

const (
	dirIsAFileStr = "unable to create \"%s\", path exists and is a file"
	pathExistsStr = "unable to create \"%s\", path already exists"
	genericErrStr = "unable to create \"%s\": %s"
)

var (
	config        GochanConfig
	readBannedIPs []string
	bbcompiler    bbcode.Compiler
	version       *GochanVersion
)

type RecentPost struct {
	BoardName string
	BoardID   int
	PostID    int
	ParentID  int
	Name      string
	Tripcode  string
	Message   string
	Filename  string
	ThumbW    int
	ThumbH    int
	IP        string
	Timestamp time.Time
}

func (p *RecentPost) GetURL(includeDomain bool) string {
	postURL := ""
	if includeDomain {
		postURL += config.SiteDomain
	}
	idStr := strconv.Itoa(p.PostID)
	postURL += config.SiteWebfolder + p.BoardName + "/res/"
	if p.ParentID == 0 {
		postURL += idStr + ".html#" + idStr
	} else {
		postURL += strconv.Itoa(p.ParentID) + ".html#" + idStr
	}
	return postURL
}

type Thread struct {
	OP            Post   `json:"-"`
	NumReplies    int    `json:"replies"`
	NumImages     int    `json:"images"`
	OmittedPosts  int    `json:"omitted_posts"`
	OmittedImages int    `json:"omitted_images"`
	BoardReplies  []Post `json:"-"`
	Sticky        int    `json:"sticky"`
	Locked        int    `json:"locked"`
	ThreadPage    int    `json:"-"`
}

// SQL Table structs

type Announcement struct {
	ID        uint   `json:"no"`
	Subject   string `json:"sub"`
	Message   string `json:"com"`
	Poster    string `json:"name"`
	Timestamp time.Time
}

type BanAppeal struct {
	ID            int
	Ban           int
	Message       string
	Denied        bool
	StaffResponse string
}

func (a *BanAppeal) GetBan() (*BanInfo, error) {
	var ban BanInfo
	err := queryRowSQL("SELECT * FROM DBPREFIXbanlist WHERE id = ? LIMIT 1",
		[]interface{}{a.ID}, []interface{}{
			&ban.ID, &ban.AllowRead, &ban.IP, &ban.Name, &ban.NameIsRegex, &ban.SilentBan,
			&ban.Boards, &ban.Staff, &ban.Timestamp, &ban.Expires, &ban.Permaban, &ban.Reason,
			&ban.StaffNote, &ban.AppealAt},
	)
	return &ban, err
}

type BanInfo struct {
	ID          uint
	AllowRead   bool
	IP          string
	Name        string
	NameIsRegex bool
	SilentBan   uint8
	Boards      string
	Staff       string
	Timestamp   time.Time
	Expires     time.Time
	Permaban    bool
	Reason      string
	Type        int
	StaffNote   string
	AppealAt    time.Time
	CanAppeal   bool
}

type BannedHash struct {
	ID          uint
	Checksum    string
	Description string
}

type Board struct {
	ID                     int            `json:"-"`
	CurrentPage            int            `json:"-"`
	NumPages               int            `json:"pages"`
	ListOrder              int            `json:"-"`
	Dir                    string         `json:"board"`
	Type                   int            `json:"-"`
	UploadType             int            `json:"-"`
	Title                  string         `json:"title"`
	Subtitle               string         `json:"meta_description"`
	Description            string         `json:"-"`
	Section                int            `json:"-"`
	MaxFilesize            int            `json:"max_filesize"`
	MaxPages               int            `json:"max_pages"`
	DefaultStyle           string         `json:"-"`
	Locked                 bool           `json:"is_archived"`
	CreatedOn              time.Time      `json:"-"`
	Anonymous              string         `json:"-"`
	ForcedAnon             bool           `json:"-"`
	MaxAge                 int            `json:"-"`
	AutosageAfter          int            `json:"bump_limit"`
	NoImagesAfter          int            `json:"image_limit"`
	MaxMessageLength       int            `json:"max_comment_chars"`
	EmbedsAllowed          bool           `json:"-"`
	RedirectToThread       bool           `json:"-"`
	ShowID                 bool           `json:"-"`
	RequireFile            bool           `json:"-"`
	EnableCatalog          bool           `json:"-"`
	EnableSpoileredImages  bool           `json:"-"`
	EnableSpoileredThreads bool           `json:"-"`
	Worksafe               bool           `json:"ws_board"`
	ThreadPage             int            `json:"-"`
	Cooldowns              BoardCooldowns `json:"cooldowns"`
	ThreadsPerPage         int            `json:"per_page"`
}

// AbsolutePath returns the full filepath of the board directory
func (board *Board) AbsolutePath(subpath ...string) string {
	return path.Join(config.DocumentRoot, board.Dir, path.Join(subpath...))
}

// WebPath returns a string that represents the file's path as accessible by a browser
// fileType should be "boardPage", "threadPage", "upload", or "thumb"
func (board *Board) WebPath(fileName string, fileType string) string {
	var filePath string
	switch fileType {
	case "":
		fallthrough
	case "boardPage":
		filePath = path.Join(config.SiteWebfolder, board.Dir, fileName)
	case "threadPage":
		filePath = path.Join(config.SiteWebfolder, board.Dir, "res", fileName)
	case "upload":
		filePath = path.Join(config.SiteWebfolder, board.Dir, "src", fileName)
	case "thumb":
		filePath = path.Join(config.SiteWebfolder, board.Dir, "thumb", fileName)
	}
	return filePath
}

func (board *Board) PagePath(pageNum interface{}) string {
	var page string
	pageNumStr := fmt.Sprintf("%v", pageNum)
	if pageNumStr == "prev" {
		if board.CurrentPage < 2 {
			page = "1"
		} else {
			page = strconv.Itoa(board.CurrentPage - 1)
		}
	} else if pageNumStr == "next" {
		if board.CurrentPage >= board.NumPages {
			page = strconv.Itoa(board.NumPages)
		} else {
			page = strconv.Itoa(board.CurrentPage + 1)
		}
	} else {
		page = pageNumStr
	}
	return board.WebPath(page+".html", "boardPage")
}

// Build builds the board and its thread files
// if newBoard is true, it adds a row to DBPREFIXboards and fails if it exists
// if force is true, it doesn't fail if the directories exist but does fail if it is a file
func (board *Board) Build(newBoard bool, force bool) error {
	var err error
	if board.Dir == "" {
		return errors.New("board must have a directory before it is built")
	}
	if board.Title == "" {
		return errors.New("board must have a title before it is built")
	}

	dirPath := board.AbsolutePath()
	resPath := board.AbsolutePath("res")
	srcPath := board.AbsolutePath("src")
	thumbPath := board.AbsolutePath("thumb")
	dirInfo, _ := os.Stat(dirPath)
	resInfo, _ := os.Stat(resPath)
	srcInfo, _ := os.Stat(srcPath)
	thumbInfo, _ := os.Stat(thumbPath)
	if dirInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, dirPath)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, dirPath)
		}
	} else {
		if err = os.Mkdir(dirPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, dirPath, err.Error())
		}
	}

	if resInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, resPath)
		}
		if !resInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, resPath)
		}
	} else {
		if err = os.Mkdir(resPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, resPath, err.Error())
		}
	}

	if srcInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, srcPath)
		}
		if !srcInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, srcPath)
		}
	} else {
		if err = os.Mkdir(srcPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, srcPath, err.Error())
		}
	}

	if thumbInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, thumbPath)
		}
		if !thumbInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, thumbPath)
		}
	} else {
		if err = os.Mkdir(thumbPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, thumbPath, err.Error())
		}
	}

	if newBoard {
		var numRows int
		queryRowSQL("SELECT COUNT(*) FROM DBPREFIXboards WHERE `dir` = ?",
			[]interface{}{board.Dir},
			[]interface{}{&numRows},
		)
		if numRows > 0 {
			return errors.New("board already exists in database")
		}
		board.CreatedOn = time.Now()
		var result sql.Result
		if result, err = execSQL("INSERT INTO DBPREFIXboards "+
			"(list_order,dir,type,upload_type,title,subtitle,description,"+
			"section,max_file_size,max_pages,default_style,locked,created_on,"+
			"anonymous,forced_anon,max_age,autosage_after,no_images_after,max_message_length,"+
			"embeds_allowed,redirect_to_thread,require_file,enable_catalog) "+
			"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
			board.ListOrder, board.Dir, board.Type, board.UploadType,
			board.Title, board.Subtitle, board.Description, board.Section,
			board.MaxFilesize, board.MaxPages, board.DefaultStyle,
			board.Locked, getSpecificSQLDateTime(board.CreatedOn), board.Anonymous,
			board.ForcedAnon, board.MaxAge, board.AutosageAfter,
			board.NoImagesAfter, board.MaxMessageLength, board.EmbedsAllowed,
			board.RedirectToThread, board.RequireFile, board.EnableCatalog,
		); err != nil {
			return err
		}
		boardID, _ := result.LastInsertId()
		board.ID = int(boardID)
	} else {
		if err = board.UpdateID(); err != nil {
			return err
		}
	}
	buildBoardPages(board)
	buildThreads(true, board.ID, 0)
	resetBoardSectionArrays()
	buildFrontPage()
	if board.EnableCatalog {
		buildCatalog(board.ID)
	}
	buildBoardListJSON()
	return nil
}

// PopulateData gets the board data from the database and sets the respective properties.
// if id > -1, the ID will be used to search the database. Otherwise dir will be used
func (board *Board) PopulateData(id int, dir string) error {
	queryStr := "SELECT * FROM DBPREFIXboards WHERE id = ?"
	var values []interface{}
	if id > -1 {
		values = append(values, id)
	} else {
		queryStr = "SELECT * FROM DBPREFIXboards WHERE dir = ?"
		values = append(values, dir)
	}

	return queryRowSQL(queryStr, values, []interface{}{
		&board.ID, &board.ListOrder, &board.Dir, &board.Type, &board.UploadType,
		&board.Title, &board.Subtitle, &board.Description, &board.Section,
		&board.MaxFilesize, &board.MaxPages, &board.DefaultStyle, &board.Locked,
		&board.CreatedOn, &board.Anonymous, &board.ForcedAnon, &board.MaxAge,
		&board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength,
		&board.EmbedsAllowed, &board.RedirectToThread, &board.RequireFile,
		&board.EnableCatalog})
}

func (board *Board) SetDefaults() {
	board.ListOrder = 0
	board.Section = 1
	board.MaxFilesize = 4096
	board.MaxPages = 11
	board.DefaultStyle = config.DefaultStyle
	board.Locked = false
	board.Anonymous = "Anonymous"
	board.ForcedAnon = false
	board.MaxAge = 0
	board.AutosageAfter = 200
	board.NoImagesAfter = 0
	board.MaxMessageLength = 8192
	board.EmbedsAllowed = true
	board.RedirectToThread = false
	board.ShowID = false
	board.RequireFile = false
	board.EnableCatalog = true
	board.EnableSpoileredImages = true
	board.EnableSpoileredThreads = true
	board.Worksafe = true
	board.ThreadsPerPage = 10
}

func (board *Board) UpdateID() error {
	return queryRowSQL("SELECT id FROM DBPREFIXboards WHERE dir = ?",
		[]interface{}{board.Dir},
		[]interface{}{&board.ID})
}

type BoardSection struct {
	ID           int
	ListOrder    int
	Hidden       bool
	Name         string
	Abbreviation string
}

// Post represents each post in the database
type Post struct {
	ID               int       `json:"no"`
	ParentID         int       `json:"resto"`
	CurrentPage      int       `json:"-"`
	BoardID          int       `json:"-"`
	Name             string    `json:"name"`
	Tripcode         string    `json:"trip"`
	Email            string    `json:"email"`
	Subject          string    `json:"sub"`
	MessageHTML      string    `json:"com"`
	MessageText      string    `json:"-"`
	Password         string    `json:"-"`
	Filename         string    `json:"tim"`
	FilenameOriginal string    `json:"filename"`
	FileChecksum     string    `json:"md5"`
	FileExt          string    `json:"extension"`
	Filesize         int       `json:"fsize"`
	ImageW           int       `json:"w"`
	ImageH           int       `json:"h"`
	ThumbW           int       `json:"tn_w"`
	ThumbH           int       `json:"tn_h"`
	IP               string    `json:"-"`
	Capcode          string    `json:"capcode"`
	Timestamp        time.Time `json:"time"`
	Autosage         bool      `json:"-"`
	DeletedTimestamp time.Time `json:"-"`
	Bumped           time.Time `json:"last_modified"`
	Stickied         bool      `json:"-"`
	Locked           bool      `json:"-"`
	Reviewed         bool      `json:"-"`
}

func (p *Post) GetURL(includeDomain bool) string {
	postURL := ""
	if includeDomain {
		postURL += config.SiteDomain
	}
	var board Board
	if err := board.PopulateData(p.BoardID, ""); err != nil {
		return postURL
	}

	idStr := strconv.Itoa(p.ID)
	postURL += config.SiteWebfolder + board.Dir + "/res/"
	if p.ParentID == 0 {
		postURL += idStr + ".html#" + idStr
	} else {
		postURL += strconv.Itoa(p.ParentID) + ".html#" + idStr
	}
	return postURL
}

// Sanitize escapes HTML strings in a post. This should be run immediately before
// the post is inserted into the database
func (p *Post) Sanitize() {
	p.Name = html.EscapeString(p.Name)
	p.Email = html.EscapeString(p.Email)
	p.Subject = html.EscapeString(p.Subject)
	p.Password = html.EscapeString(p.Password)
}

type Report struct {
	ID        uint
	Board     string
	PostID    uint
	Timestamp time.Time
	IP        string
	Reason    string
	Cleared   bool
	IsTemp    bool
}

type LoginSession struct {
	ID      uint
	Name    string
	Data    string
	Expires string
}

// Staff represents a single staff member's info stored in the database
type Staff struct {
	ID               int
	Username         string
	PasswordChecksum string
	Rank             int
	Boards           string
	AddedOn          time.Time
	LastActive       time.Time
}

type WordFilter struct {
	ID     int
	From   string
	To     string
	Boards string
	RegEx  bool
}

type BoardCooldowns struct {
	NewThread  int `json:"threads"`
	Reply      int `json:"replies"`
	ImageReply int `json:"images"`
}

type Style struct {
	Name     string
	Filename string
}

// GochanConfig stores crucial info and is read from/written to gochan.json
type GochanConfig struct {
	ListenIP   string
	Port       int
	FirstPage  []string
	Username   string
	UseFastCGI bool

	DocumentRoot string
	TemplateDir  string
	LogDir       string

	DBtype     string
	DBhost     string
	DBname     string
	DBusername string
	DBpassword string
	DBprefix   string

	Lockdown        bool     `description:"Disables posting." default:"unchecked"`
	LockdownMessage string   `description:"Message displayed when someone tries to post while the site is on lockdown."`
	Sillytags       []string `description:"List of randomly selected staff tags separated by line, e.g. <span style=\"color: red;\">## Mod</span>, to be randomly assigned to posts if UseSillytags is checked. Don't include the \"## \""`
	UseSillytags    bool     `description:"Use Sillytags" default:"unchecked"`
	Modboard        string   `description:"A super secret clubhouse board that only staff can view/post to." default:"staff"`

	SiteName      string `description:"The name of the site that appears in the header of the front page." default:"Gochan"`
	SiteSlogan    string `description:"The text that appears below SiteName on the home page"`
	SiteHeaderURL string `description:"To be honest, I'm not even sure what this does. It'll probably be removed later."`
	SiteWebfolder string `description:"The HTTP root appearing in the browser (e.g. https://gochan.org/&lt;SiteWebFolder&gt;" default:"/"`
	SiteDomain    string `description:"The server's domain (duh). Do not edit this unless you know what you are doing or BAD THINGS WILL HAPPEN!" default:"127.0.0.1" critical:"true"`
	DomainRegex   string `description:"Regular expression used for incoming request validation. Do not edit this unless you know what you are doing or BAD THINGS WILL HAPPEN!" default:"(https|http):\\\\/\\\\/(gochan\\\\.lunachan\\.net|gochan\\\\.org)\\/(.*)" critical:"true"`

	Styles       []Style `description:"List of styles (one per line) that should be accessed online at &lt;SiteWebFolder&gt;/css/&lt;Style&gt;/"`
	DefaultStyle string  `description:"Filename of the default Style. This should appear in the list above or bad things might happen."`

	AllowDuplicateImages bool     `description:"Disabling this will cause gochan to reject a post if the image has already been uploaded for another post.<br />This may end up being removed or being made board-specific in the future." default:"checked"`
	AllowVideoUploads    bool     `description:"Allows users to upload .webm videos. <br />This may end up being removed or being made board-specific in the future."`
	NewThreadDelay       int      `description:"The amount of time in seconds that is required before an IP can make a new thread.<br />This may end up being removed or being made board-specific in the future." default:"30"`
	ReplyDelay           int      `description:"Same as the above, but for replies." default:"7"`
	MaxLineLength        int      `description:"Any line in a post that exceeds this will be split into two (or more) lines.<br />I'm not really sure why this is here, so it may end up being removed." default:"150"`
	ReservedTrips        []string `description:"Secure tripcodes (!!Something) can be reserved here.<br />Each reservation should go on its own line and should look like this:<br />TripPassword1##Tripcode1<br />TripPassword2##Tripcode2"`

	ThumbWidth          int `description:"OP thumbnails use this as their max width.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger." default:"200"`
	ThumbHeight         int `description:"OP thumbnails use this as their max height.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger." default:"200"`
	ThumbWidth_reply    int `description:"Same as ThumbWidth and ThumbHeight but for reply images." default:"125"`
	ThumbHeight_reply   int `description:"Same as ThumbWidth and ThumbHeight but for reply images." default:"125"`
	ThumbWidth_catalog  int `description:"Same as ThumbWidth and ThumbHeight but for catalog images." default:"50"`
	ThumbHeight_catalog int `description:"Same as ThumbWidth and ThumbHeight but for catalog images." default:"50"`

	ThreadsPerPage           int      `default:"15"`
	RepliesOnBoardPage       int      `description:"Number of replies to a thread to show on the board page." default:"3"`
	StickyRepliesOnBoardPage int      `description:"Same as above for stickied threads." default:"1"`
	BanColors                []string `description:"Colors to be used for public ban messages (e.g. USER WAS BANNED FOR THIS POST).<br />Each entry should be on its own line, and should look something like this:<br />username1:#FF0000<br />username2:#FAF00F<br />username3:blue<br />Invalid entries/nonexistent usernames will show a warning and use the default red."`
	BanMsg                   string   `description:"The default public ban message." default:"USER WAS BANNED FOR THIS POST"`
	EmbedWidth               int      `description:"The width for inline/expanded webm videos." default:"200"`
	EmbedHeight              int      `description:"The height for inline/expanded webm videos." default:"164"`
	ExpandButton             bool     `description:"If checked, adds [Embed] after a Youtube, Vimeo, etc link to toggle an inline video frame." default:"checked"`
	ImagesOpenNewTab         bool     `description:"If checked, thumbnails will open the respective image/video in a new tab instead of expanding them." default:"unchecked"`
	MakeURLsHyperlinked      bool     `description:"If checked, URLs in posts will be turned into a hyperlink. If unchecked, ExpandButton and NewTabOnOutlinks are ignored." default:"checked"`
	NewTabOnOutlinks         bool     `description:"If checked, links to external sites will open in a new tab." default:"checked"`

	MinifyHTML          bool `description:"If checked, gochan will minify html files when building" default:"checked"`
	MinifyJS            bool `description:"If checked, gochan will minify js and json files when building" default:"checked"`
	UseMinifiedGochanJS bool `json:"-"`

	DateTimeFormat        string `description:"The format used for dates. See <a href=\"https://golang.org/pkg/time/#Time.Format\">here</a> for more info." default:"Mon, January 02, 2006 15:04 PM"`
	AkismetAPIKey         string `description:"The API key to be sent to Akismet for post spam checking. If the key is invalid, Akismet won't be used."`
	UseCaptcha            bool   `description:"If checked, a captcha will be generated"`
	CaptchaWidth          int    `description:"Width of the generated captcha image" default:"240"`
	CaptchaHeight         int    `description:"Height of the generated captcha image" default:"80"`
	CaptchaMinutesExpire  int    `description:"Number of minutes before a user has to enter a new CAPTCHA before posting. If <1 they have to submit one for every post." default:"15"`
	EnableGeoIP           bool   `description:"If checked, this enables the usage of GeoIP for posts." default:"checked"`
	GeoIPDBlocation       string `description:"Specifies the location of the GeoIP database file. If you're using CloudFlare, you can set it to cf to rely on CloudFlare for GeoIP information." default:"/usr/share/GeoIP/GeoIP.dat"`
	MaxRecentPosts        int    `description:"The maximum number of posts to show on the Recent Posts list on the front page." default:"3"`
	RecentPostsWithNoFile bool   `description:"If checked, recent posts with no image/upload are shown on the front page (as well as those with images" default:"unchecked"`
	EnableAppeals         bool   `description:"If checked, allow banned users to appeal their bans.<br />This will likely be removed (permanently allowing appeals) or made board-specific in the future." default:"checked"`
	MaxLogDays            int    `description:"The maximum number of days to keep messages in the moderation/staff log file."`
	RandomSeed            string `critical:"true"`
	TimeZone              int    `json:"-"`
}

func (cfg *GochanConfig) CheckString(val, defaultVal string, critical bool, msg string) string {
	if val == "" {
		val = defaultVal
		flags := lStdLog | lErrorLog
		if critical {
			flags |= lFatal
		}
		gclog.Print(flags, msg)
	}
	return val
}

func (cfg *GochanConfig) CheckInt(val, defaultVal int, critical bool, msg string) int {
	if val == 0 {
		val = defaultVal
		flags := lStdLog | lErrorLog
		if critical {
			flags |= lFatal
		}
		gclog.Print(flags, msg)
	}
	return val
}

func initConfig() {
	cfgPath := findResource("gochan.json", "/etc/gochan/gochan.json")
	if cfgPath == "" {
		fmt.Println("gochan.json not found")
		os.Exit(1)
	}

	jfile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", cfgPath, err.Error())
		os.Exit(1)
	}

	if err = json.Unmarshal(jfile, &config); err != nil {
		fmt.Printf("Error parsing %s: %s\n", cfgPath, err.Error())
		os.Exit(1)
	}

	config.LogDir = findResource(config.LogDir, "log", "/var/log/gochan/")
	if gclog, err = initLogs(
		path.Join(config.LogDir, "access.log"),
		path.Join(config.LogDir, "error.log"),
		path.Join(config.LogDir, "staff.log"),
	); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config.CheckString(config.ListenIP, "", true, "ListenIP not set in gochan.json, halting.")

	if config.Port == 0 {
		config.Port = 80
	}

	if len(config.FirstPage) == 0 {
		config.FirstPage = []string{"index.html", "board.html", "firstrun.html"}
	}

	config.Username = config.CheckString(config.Username, "gochan", false, "Username not set in gochan.json, using 'gochan' as default")
	config.DocumentRoot = config.CheckString(config.DocumentRoot, "gochan", true, "DocumentRoot not set in gochan.json, halting.")

	wd, wderr := os.Getwd()
	if wderr == nil {
		_, staterr := os.Stat(path.Join(wd, config.DocumentRoot, "css"))
		if staterr == nil {
			config.DocumentRoot = path.Join(wd, config.DocumentRoot)
		}
	}

	config.TemplateDir = config.CheckString(
		findResource(config.TemplateDir, "templates", "/usr/local/share/gochan/templates/", "/usr/share/gochan/templates/"),
		"", true, "Unable to locate template directory, halting.")

	config.CheckString(config.DBtype, "", true, "DBtype not set in gochan.json, halting (currently supported values: mysql,postgresql,sqlite3)")
	config.CheckString(config.DBhost, "", true, "DBhost not set in gochan.json, halting.")
	config.DBname = config.CheckString(config.DBname, "gochan", false,
		"DBname not set in gochan.json, setting to 'gochan'")

	config.CheckString(config.DBusername, "", true, "DBusername not set in gochan, halting.")
	config.CheckString(config.DBpassword, "", true, "DBpassword not set in gochan, halting.")
	config.LockdownMessage = config.CheckString(config.LockdownMessage,
		"The administrator has temporarily disabled posting. We apologize for the inconvenience", false, "")

	config.CheckString(config.SiteName, "", true, "SiteName not set in gochan.json, halting.")
	config.CheckString(config.SiteDomain, "", true, "SiteName not set in gochan.json, halting.")

	if config.SiteWebfolder == "" {
		gclog.Print(lErrorLog|lStdLog, "SiteWebFolder not set in gochan.json, using / as default.")
	} else if string(config.SiteWebfolder[0]) != "/" {
		config.SiteWebfolder = "/" + config.SiteWebfolder
	}
	if config.SiteWebfolder[len(config.SiteWebfolder)-1:] != "/" {
		config.SiteWebfolder += "/"
	}
	config.CheckString(config.DomainRegex, "", true, `DomainRegex not set in gochan.json. Consider using something like "(https|http):\\/\\/(gochan\\.org)\\/(.*)"`)
	//config.DomainRegex = "(https|http):\\/\\/(" + config.SiteDomain + ")\\/(.*)"

	if config.Styles == nil {
		gclog.Print(lErrorLog|lStdLog|lFatal, "Styles not set in gochan.json, halting.")
	}

	config.DefaultStyle = config.CheckString(config.DefaultStyle, config.Styles[0].Filename, false, "")

	config.NewThreadDelay = config.CheckInt(config.NewThreadDelay, 30, false, "")
	config.ReplyDelay = config.CheckInt(config.ReplyDelay, 7, false, "")
	config.MaxLineLength = config.CheckInt(config.MaxLineLength, 150, false, "")
	//ReservedTrips string //eventually this will be map[string]string

	config.ThumbWidth = config.CheckInt(config.ThumbWidth, 200, false, "")
	config.ThumbHeight = config.CheckInt(config.ThumbHeight, 200, false, "")
	config.ThumbWidth_reply = config.CheckInt(config.ThumbWidth_reply, 125, false, "")
	config.ThumbHeight_reply = config.CheckInt(config.ThumbHeight_reply, 125, false, "")
	config.ThumbWidth_catalog = config.CheckInt(config.ThumbWidth_catalog, 50, false, "")
	config.ThumbHeight_catalog = config.CheckInt(config.ThumbHeight_catalog, 50, false, "")

	config.ThreadsPerPage = config.CheckInt(config.ThreadsPerPage, 10, false, "")
	config.RepliesOnBoardPage = config.CheckInt(config.RepliesOnBoardPage, 3, false, "")
	config.StickyRepliesOnBoardPage = config.CheckInt(config.StickyRepliesOnBoardPage, 1, false, "")

	/*config.BanColors, err = c.GetString("threads", "ban_colors") //eventually this will be map[string] string
	if err != nil {
		config.BanColors = "admin:#CC0000"
	}*/

	config.BanMsg = config.CheckString(config.BanMsg, "(USER WAS BANNED FOR THIS POST)", false, "")
	config.DateTimeFormat = config.CheckString(config.DateTimeFormat, "Mon, January 02, 2006 15:04 PM", false, "")

	config.CaptchaWidth = config.CheckInt(config.CaptchaWidth, 240, false, "")
	config.CaptchaHeight = config.CheckInt(config.CaptchaHeight, 80, false, "")

	if config.EnableGeoIP {
		if config.GeoIPDBlocation == "" {
			gclog.Print(lErrorLog|lStdLog, "GeoIPDBlocation not set in gochan.json, disabling EnableGeoIP")
			config.EnableGeoIP = false
		}
	}

	if config.MaxLogDays == 0 {
		config.MaxLogDays = 15
	}

	if config.RandomSeed == "" {
		gclog.Print(lErrorLog|lStdLog, "RandomSeed not set in gochan.json, Generating a random one.")
		for i := 0; i < 8; i++ {
			num := rand.Intn(127-32) + 32
			config.RandomSeed += fmt.Sprintf("%c", num)
		}
		configJSON, _ := json.MarshalIndent(config, "", "\t")
		if err = ioutil.WriteFile(cfgPath, configJSON, 0777); err != nil {
			gclog.Printf(lErrorLog|lStdLog|lFatal, "Unable to write %s with randomly generated seed: %s", cfgPath, err.Error())
		}
	}

	_, zoneOffset := time.Now().Zone()
	config.TimeZone = zoneOffset / 60 / 60

	bbcompiler = bbcode.NewCompiler(true, true)
	bbcompiler.SetTag("center", nil)
	bbcompiler.SetTag("code", nil)
	bbcompiler.SetTag("color", nil)
	bbcompiler.SetTag("img", nil)
	bbcompiler.SetTag("quote", nil)
	bbcompiler.SetTag("size", nil)

	version = ParseVersion(versionStr)
	version.Normalize()
}
