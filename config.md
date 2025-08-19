ValueSpec doc: DirFileMode is the default file mode for directories created by gochan

ValueSpec doc: NormalFileMode is the default file mode for files created by gochan

ValueSpec doc: ConfigPathEnvVar is the environment variable used to set the path to gochan.js if it is set

# Configuration
See [gochan.example.json](examples/configs/gochan.example.json) for an example gochan.json.

**Make sure gochan has read-write permission for `DocumentRoot` and `LogDir` and read permission for `TemplateDir`**

Fields in the table marked as board options can be overridden on individual boards by adding them to  board.json, which gochan looks for in the board directory or in the same directory as gochan.json.

Field                      |Type                    |Board option |Default                                                                                |Info
---------------------------|------------------------|-------------|---------------------------------------------------------------------------------------|--------------
ListenAddress              |string                  |No           |                                                                                       |ListenAddress is the IP address or domain name that the server will listen on  
Port                       |int                     |No           |80                                                                                     |Port is the port that the server will listen on 
UseFastCGI                 |bool                    |No           |false                                                                                  |UseFastCGI tells the server to listen on FastCGI instead of HTTP if true 
DocumentRoot               |string                  |No           |                                                                                       |DocumentRoot is the path to the directory that contains the served static files  
TemplateDir                |string                  |No           |                                                                                       |TemplateDir is the path to the directory that contains the template files  
LogDir                     |string                  |No           |                                                                                       |LogDir is the path to the directory that contains the log files. It must be writable by the server and will be created if it doesn't exist  
Plugins                    |[]string                |No           |                                                                                       |Plugins is a list of paths to plugins to be loaded on startup. In Windows, only .lua plugins are supported. In Unix, .so plugins are also supported, but they must be compiled with the same Go version as the server and must be compiled in plugin mode  
WebRoot                    |string                  |No           |/                                                                                      |WebRoot is the base URL path that the server will serve files and generated pages from. 
SiteHost                   |string                  |No           |                                                                                       |SiteHost is the publicly accessible domain name or IP address of the site, e.g. "example.com" used for anti-spam checking  
CheckRequestReferer        |bool                    |No           |true                                                                                   |CheckRequestReferer tells the server to validate the Referer header from requests to prevent CSRF attacks. 
LogLevelStr                |string                  |No           |info                                                                                   |LogLevel determines the minimum level of log event to output. Any events lower than this level will be ignored. Valid values are "trace", "debug", "info", "warn", "error", "fatal", and "panic". 
RandomSeed                 |string                  |No           |                                                                                       |RandomSeed is a random string used for generating secure tokens. It will be generated if not set and must not be changed  
DBtype                     |string                  |No           |                                                                                       |DBtype is the type of SQL database to use. Currently supported values are "mysql", "postgres", and "sqlite3"  
DBhost                     |string                  |No           |                                                                                       |DBhost is the hostname or IP address of the SQL server, or the path to the SQLite database file. To connect to a MySQL database, set `DBhost` to "x.x.x.x:3306" (replacing x.x.x.x with your database server's IP or domain) or a different port, if necessary. You can also use a UNIX socket if you have it set up, like "unix(/var/run/mysqld/mysqld.sock)". To connect to a PostgreSQL database, set `DBhost` to the IP address or hostname. Using a UNIX socket may work as well, but it is currently untested.  
DBname                     |string                  |No           |                                                                                       |DBname is the name of the SQL database to connect to  
DBusername                 |string                  |No           |                                                                                       |DBusername is the username to use when authenticating with the SQL server  
DBpassword                 |string                  |No           |                                                                                       |DBpassword is the password to use when authenticating with the SQL server  
DBprefix                   |string                  |No           |                                                                                       |DBprefix is the prefix to add to table names in the database. It is not requried but may be useful if you need to share a database. Once you set it and do the initial setup, do not change it, as gochan will think the tables are missing and try to recreate them.  
DBTimeoutSeconds           |int                     |No           |15                                                                                     |DBTimeoutSeconds sets the timeout for SQL queries in seconds, 0 means no timeout. 
DBMaxOpenConnections       |int                     |No           |10                                                                                     |DBMaxOpenConnections is the maximum number of open connections to the database connection pool. 
DBMaxIdleConnections       |int                     |No           |10                                                                                     |DBMaxIdleConnections is the maximum number of idle connections to the database connection pool. 
DBConnMaxLifetimeMin       |int                     |No           |3                                                                                      |DBConnMaxLifetimeMin is the maximum lifetime of a connection in minutes. 
FirstPage                  |[]string                |No           |["index.html", "firstrun.html", "1.html"]                                              |FirstPage is a list of possible filenames to look for if a directory is requested 
Username                   |string                  |No           |                                                                                       |Username is the name of the user that the server will run as, if set, or the current user if empty or unset. It must be a valid user on the system if it is set  
CookieMaxAge               |string                  |No           |1y                                                                                     |CookieMaxAge is the parsed max age duration of cookies, e.g. "1 year 2 months 3 days 4 hours" or "1y2mo3d4h". 
StaffSessionDuration       |string                  |No           |3mo                                                                                    |StaffSessionDuration is the parsed max age duration of staff session cookies, e.g. "1 year 2 months 3 days 4 hours" or "1y2mo3d4h". 
SiteName                   |string                  |No           |Gochan                                                                                 |SiteName is the name of the site, displayed in the title and front page header 
SiteSlogan                 |string                  |No           |                                                                                       |SiteSlogan is the community slogan displayed on the front page below the site name  
MaxRecentPosts             |int                     |No           |15                                                                                     |MaxRecentPosts is the number of recent posts to display on the front page 
RecentPostsWithNoFile      |bool                    |No           |false                                                                                  |RecentPostsWithNoFile determines whether to include posts with no file in the recent posts list 
EnableAppeals              |bool                    |No           |true                                                                                   |EnableAppeals determines whether to allow users to appeal bans 
MinifyHTML                 |bool                    |No           |true                                                                                   |MinifyHTML tells the server to minify HTML output before sending it to the client 
MinifyJS                   |bool                    |No           |true                                                                                   |MinifyJS tells the server to minify JavaScript and JSON output before sending it to the client 
GeoIPType                  |string                  |No           |                                                                                       |GeoIPType is the type of GeoIP database to use. Currently only "mmdb" is supported, though other types may be provided by plugins  
GeoIPOptions               |map[string]any          |No           |                                                                                       |GeoIPOptions is a map of options to pass to the GeoIP plugin  
Captcha                    |CaptchaConfig           |No           |                                                                                       |Captcha options for spam prevention. Currently only hcaptcha is supported  
FingerprintVideoThumbnails |bool                    |No           |false                                                                                  |FingerprintVideoThumbnails determines whether to use video thumbnails for image fingerprinting. If false, the video file will not be checked by fingerprinting filters 
FingerprintHashLength      |int                     |No           |16                                                                                     |FingerprintHashLength is the length of the hash used for image fingerprinting 
InheritGlobalStyles        |bool                    |Yes          |true                                                                                   |InheritGlobalStyles determines whether to use the global styles in addition to the board's styles, as opposed to only the board's styles 
Styles                     |[]Style                 |Yes          |                                                                                       |Styles is a list of Gochan themes with Name and Filename fields, choosable by the user  
DefaultStyle               |string                  |Yes          |pipes.css                                                                              |DefaultStyle is the filename of the default style to use for the board or the site. If it is not set, the first style in the Styles list will be used 
IncludeGlobalStyles        |[]string                |Yes          |                                                                                       |IncludeGlobalStyles is a list of additional CSS files to be loaded on the board pages, or all pages if this is the global configuration.  
IncludeScripts             |[]IncludeScript         |Yes          |                                                                                       |IncludeScripts is a list of additional scripts to be loaded on the board pages, or all pages if this is the global configuration.  
Banners                    |[]PageBanner            |Yes          |                                                                                       |Banners is a list of banners to display on the board's front page, with Filename, Width, and Height fields  
Lockdown                   |bool                    |Yes          |false                                                                                  |Lockdown prevents users from posting if true 
LockdownMessage            |string                  |Yes          |This imageboard has temporarily disabled posting. We apologize for the inconvenience   |LockdownMessage is the message displayed to users if they try to cretae a post when the site is in lockdown 
DateTimeFormat             |string                  |Yes          |Mon, January 02, 2006 3:04:05 PM                                                       |DateTimeFormat is the human readable format to use for showing post timestamps. See [the official documentation](https://pkg.go.dev/time#Time.Format) for more information. 
ShowPosterID               |bool                    |Yes          |false                                                                                  |ShowPosterID determines whether to show the generated thread-unique poster ID in the post header (not yet implemented) 
EnableSpoileredImages      |bool                    |Yes          |true                                                                                   |EnableSpoileredImages determines whether to allow users to spoiler images (not yet implemented) 
EnableSpoileredThreads     |bool                    |Yes          |true                                                                                   |EnableSpoileredThreads determines whether to allow users to spoiler threads (not yet implemented) 
Worksafe                   |bool                    |Yes          |true                                                                                   |Worksafe determines whether the board is worksafe or not. If it is set to true, threads cannot be marked NSFW (given a hashtag with the text NSFW, case insensitive). 
Cooldowns                  |BoardCooldowns          |Yes          |                                                                                       |Cooldowns is used to prevent spamming by setting the number of seconds the user must wait before creating new threads or replies  
RenderURLsAsLinks          |bool                    |Yes          |true                                                                                   |RenderURLsAsLinks determines whether to render URLs as clickable links in posts 
ThreadsPerPage             |int                     |Yes          |20                                                                                     |ThreadsPerPage is the number of threads to display per page 
EnableGeoIP                |bool                    |Yes          |false                                                                                  |EnableGeoIP shows a dropdown box allowing the user to set their post flag as their country 
EnableNoFlag               |bool                    |Yes          |false                                                                                  |EnableNoFlag allows the user to post without a flag. It is only used if EnableGeoIP or CustomFlags is true 
CustomFlags                |[]geoip.Country         |Yes          |                                                                                       |CustomFlags is a list of non-geoip flags with Name (viewable to the user) and Flag (flag image filename) fields  
MaxMessageLength           |int                     |Yes          |2000                                                                                   |MaxMessageLength is the maximum number of characters allowed in a post 
ReservedTrips              |map[string]string       |Yes          |                                                                                       |ReservedTrips is used for reserving secure tripcodes. It should be a map of input strings to output tripcode strings. For example, if you have `{"abcd":"WXYZ"}` and someone posts with the name Name##abcd, their name will instead show up as Name!!WXYZ on the site.  
ThreadsPerPage             |int                     |Yes          |20                                                                                     |ThreadsPerPage is the number of threads to display per page 
RepliesOnBoardPage         |int                     |Yes          |3                                                                                      |RepliesOnBoardPage is the number of replies to display on the board page 
StickyRepliesOnBoardPage   |int                     |Yes          |1                                                                                      |StickyRepliesOnBoardPage is the number of replies to display on the board page for sticky threads 
NewThreadsRequireUpload    |bool                    |Yes          |false                                                                                  |NewThreadsRequireUpload determines whether to require an upload to create a new thread 
EnableCyclicThreads        |bool                    |Yes          |true                                                                                   |EnableCyclicThreads allows users to create threads that have a maximum number of replies before the oldest reply is deleted 
CyclicThreadNumPosts       |int                     |Yes          |500                                                                                    |CyclicThreadNumPost determines the number of posts a cyclic thread can have before the oldest post is deleted 
BanColors                  |map[string]string       |Yes          |                                                                                       |BanColors is a list of colors to use for the ban message with the staff name as the key. If the staff name is not found in the list, the default style color will be used.  
BanMessage                 |string                  |Yes          |USER WAS BANNED FOR THIS POST                                                          |BanMessage is the default message shown on a post that a user was banned for 
EmbedWidth                 |int                     |Yes          |400                                                                                    |EmbedWidth is the width of embedded external media files 
EmbedHeight                |int                     |Yes          |300                                                                                    |EmbedHeight is the height of embedded external media files 
EmbedMatchers              |map[string]EmbedMatcher |Yes          |                                                                                       |EmbedMatchers is a map of site ID keys to objects used to match (via regular expression) URLs and embed them in posts via templates, with an optional image thumbnail if supported. If a URL template is not provided, the video/frame will be embedded directly. If EmbedMatchers is nil, embedding is disabled for the board, or globally if it is in the global configuration.  
ImagesOpenNewTab           |bool                    |Yes          |true                                                                                   |ImagesOpenNewTab determines whether to open images in a new tab when an image link is clicked 
NewTabOnExternalLinks      |bool                    |Yes          |true                                                                                   |NewTabOnExternalLinks determines whether to open external links in a new tab 
DisableBBcode              |bool                    |Yes          |false                                                                                  |DisableBBcode will disable BBCode to HTML conversion if true 
AllowDiceRerolls           |bool                    |Yes          |false                                                                                  |AllowDiceRerolls determines whether to allow users to edit posts to reroll dice 
RejectDuplicateUploads     |bool                    |Yes          |false                                                                                  |RejectDuplicateUploads determines whether to reject images and videos that have already been uploaded 
ThumbWidth                 |int                     |Yes          |200                                                                                    |ThumbWidth is the maximum width that thumbnails in the top thread post will be scaled down to 
ThumbHeight                |int                     |Yes          |200                                                                                    |ThumbHeight is the maximum height that thumbnails in the top thread post will be scaled down to 
ThumbWidthReply            |int                     |Yes          |125                                                                                    |ThumbWidthReply is the maximum width that thumbnails in thread replies will be scaled down to 
ThumbHeightReply           |int                     |Yes          |125                                                                                    |ThumbHeightReply is the maximum height that thumbnails in thread replies will be scaled down to 
ThumbWidthCatalog          |int                     |Yes          |50                                                                                     |ThumbWidthCatalog is the maximum width that thumbnails on the board catalog page will be scaled down to 
ThumbHeightCatalog         |int                     |Yes          |50                                                                                     |ThumbHeightCatalog is the maximum height that thumbnails on the board catalog page will be scaled down to 
AllowOtherExtensions       |map[string]string       |Yes          |                                                                                       |AllowOtherExtensions is a map of file extensions to use for uploads that are not images or videos The key is the extension (e.g. ".pdf") and the value is the filename of the thumbnail to use in /static  
StripImageMetadata         |string                  |Yes          |                                                                                       |StripImageMetadata sets what (if any) metadata to remove from uploaded images using exiftool. Valid values are "", "none" (has the same effect as ""), "exif", or "all" (for stripping all metadata)  
ExiftoolPath               |string                  |Yes          |                                                                                       |ExiftoolPath is the path to the exiftool command. If unset or empty, the system path will be used to find it  

Example options for `GeoIPOptions`:
```JSONC
"GeoIPType": "mmdb",
"GeoIPOptions": {
	"dbLocation": "/usr/share/geoip/GeoIP2.mmdb",
	"isoCode": "en" // optional
}
```

`CustomFlags` is an array with custom post flags, selectable via dropdown. The `Flag` value is assumed to be a file in /static/flags/. Example:
```JSON
"CustomFlags": [
	{"Flag":"california.png", "Name": "California"},
	{"Flag":"cia.png", "Name": "CIA"},
	{"Flag":"lgbtq.png", "Name": "LGBTQ"},
	{"Flag":"ms-dos.png", "Name": "MS-DOS"},
	{"Flag":"stallman.png", "Name": "Stallman"},
	{"Flag":"templeos.png", "Name": "TempleOS"},
	{"Flag":"tux.png", "Name": "Linux"},
	{"Flag":"windows9x.png", "Name": "Windows 9x"}
]
```

## CaptchaConfig
Field                |Type   |Info
---------------------|-------|--------------
Type                 |string |Type is the type of captcha to use. Currently only "hcaptcha" is supported  
OnlyNeededForThreads |bool   |OnlyNeededForThreads determines whether to require a captcha only when creating a new thread, or for all posts  
SiteKey              |string |SiteKey is the public key for the captcha service. Usage depends on the captcha service  
AccountSecret        |string |AccountSecret is the secret key for the captcha service. Usage depends on the captcha service  

## PageBanner
PageBanner represents the filename and dimensions of a banner image to display on board and thread pages
Field    |Type   |Info
---------|-------|--------------
Filename |string |Filename is the name of the image file to display as seen by the browser  
Width    |int    |Width is the width of the image in pixels  
Height   |int    |Height is the height of the image in pixels  

## BoardCooldowns
Field      |Type  |Default    |Info
-----------|------|-----------|--------------
NewThread  |int   |30         |NewThread is the number of seconds the user must wait before creating new threads. 
Reply      |int   |7          |NewReply is the number of seconds the user must wait after replying to a thread before they can create another reply. 
ImageReply |int   |7          |NewImageReply is the number of seconds the user must wait after replying to a thread with an upload before they can create another reply. 

## geoip.Country
Country represents the country data (or custom flag data) used by gochan.
Field  |Type   |Info
-------|-------|--------------
Flag   |string |Flag is the country abbreviation for standard geoip countries, or the filename accessible in /static/flags/{flag} for custom flags  
Name   |string |Name is the configured flag name that shows up in the dropdown box and the image alt text  

