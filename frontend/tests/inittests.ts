// jQuery can't be imported in a non-browser environment (jest.config.ts), so we import it here after jsdom has been loaded
import jquery from "jquery";

(global as {
	jQuery?: typeof jquery;
}).jQuery = jquery;

global.styles = [
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
(global as unknown as {simpleHTML:string}).simpleHTML = `<!DOCTYPE html>
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