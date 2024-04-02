package gctemplates_test

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
			},
			expectedOutput: `var styles=[{Name:"Pipes",Filename:"pipes.css"},{Name:"Yotsuba A",Filename:"yotsuba.css"}];var defaultStyle="pipes.css";var webroot="/";var serverTZ=-1;`,
		},
		{
			desc: "empty values",
			data: map[string]any{
				"defaultStyle": "",
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `var styles=[];var defaultStyle="";var webroot="";var serverTZ=0;`,
		},
		{
			desc: "escaped string",
			data: map[string]any{
				"defaultStyle": `"a\a"`,
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `var styles=[];var defaultStyle="\&#34;a\\a\&#34;";var webroot="";var serverTZ=0;`,
		},
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
