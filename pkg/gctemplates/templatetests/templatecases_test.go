package templatetests_test

import (
	"bytes"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
)

const (
	headBeginning = `<!DOCTYPE html><html lang="en"><head>` +
		`<meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">`

	headEndAndBodyStart = `<link rel="stylesheet"href="/css/global.css"/>` +
		`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
		`<link rel="shortcut icon"href="/favicon.png">` +
		`<script type="text/javascript"src="/js/consts.js"></script>` +
		`<script type="text/javascript"src="/js/gochan.js"defer></script></head>` +
		`<body><div id="topbar"><div class="topbar-section"><a href="/"class="topbar-item">home</a></div></div>` +
		`<header><h1 id="board-title">Gochan</h1></header>` +
		`<div id="content"><div class="section-block banpage-block">`

	normalBanHeader = headBeginning + `<title>YOU ARE BANNED:(</title>` + headEndAndBodyStart +
		`<div class="section-title-block"><span class="section-title ban-title">YOU ARE BANNED:(</span></div>` +
		`<div class="section-body"><div id="ban-info">`

	bannedForeverHeader = headBeginning + `<title>YOU'RE PERMABANNED,&nbsp;IDIOT!</title>` + headEndAndBodyStart +
		`<div class="section-title-block"><span class="section-title ban-title">YOU'RE PERMABANNED,IDIOT!</span></div>` +
		`<div class="section-body"><div id="ban-info">`

	appealForm = `<form id="appeal-form"action="/post"method="POST">` +
		`<input type="hidden"name="board"value=""><input type="hidden"name="banid"value="0">` +
		`<textarea rows="4"cols="48"name="appealmsg"id="postmsg"placeholder="Appeal message"></textarea>` +
		`<input type="submit"name="doappeal"value="Submit"/><br/></form>`

	footer = `<footer>Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 4.0</a><br /></footer></div></body></html>`
)

var (
	testingSiteConfig = config.SiteConfig{
		SiteName:   "Gochan",
		SiteSlogan: "Gochan test",
	}
	simpleBoardConfig = config.BoardConfig{
		DefaultStyle: "pipes.css",
		Styles: []config.Style{
			{Name: "Pipes", Filename: "pipes.css"},
			{Name: "Yotsuba A", Filename: "yotsuba.css"},
			{Name: "Yotsuba B", Filename: "yotsubab.css"},
		},
		Banners: []config.PageBanner{
			{Filename: "banner1.png", Width: 300, Height: 100},
			{Filename: "banner2.png", Width: 300, Height: 100},
			{Filename: "banner3.png", Width: 300, Height: 100},
		},
	}

	simpleBoard1 = &gcsql.Board{
		ID:            1,
		SectionID:     1,
		URI:           "test",
		Dir:           "test",
		Title:         "Testing board",
		Subtitle:      "Board for testing",
		Description:   "Board for testing",
		DefaultStyle:  "pipes.css",
		AnonymousName: "Anonymous Coward",
	}

	simpleBoard2 = &gcsql.Board{
		ID:            2,
		SectionID:     2,
		URI:           "sup",
		Dir:           "sup",
		Title:         "Gochan Support board",
		Subtitle:      "Board for helping out gochan users/admins",
		Description:   "Board for helping out gochan users/admins",
		DefaultStyle:  "yotsuba.css",
		AnonymousName: "Anonymous Coward",
	}

	banPageCases = []templateTestCase{
		{
			desc: "appealable permaban",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						Permanent: true,
						CanAppeal: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: normalBanHeader +
				`You are banned from posting on<span class="ban-boards">all boards</span>for the following reason:<p class="reason">ban message goes here</p>` +
				`Your ban was placed on<time datetime="0001-01-01T00:00:00Z"class="ban-timestamp">Mon,January 01,0001 12:00:00 AM</time> and will <span class="ban-timestamp">not expire</span>.<br/>` +
				`Your IP address is<span class="ban-ip">192.168.56.1</span>.<br /><br/>` +
				`You may appeal this ban:<br/>` + appealForm + `</div></div></div>` +
				footer,
		},
		{
			desc: "unappealable permaban (banned forever)",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						IsActive:  true,
						Permanent: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: bannedForeverHeader + `You are banned from posting on<span class="ban-boards">all boards</span>for the following reason:` +
				`<p class="reason">ban message goes here</p>Your ban was placed on<time datetime="0001-01-01T00:00:00Z"class="ban-timestamp">Mon,January 01,0001 12:00:00 AM</time> ` +
				`and will <span class="ban-timestamp">not expire</span>.<br/>` +
				`Your IP address is<span class="ban-ip">192.168.56.1</span>.<br /><br/>You may<span class="ban-timestamp">not</span> appeal this ban.<br /></div>` +
				`<img id="banpage-image" src="/static/permabanned.jpg"/><br/>` +
				`<audio id="jack"preload="auto"autobuffer loop><source src="/static/hittheroad.ogg"/><source src="/static/hittheroad.wav"/><source src="/static/hittheroad.mp3"/></audio>` +
				`<script type="text/javascript">document.getElementById("jack").play();</script></div></div>` +
				footer,
		},
		{
			desc: "appealable temporary ban",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						CanAppeal: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: normalBanHeader +
				`You are banned from posting on<span class="ban-boards">all boards</span>for the following reason:<p class="reason">ban message goes here</p>Your ban was placed on<time datetime="0001-01-01T00:00:00Z"class="ban-timestamp">Mon,January 01,0001 12:00:00 AM</time> and will expire on <time class="ban-timestamp" datetime="0001-01-01T00:00:00Z">Mon, January 01, 0001 12:00:00 AM</time>.<br/>Your IP address is<span class="ban-ip">192.168.56.1</span>.<br /><br/>You may appeal this ban:<br/><form id="appeal-form"action="/post"method="POST"><input type="hidden"name="board"value=""><input type="hidden"name="banid"value="0"><textarea rows="4"cols="48"name="appealmsg"id="postmsg"placeholder="Appeal message"></textarea><input type="submit"name="doappeal"value="Submit"/><br/></form></div></div></div><footer>Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 4.0</a><br /></footer></div></body></html>`,
		},
		{
			desc: "unappealable temporary ban",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						StaffID: 1,
						Message: "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: normalBanHeader + `You are banned from posting on<span class="ban-boards">all boards</span>for the following reason:` +
				`<p class="reason">ban message goes here</p>` +
				`Your ban was placed on<time datetime="0001-01-01T00:00:00Z"class="ban-timestamp">Mon,January 01,0001 12:00:00 AM</time> ` +
				`and will expire on <time class="ban-timestamp" datetime="0001-01-01T00:00:00Z">Mon, January 01, 0001 12:00:00 AM</time>.<br/>` +
				`Your IP address is<span class="ban-ip">192.168.56.1</span>.<br /><br/>You may<span class="ban-timestamp">not</span> appeal this ban.<br />` +
				`</div></div></div>` + footer,
		},
	}

	boardPageTestCases = []templateTestCase{
		{
			desc: "base case, no threads",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			expectedOutput: boardPageHeaderBase +
				`<form action="/util"method="POST"id="main-form"><div id="right-bottom-content"><div id="report-delbox"><input type="hidden"name="board"value="test"/><input type="hidden"name="boardid"value="1"/><label>[<input type="checkbox"name="fileonly"/>File only]</label> <input type="password" size="10" name="password" id="delete-password" /><input type="submit"name="delete_btn"value="Delete"onclick="return confirm('Are you sure you want to delete these posts?')"/><br/>Report reason:<input type="text"size="10"name="reason"id="reason"/><input type="submit"name="report_btn"value="Report"/><br/><input type="submit"name="edit_btn"value="Edit post"/>&nbsp;<input type="submit"name="move_btn"value="Move thread"/></div></div></form><div id="left-bottom-content"><a href="#"onClick="window.location.reload(); return false;">Update</a>|<a href="#">Scroll to top</a><br/><table id="pages"><tr><td>[<a href="/test/1.html">1</a>]</td></tr></table><span id="boardmenu-bottom">[<a href="/">home</a>]&nbsp;[]</span></div>` +
				footer,
		},
		{
			desc: "base case, multi threads and pages",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			expectedOutput: boardPageHeaderBase +
				`<form action="/util"method="POST"id="main-form"><div id="right-bottom-content"><div id="report-delbox"><input type="hidden"name="board"value="test"/><input type="hidden"name="boardid"value="1"/><label>[<input type="checkbox"name="fileonly"/>File only]</label> <input type="password" size="10" name="password" id="delete-password" /><input type="submit"name="delete_btn"value="Delete"onclick="return confirm('Are you sure you want to delete these posts?')"/><br/>Report reason:<input type="text"size="10"name="reason"id="reason"/><input type="submit"name="report_btn"value="Report"/><br/><input type="submit"name="edit_btn"value="Edit post"/>&nbsp;<input type="submit"name="move_btn"value="Move thread"/></div></div></form><div id="left-bottom-content"><a href="#"onClick="window.location.reload(); return false;">Update</a>|<a href="#">Scroll to top</a><br/><table id="pages"><tr><td>[<a href="/test/1.html">1</a>]</td></tr></table><span id="boardmenu-bottom">[<a href="/">home</a>]&nbsp;[]</span></div>` +
				footer,
		},
	}

	jsConstsCases = []templateTestCase{
		{
			desc: "base test",
			data: map[string]any{
				"styles": []config.Style{
					{Name: "Pipes", Filename: "pipes.css"},
					{Name: "Yotsuba A", Filename: "yotsuba.css"},
				},
				"defaultStyle": "pipes.css",
				"webroot":      "/",
				"timezone":     -1,
				"fileTypes": map[string]string{
					".ext": "thumb.png",
				},
			},
			expectedOutput: `const styles=[{Name:"Pipes",Filename:"pipes.css"},{Name:"Yotsuba A",Filename:"yotsuba.css"}];const defaultStyle="pipes.css";const webroot="/";const serverTZ=-1;const fileTypes=[".ext",];`,
		},
		{
			desc: "empty values",
			data: map[string]any{
				"defaultStyle": "",
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `const styles=[];const defaultStyle="";const webroot="";const serverTZ=0;const fileTypes=[];`,
		},
		{
			desc: "escaped string",
			data: map[string]any{
				"defaultStyle": `"a\a"`,
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `const styles=[];const defaultStyle="\&#34;a\\a\&#34;";const webroot="";const serverTZ=0;const fileTypes=[];`,
		},
	}

	baseFooterCases = []templateTestCase{
		{
			desc: "base footer test",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			expectedOutput: footer,
		},
		{
			desc: "base footer test",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard2,
				"numPages":    3,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			expectedOutput: footer,
		},
	}

	baseHeaderCases = []templateTestCase{
		{
			desc: "Header Test /test/",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard1,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			expectedOutput: headBeginning +
				`<title>/test/-Testing board</title>` +
				`<link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
				`<link rel="shortcut icon"href="/favicon.png">` +
				`<script type="text/javascript"src="/js/consts.js"></script>` +
				`<script type="text/javascript"src="/js/gochan.js"defer></script>` +
				`</head><body><div id="topbar"><div class="topbar-section">` +
				`<a href="/"class="topbar-item">home</a></div>` +
				`<div class="topbar-section"><a href="/test/"class="topbar-item"title="Testing board">/test/</a>` +
				`<a href="/test2/" class="topbar-item" title="Testing board#2">/test2/</a></div></div>` +
				`<div id="content">`,
		},
		{
			desc: "Header Test /sup/",
			data: map[string]any{
				"boardConfig": simpleBoardConfig,
				"board":       simpleBoard2,
				"numPages":    1,
				"sections": []gcsql.Section{
					{ID: 1},
				},
			},
			expectedOutput: headBeginning +
				`<title>/sup/-Gochan Support board</title>` +
				`<link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
				`<link rel="shortcut icon"href="/favicon.png">` +
				`<script type="text/javascript"src="/js/consts.js"></script>` +
				`<script type="text/javascript"src="/js/gochan.js"defer></script>` +
				`</head><body><div id="topbar"><div class="topbar-section">` +
				`<a href="/"class="topbar-item">home</a></div>` +
				`<div class="topbar-section"><a href="/test/"class="topbar-item"title="Testing board">/test/</a>` +
				`<a href="/test2/" class="topbar-item" title="Testing board#2">/test2/</a></div></div>` +
				`<div id="content">`,
		},
		{
			desc: "Perma Ban Header Test",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						IsActive:  true,
						Permanent: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: `<!DOCTYPE html><html lang="en"><head>` +
				`<meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">` +
				`<title>YOU'RE PERMABANNED,&nbsp;IDIOT!</title><link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/><link rel="shortcut icon"href="/favicon.png">` +
				`<script type="text/javascript"src="/js/consts.js"></script><script type="text/javascript"src="/js/gochan.js"defer></script>` +
				`</head><body><div id="topbar"><div class="topbar-section"><a href="/"class="topbar-item">home</a></div></div><div id="content">`,
		},
		{
			desc: "Appealable Perma Ban Header Test",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						Permanent: true,
						CanAppeal: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: `<!DOCTYPE html><html lang="en"><head>` +
				`<meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">` +
				`<title>YOU ARE BANNED:(</title><link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/><link rel="shortcut icon"href="/favicon.png">` +
				`<script type="text/javascript"src="/js/consts.js"></script><script type="text/javascript"src="/js/gochan.js"defer></script>` +
				`</head><body><div id="topbar"><div class="topbar-section"><a href="/"class="topbar-item">home</a></div></div><div id="content">`,
		},
		{
			desc: "Appealable Temp Ban Header Test",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						CanAppeal: true,
						StaffID:   1,
						Message:   "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: `<!DOCTYPE html><html lang="en">` +
				`<head><meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">` +
				`<title>YOU ARE BANNED:(</title><link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/><link rel="shortcut icon"href="/favicon.png">` +
				`<script type="text/javascript"src="/js/consts.js"></script><script type="text/javascript"src="/js/gochan.js"defer></script>` +
				`</head><body><div id="topbar"><div class="topbar-section"><a href="/"class="topbar-item">home</a></div></div><div id="content">`,
		},
		{
			desc: "Unappealable Temp Ban Header Test",
			data: map[string]any{
				"ban": &gcsql.IPBan{
					RangeStart: "192.168.56.0",
					RangeEnd:   "192.168.56.255",
					IPBanBase: gcsql.IPBanBase{
						StaffID: 1,
						Message: "ban message goes here",
					},
				},
				"ip":         "192.168.56.1",
				"siteConfig": testingSiteConfig,
				"systemCritical": config.SystemCriticalConfig{
					WebRoot: "/",
				},
				"boardConfig": config.BoardConfig{
					DefaultStyle: "pipes.css",
				},
			},
			expectedOutput: `<!DOCTYPE html><html lang="en"><head>` +
				`<meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">` +
				`<title>YOU ARE BANNED:(</title><link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/><link rel="shortcut icon"href="/favicon.png">` +
				`<script type="text/javascript"src="/js/consts.js"></script><script type="text/javascript"src="/js/gochan.js"defer></script>` +
				`</head><body><div id="topbar"><div class="topbar-section"><a href="/"class="topbar-item">home</a></div></div><div id="content">`,
		},
	}
)

const (
	boardPageHeaderBase = `<!DOCTYPE html><html lang="en"><head>` +
		`<meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">` +
		`<title>/test/-Testing board</title>` +
		`<link rel="stylesheet"href="/css/global.css"/><link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
		`<link rel="shortcut icon"href="/favicon.png">` +
		`<script type="text/javascript"src="/js/consts.js"></script><script type="text/javascript"src="/js/gochan.js"defer></script></head>` +
		`<body><div id="topbar"><div class="topbar-section"><a href="/"class="topbar-item">home</a></div>` +
		`<div class="topbar-section"><a href="/test/"class="topbar-item"title="Testing board">/test/</a><a href="/test2/" class="topbar-item" title="Testing board#2">/test2/</a></div></div>` +
		`<div id="content"><header><h1 id="board-title">/test/-Testing board</h1><div id="board-subtitle">Board for testing<br/><a href="/test/catalog.html">Catalog</a> | <a href="#footer">Bottom</a></div></header><hr />` +
		`<div id="postbox-area"><form id="postform"name="postform"action="/post"method="POST"enctype="multipart/form-data">` +
		`<input type="hidden"name="threadid"value="0"/><input type="hidden"name="boardid"value="1"/>` +
		`<table id="postbox-static"><tr><th class="postblock">Name</th><td><input type="text" name="postname" maxlength="100" size="25" /></td></tr>` +
		`<tr><th class="postblock">Email</th><td><input type="text" name="postemail" maxlength="100" size="25" /></td></tr>` +
		`<tr><th class="postblock">Subject</th><td><input type="text"name="postsubject"size="25"maxlength="100"><input type="text"name="username"style="display:none"/><input type="submit"value="Post"/></td></tr>` +
		`<tr><th class="postblock">Message</th><td><textarea rows="5" cols="35" name="postmsg" id="postmsg"></textarea></td></tr>` +
		`<tr><th class="postblock">File</th><td><input name="imagefile" type="file" accept="image/jpeg,image/png,image/gif,video/webm,video/mp4">` +
		`<input type="checkbox" id="spoiler" name="spoiler"/><label for="spoiler">Spoiler</label></td></tr>` +
		`<tr id="threadoptions"style="display: none;"><th class="postblock">Options</th><td></td></tr>` +
		`<tr><th class="postblock">Password</th><td><input type="password" id="postpassword" name="postpassword" size="14" />(for post/file deletion)</td></tr></table>` +
		`<input type="password" name="dummy2" style="display:none"/></form></div><hr />`
)

type templateTestCase struct {
	desc           string
	data           any
	expectsError   bool
	expectedOutput string
}

func (tC *templateTestCase) Run(t *testing.T, templateName string) {
	buf := new(bytes.Buffer)
	err := serverutil.MinifyTemplate(templateName, tC.data, buf, "text/javascript")
	if tC.expectsError {
		assert.Error(t, err)
	} else {
		if !assert.NoError(t, err) {
			// var allStaff []gcsql.Staff

			return
		}
		assert.Equal(t, tC.expectedOutput, buf.String())
	}
}
