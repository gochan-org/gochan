package main 

import (
	"strconv"
	"strings"
	"go-logfile/logfile"
	"goconf/conf"
)

type BanlistTable struct {
	id int
	Type bool
	expired bool
	allowread bool
	ip string
	ipmd5 string
	globalban bool
	silentban bool
	boards string
	by string
	at int
	until int
	reason string
	staffnote string
	appeal string
	appealat int
}

type BannedHashesTable struct {
	id int
	md5 string
	bantime int
	description string
}

type BannedTripcodesTable struct {
	id int
	name string
	tripcode string
}

type BlotterTable struct {
	id int
	important bool
	at int
	message string
}

type BoardFiletypesTable struct {
	boardid int
	typeid int
}

type BoardsTable struct {
	id int
	order int
	name string
	Type bool 
	start int
	uploadtype bool 
	desc string
	image string
	section int
	maximagesize int
	maxpages int
	maxage int
	markpage int
	maxreplies int
	messagelength int
	createdon int
	locked bool 
	includeheader string 
	redirecttothread bool 
	anonymous string
	forcedanon bool 
	embeds_allowed string
	trial bool 
	popular bool 
	defaultstyle string
	locale string
	showid bool 
	compactlist bool 
	enablereporting bool 
	enablecaptcha bool 
	enablenofile bool 
	enablearchiving bool 
	enablecatalog bool 
	loadbalanceurl string
	loadbalancepassword string
}

type BoardSectionsTable struct {
	id int
	order int
	hidden bool
	name string
	abbreviation string
}

type EmbedsTable struct {
	id int
	filetype string
	name string
	videourl string
	width int
	height int
	code string
}

type FiletypesTable struct {
	id int
	filetype string
	mime string
	image string
	image_w int
	image_h int
	force_thumb bool
}

type FrontTable struct {
	id int
	page int
	order int
	subject string
	message string
	timestamp int
	poster string
	email string
}

type FrontLinksTable struct {
	title string
	url string
}

type LoginAttemptsTable struct {
	username string
	ip string
	timestamp int
}

type ModLogTable struct {
	entry string
	user string
	category int
	timestamp int
}

type ModpageAnnouncementsTable struct {
	id int
	parentid int
	subject string
	postedat int
	postedby string
	message string
}

type PollTable struct {
	ip string
	selection string
	time int
}

type PostTable struct {
	id int
	boardid int
	parentid int
	name string
	tripcode string
	email string
	subject string
	message string
	password string
	file string
	file_md5 string
	file_type string
	file_original string
	file_size int
	file_size_formatted string
	image_w int
	image_h int
	thumb_w int
	thumb_h int
	ip string
	ipmd5 string
	tag string
	timestamp int
	stickied bool
	locked bool
	autosage int
	posterauthority int
	reviewed bool
	deleted_timestamp int
	IS_DELETED bool
	bumped int
	sillytag string
}

type ReportsTable struct {
	id int
	cleared bool
	board string
	postid int
	when int
	ip string
	reason string
}

type StaffTable struct {
	id int
	username string
	password string
	salt string
	Type int
	boards string
	addedon int
	lastactive int
	em_contact string
}

type TrackerTable struct {
	index int
	search_query string
	found_names string
	found_ips string
}

type WatchedThreadsTable struct {
	id int
	threadid int
	board string
	ip string
	lastsawreplyid int
}

type WordFiltersTable struct {
	id int
	word string
	replacedby string
	boards string
	time int
	regex bool
}

var (
	needs_initial_setup = true
	config,_ = conf.ReadConfigFile("config.cfg")
	log_dir,_ = config.GetString("server","log_dir")
	access_log,_ = logfile.OpenLogFile(log_dir+"/access.log",false)
	error_log,_ = logfile.OpenLogFile(log_dir+"/error.log",false)
	document_root,_ = config.GetString("server","document_root")
	first_page_str,_ = config.GetString("server","first_page")
	first_page = strings.Split(first_page_str,",")
	domain,_ = config.GetString("server","domain")
	port,_ = config.GetInt("server","port")
	db_type,_ = config.GetString("database","type")
	db_name,_ = config.GetString("database", "name")
	db_host,_ = config.GetString("database","host")
	db_username,_ = config.GetString("database","username")
	db_password,_ = config.GetString("database","password")
	db_prefix,_ = config.GetString("database","prefix")
	db_persistent,_ = config.GetBool("database","keepalive")
	db_persistent_str = strconv.Itoa(Btoi(db_persistent))
	lockdown bool
	lockdown_message string
	sillytags string
	use_sillytags bool
	site_name string
	site_slogan string
	site_headerurl bool
	site_irc string
	site_banreason string
	site_allowdupes bool
	webfolder string
	webpath string
	root_dir string
	template_dir string   
	cached_template_dir string
	styles,_ = config.GetString("styles","styles")
	styles_arr = strings.Split(styles,",")
	default_style string
	style_switcher bool
	dropdown_style_switcher bool
	styles_txt []string
	default_txt_style string
	txt_style_switcher bool
	menu_type string
	menu_styles []string  
	default_menu_style string
	menu_style_switcher bool
	new_thread_delay int
	reply_delay int
	line_length int
	thumb_width int  
	thumb_height int  
	reply_thumb_width int
	reply_thumb_height int
	catalog_thumb_width int
	catalog_thumb_height int
	thumb_method string
	animated_thumbs bool
	new_window bool
	make_links bool 
	no_message_thread bool
	no_message_reply bool
	img_threads_per_page int       
	txt_threads_per_page int
	replies_on_boardpage int
	sticky_replies_on_boardpage int
	thumb_msg bool
	ban_colors []string
	ban_msg string
	traditional_read bool
	youtube_width int
	youtube_height int
	use_dir_title bool
	make_rss bool
	expand bool
	quick_reply bool
	watched_threads bool
	gen_firstlast_pages bool
	use_blotter bool
	make_sitemap bool
	enable_appeals bool
	max_modlog_days int
	random_seed string
	use_static_menu bool
	generate_boardlist bool
)