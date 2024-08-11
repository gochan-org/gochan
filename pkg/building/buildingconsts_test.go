package building

const (
	expectedMinifiedJS   = `var styles=[{Name:"test1",Filename:"test1.css"},{Name:"test2",Filename:"test2.css"}];var defaultStyle="test1.css";var webroot="/chan";var serverTZ=8;var fileTypes=[];`
	expectedUnminifiedJS = `var styles = [{Name: "test1", Filename: "test1.css"},{Name: "test2", Filename: "test2.css"}];
var defaultStyle = "test1.css";
var webroot = "/chan";
var serverTZ = 8;
var fileTypes = [];`
	expectedMinifiedFront   = `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1"><title>Gochan</title><link rel=stylesheet href=/chan/css/global.css><link id=theme rel=stylesheet href=/chan/css/test1.css><link rel="shortcut icon" href=/chan/favicon.png><script src=/chan/js/consts.js></script><script src=/chan/js/gochan.js></script><div id=topbar><div class=topbar-section><a href=/chan/ class=topbar-item>home</a></div><div class=topbar-section><a href=/chan/test/ class=topbar-item title="Testing board">/test/</a><a href=/chan/test2/ class=topbar-item title="Testing board 2">/test2/</a></div></div><div id=content><div id=top-pane><span id=site-title>Gochan</span><br><span id=site-slogan></span></div><br><div id=frontpage><div class=section-block style="margin: 16px 64px 16px 64px;"><div class="section-body front-intro">Welcome to Gochan!</div></div><div class=section-block><div class=section-title-block><b>Boards</b></div><div class=section-body><ul style="float:left; list-style: none"><li style="text-align: center; font-weight: bold"><b><u>Main</u></b><li><a href=/chan/test/ title="Board for testing description">/test/</a> — Testing board<li><a href=/chan/test2/ title="Board for testing description 2">/test2/</a> — Testing board 2</ul></div></div><div class=section-block><div class=section-title-block><b>Recent Posts</b></div><div class=section-body><div id=recent-posts><div class=recent-post><a href=/chan/test/res/1.html#1 class=front-reply target=_blank><img src=/chan/test/thumb alt="post thumbnail"></a><br><br><a href=/chan/test/>/test/</a><hr>message_raw</div><div class=recent-post><a href=/chan/test/res/1.html#2 class=front-reply target=_blank><img src=/chan/test/thumb alt="post thumbnail"></a><br><br><a href=/chan/test/>/test/</a><hr>message_raw</div></div></div></div></div><div id=footer>Powered by <a href=http://github.com/gochan-org/gochan/>Gochan 3.11.0</a><br></div></div>`
	expectedUnminifiedFront = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Gochan</title>
	<link rel="stylesheet" href="/chan/css/global.css" />
	<link id="theme" rel="stylesheet" href="/chan/css/test1.css" />
	<link rel="shortcut icon" href="/chan/favicon.png"><script type="text/javascript" src="/chan/js/consts.js"></script>
	<script type="text/javascript" src="/chan/js/gochan.js"></script>
</head>
<body>
<div id="topbar">
	<div class="topbar-section"><a href="/chan/" class="topbar-item">home</a></div><div class="topbar-section"><a href="/chan/test/" class="topbar-item" title="Testing board">/test/</a><a href="/chan/test2/" class="topbar-item" title="Testing board 2">/test2/</a></div></div>

<div id="content">
	<div id="top-pane">
		<span id="site-title">Gochan</span><br />
		<span id="site-slogan"></span>
	</div><br />
	<div id="frontpage">
		<div class="section-block" style="margin: 16px 64px 16px 64px;">
			<div class="section-body front-intro">
				Welcome to Gochan!
			</div>
		</div>
		<div class="section-block">
			<div class="section-title-block"><b>Boards</b></div>
			<div class="section-body">
						<ul style="float:left; list-style: none">
						<li style="text-align: center; font-weight: bold"><b><u>Main</u></b></li>
						
							
								<li><a href="/chan/test/" title="Board for testing description">/test/</a> — Testing board</li>
							
						
							
								<li><a href="/chan/test2/" title="Board for testing description 2">/test2/</a> — Testing board 2</li>
							
						
						</ul>
					
				
			</div>
		</div>
		<div class="section-block">
			<div class="section-title-block"><b>Recent Posts</b></div>
			<div class="section-body">
				<div id="recent-posts">
					<div class="recent-post">
						<a href="/chan/test/res/1.html#1" class="front-reply" target="_blank"><img src="/chan/test/thumb" alt="post thumbnail"/></a><br />
						<br />
						<a href="/chan/test/">/test/</a><hr />
						message_raw
					</div>
					<div class="recent-post">
						<a href="/chan/test/res/1.html#2" class="front-reply" target="_blank"><img src="/chan/test/thumb" alt="post thumbnail"/></a><br />
						<br />
						<a href="/chan/test/">/test/</a><hr />
						message_raw
					</div>
				</div>
			</div>
		</div>
	</div>
<div id="footer">
	Powered by <a href="http://github.com/gochan-org/gochan/">Gochan 3.11.0</a><br />
</div>
</div>
</body>
</html>

`
)
