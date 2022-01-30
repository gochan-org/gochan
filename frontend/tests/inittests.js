import { jest } from "@jest/globals";

// mock variables the browser would normally get from {webroot}js/consts.js
global.styles=[
	{Name: "Pipes", Filename: "pipes.css"},
	{Name: "BunkerChan", Filename: "bunkerchan.css"},
	{Name: "Burichan", Filename: "burichan.css"},
	{Name: "Clear", Filename: "clear.css"},
	{Name: "Dark", Filename: "dark.css"},
	{Name: "Photon", Filename: "photon.css"},
	{Name: "Yotsuba", Filename: "yotsuba.css"},
	{Name: "Yotsuba B", Filename: "yotsubab.css"},
	{Name: "Windows 9x", Filename: "win9x.css"}
];

global.defaultStyle = "pipes.css";
global.webroot = "/";
global.serverTZ = -8;

global.simpleHTML = `<!DOCTYPE html>
<html>
<body>
<form id="postform">
<input name="postname" id="postname" type="text"/>
<input name="postemail" id="postemail" type="text"/>
<input name="postpassword" type="password" />
<input name="delete-password" type="password" />
<textarea id="postmsg" name="postmsg"></textarea>
</form>
</body>
</html>`;