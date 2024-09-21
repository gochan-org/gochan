package templatetests_test

import (
	"bytes"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
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
			expectedOutput: `<!DOCTYPE html><html><head><title>Banned</title>` +
				`<link rel="shortcut icon"href="/favicon.png">` +
				`<link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
				`<script type="text/javascript"src="/js/consts.js"></script>` +
				`<script type="text/javascript"src="/js/gochan.js"></script></head>` +
				`<body><div id="top-pane"><span id="site-title">Gochan</span><br /><span id="site-slogan">Gochan test</span></div><br />` +
				`<div class="section-block"style="margin: 0px 26px 0px 24px">` +
				`<div class="section-title-block"><span class="section-title"><b>YOU ARE BANNED&nbsp;:(</b></span></div>` +
				`<div class="section-body"style="padding-top:8px"><div id="ban-info"style="float:left">You are banned from posting on<b>all boards</b>for the following reason:<br/><br/>` +
				`<b>ban message goes here</b><br/><br/>` +
				`Your ban was placed on Mon,January 01,0001 12:00:00 AM and will<b>not expire</b>.<br />` +
				`Your IP address is<b>192.168.56.1</b>.<br /><br/>You may appeal this ban:<br/>` +
				`<form id="appeal-form"action="/post"method="POST">` +
				`<input type="hidden"name="board"value=""><input type="hidden"name="banid"value="0">` +
				`<textarea rows="4"cols="48"name="appealmsg"id="postmsg"placeholder="Appeal message"></textarea><br />` +
				`<input type="submit"name="doappeal"value="Submit"/><br/></form></div></div></div>` +
				`<div id="footer">Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 3.10.1</a><br /></div></div></body></html>`,
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
			expectedOutput: `<!DOCTYPE html><html><head><title>Banned</title>` +
				`<link rel="shortcut icon"href="/favicon.png">` +
				`<link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
				`<script type="text/javascript"src="/js/consts.js"></script>` +
				`<script type="text/javascript"src="/js/gochan.js"></script></head>` +
				`<body><div id="top-pane"><span id="site-title">Gochan</span><br /><span id="site-slogan">Gochan test</span></div><br />` +
				`<div class="section-block"style="margin: 0px 26px 0px 24px"><div class="section-title-block">` +
				`<span class="section-title"><b>YOUR'E PERMABANNED,IDIOT!</b></span></div>` +
				`<div class="section-body"style="padding-top:8px"><div id="ban-info"style="float:left">You are banned from posting on<b>all boards</b>for the following reason:<br/><br/>` +
				`<b>ban message goes here</b><br/><br/>Your ban was placed on Mon,January 01,0001 12:00:00 AM and will<b>not expire</b>.<br />` +
				`Your IP address is<b>192.168.56.1</b>.<br /><br/>You may&nbsp;<b>not</b> appeal this ban.<br /></div>` +
				`<img id="banpage-image"src="/permabanned.jpg"style="float:right; margin: 4px 8px 8px 4px"/><br/>` +
				`<audio id="jack"preload="auto"autobuffer loop><source src="/static/hittheroad.ogg"/><source src="/static/hittheroad.wav"/><source src="/static/hittheroad.mp3"/></audio>` +
				`<script type="text/javascript">document.getElementById("jack").play();</script></div></div>` +
				`<div id="footer">Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 3.10.1</a><br /></div></div></body></html>`,
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
			expectedOutput: `<!DOCTYPE html><html><head><title>Banned</title>` +
				`<link rel="shortcut icon"href="/favicon.png">` +
				`<link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
				`<script type="text/javascript"src="/js/consts.js"></script>` +
				`<script type="text/javascript"src="/js/gochan.js"></script></head>` +
				`<body><div id="top-pane"><span id="site-title">Gochan</span><br /><span id="site-slogan">Gochan test</span></div><br />` +
				`<div class="section-block"style="margin: 0px 26px 0px 24px">` +
				`<div class="section-title-block"><span class="section-title"><b>YOU ARE BANNED&nbsp;:(</b></span></div>` +
				`<div class="section-body"style="padding-top:8px"><div id="ban-info"style="float:left">You are banned from posting on<b>all boards</b>for the following reason:<br/><br/>` +
				`<b>ban message goes here</b><br/><br/>` +
				`Your ban was placed on Mon,January 01,0001 12:00:00 AM and will expire on&nbsp;<b>Mon,January 01,0001 12:00:00 AM</b>.<br />` +
				`Your IP address is<b>192.168.56.1</b>.<br /><br/>You may appeal this ban:<br/>` +
				`<form id="appeal-form"action="/post"method="POST">` +
				`<input type="hidden"name="board"value=""><input type="hidden"name="banid"value="0">` +
				`<textarea rows="4"cols="48"name="appealmsg"id="postmsg"placeholder="Appeal message"></textarea><br />` +
				`<input type="submit"name="doappeal"value="Submit"/><br/></form></div></div></div>` +
				`<div id="footer">Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 3.10.1</a><br /></div></div></body></html>`,
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
			expectedOutput: `<!DOCTYPE html><html><head><title>Banned</title>` +
				`<link rel="shortcut icon"href="/favicon.png">` +
				`<link rel="stylesheet"href="/css/global.css"/>` +
				`<link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
				`<script type="text/javascript"src="/js/consts.js"></script>` +
				`<script type="text/javascript"src="/js/gochan.js"></script></head>` +
				`<body><div id="top-pane"><span id="site-title">Gochan</span><br /><span id="site-slogan">Gochan test</span></div><br />` +
				`<div class="section-block"style="margin: 0px 26px 0px 24px">` +
				`<div class="section-title-block"><span class="section-title"><b>YOU ARE BANNED&nbsp;:(</b></span></div>` +
				`<div class="section-body"style="padding-top:8px"><div id="ban-info"style="float:left">You are banned from posting on<b>all boards</b>for the following reason:<br/><br/>` +
				`<b>ban message goes here</b><br/><br/>` +
				`Your ban was placed on Mon,January 01,0001 12:00:00 AM and will expire on&nbsp;<b>Mon,January 01,0001 12:00:00 AM</b>.<br />` +
				`Your IP address is<b>192.168.56.1</b>.<br /><br/>You may&nbsp;<b>not</b> appeal this ban.<br />` +
				`</div></div></div>` +
				`<div id="footer">Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 3.10.1</a><br /></div></div></body></html>`,
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
				`<form action="/util"method="POST"id="main-form"><div id="right-bottom-content"><div id="report-delbox"><input type="hidden"name="board"value="test"/><input type="hidden"name="boardid"value="1"/><label>[<input type="checkbox"name="fileonly"/>File only]</label> <input type="password" size="10" name="password" id="delete-password" /><input type="submit"name="delete_btn"value="Delete"onclick="return confirm('Are you sure you want to delete these posts?')"/><br/>Report reason:<input type="text"size="10"name="reason"id="reason"/><input type="submit"name="report_btn"value="Report"/><br/><input type="submit"name="edit_btn"value="Edit post"/>&nbsp;<input type="submit"name="move_btn"value="Move thread"/></div></div></form><div id="left-bottom-content"><a href="#">Scroll to top</a><br/><table id="pages"><tr><td>[<a href="/test/1.html">1</a>]</td></tr></table><span id="boardmenu-bottom">[<a href="/">home</a>]&nbsp;[]</span></div>` +
				`<div id="footer">Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 3.10.1</a><br /></div></div></body></html>`,
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
				`<form action="/util"method="POST"id="main-form"><div id="right-bottom-content"><div id="report-delbox"><input type="hidden"name="board"value="test"/><input type="hidden"name="boardid"value="1"/><label>[<input type="checkbox"name="fileonly"/>File only]</label> <input type="password" size="10" name="password" id="delete-password" /><input type="submit"name="delete_btn"value="Delete"onclick="return confirm('Are you sure you want to delete these posts?')"/><br/>Report reason:<input type="text"size="10"name="reason"id="reason"/><input type="submit"name="report_btn"value="Report"/><br/><input type="submit"name="edit_btn"value="Edit post"/>&nbsp;<input type="submit"name="move_btn"value="Move thread"/></div></div></form><div id="left-bottom-content"><a href="#">Scroll to top</a><br/><table id="pages"><tr><td>[<a href="/test/1.html">1</a>]</td></tr></table><span id="boardmenu-bottom">[<a href="/">home</a>]&nbsp;[]</span></div>` +
				`<div id="footer">Powered by<a href="http://github.com/gochan-org/gochan/">Gochan 3.10.1</a><br /></div></div></body></html>`,
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
			expectedOutput: `var styles=[{Name:"Pipes",Filename:"pipes.css"},{Name:"Yotsuba A",Filename:"yotsuba.css"}];var defaultStyle="pipes.css";var webroot="/";var serverTZ=-1;var fileTypes=[".ext",];`,
		},
		{
			desc: "empty values",
			data: map[string]any{
				"defaultStyle": "",
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `var styles=[];var defaultStyle="";var webroot="";var serverTZ=0;var fileTypes=[];`,
		},
		{
			desc: "escaped string",
			data: map[string]any{
				"defaultStyle": `"a\a"`,
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `var styles=[];var defaultStyle="\&#34;a\\a\&#34;";var webroot="";var serverTZ=0;var fileTypes=[];`,
		},
	}
)

const (
	boardPageHeaderBase = `<!DOCTYPE html><html><head>` +
		`<meta charset="UTF-8"><meta name="viewport"content="width=device-width, initial-scale=1.0">` +
		`<title>/test/-Testing board</title>` +
		`<link rel="stylesheet"href="/css/global.css"/><link id="theme"rel="stylesheet"href="/css/pipes.css"/>` +
		`<link rel="shortcut icon"href="/favicon.png">` +
		`<script type="text/javascript"src="/js/consts.js"></script><script type="text/javascript"src="/js/gochan.js"></script></head>` +
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
		`<tr id="threadoptions" style="display:none;"><th class="postblock">Options</th><td></td></tr>` +
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
			return
		}
		assert.Equal(t, tC.expectedOutput, buf.String())
	}
}
