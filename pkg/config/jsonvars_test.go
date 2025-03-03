package config

import "strings"

const (
	// the bare minimum fields required to pass GochanConfig.validate.
	// this doesn't mean that the values are valid, just that they exist
	bareMinimumJSON = `{
		"ListenAddress": "127.0.0.1",
		"Port": 8080,
		"Username": "gochan",
		"UseFastCGI": true,
		"DBtype": "mysql",
		"DBhost": "127.0.0.1:3306",
		"DBname": "gochan",
		"DBusername": "gochan",
		"DBpassword": "",
		"SiteHost": "127.0.0.1",
		"SiteWebFolder": "/",
	
		"Styles": [
			{ "Name": "Pipes", "Filename": "pipes.css" },
			{ "Name": "Burichan", "Filename": "burichan.css" },
			{ "Name": "Dark", "Filename": "dark.css" },
			{ "Name": "Photon", "Filename": "photon.css" }
		],
		"RandomSeed": "jeiwohaeiogpehwgui"
	}`
	validCfgJSON = `{
		"ListenAddress": "127.0.0.1",
		"Port": 8080,
		"FirstPage": ["index.html","firstrun.html","1.html"],
		"Username": "gochan",
		"UseFastCGI": false,
		"DebugMode": false,
	
		"DocumentRoot": "html",
		"TemplateDir": "templates",
		"LogDir": "log",
	
		"DBtype": "mysql",
		"DBtype_alt": "postgres",
		"DBhost": "127.0.0.1:3306",
		"_comment": "gochan can use either a URL or a UNIX socket for MySQL connections",
		"DBname": "gochan",
		"DBusername": "gochan",
		"DBpassword": "",
		"DBprefix": "gc_",
	
		"Lockdown": false,
		"LockdownMessage": "This imageboard has temporarily disabled posting. We apologize for the inconvenience",
		"Modboard": "staff",
	
		"SiteName": "Gochan",
		"SiteSlogan": "",
		"SiteHost": "127.0.0.1",
		"SiteWebFolder": "/",
	
		"Styles": [
			{ "Name": "Pipes", "Filename": "pipes.css" },
			{ "Name": "Burichan", "Filename": "burichan.css" },
			{ "Name": "Dark", "Filename": "dark.css" },
			{ "Name": "Photon", "Filename": "photon.css" }
		],
		"DefaultStyle": "pipes.css",
	
		"RejectDuplicateImages": true,
		"NewThreadDelay": 30,
		"ReplyDelay": 7,
		"MaxLineLength": 150,
		"ReservedTrips": {
			"thischangesto": "this",
			"andthischangesto": "this"
		},
	
		"ThumbWidth": 200,
		"ThumbHeight": 200,
		"ThumbWidthReply": 125,
		"ThumbHeightReply": 125,
		"ThumbWidthCatalog": 50,
		"ThumbHeightCatalog": 50,
	
		"ThreadsPerPage": 15,
		"RepliesOnBoardPage": 3,
		"StickyRepliesOnBoardPage": 1,
		"BanMessage": "USER WAS BANNED FOR THIS POST",
		"EmbedWidth": 200,
		"EmbedHeight": 164,
		"EnableEmbeds": true,
		"ImagesOpenNewTab": true,
		"MakeURLsHyperlinked": true,
		"NewTabOnOutlinks": true,
	
		"MinifyHTML": true,
		"MinifyJS": true,
	
		"DateTimeFormat": "Mon, January 02, 2006 15:04 PM",
		"UseCaptcha": false,
		"CaptchaWidth": 240,
		"CaptchaHeight": 80,
		"CaptchaMinutesExpire": 15,
		"EnableGeoIP": true,
		"_comment": "set GeoIPDBlocation to cf to use Cloudflare's GeoIP",
		"GeoIPDBlocation": "/usr/share/GeoIP/GeoIP.dat",
		"MaxRecentPosts": 12,
		"RecentPostsWithNoFile": false,
		"EnableAppeals": true,
		"MaxLogDays": 14,
		"_comment": "Set RandomSeed to a (preferrably large) string of letters and numbers",
		"RandomSeed": ""
	}`
)

var (
	badTypeJSON = strings.ReplaceAll(validCfgJSON, `"RandomSeed": ""`, `"RandomSeed": 32`)
)
