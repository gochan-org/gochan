export function getCookie(name, defaultVal) {
	let val = defaultVal;
	let cookieArr = document.cookie.split("; ");

	for(const cookie of cookieArr) {
		let pair = cookie.split("=");
		if(pair[0] != name) continue;
		try {
			val = decodeURIComponent(pair[1]);
		} catch(err) {
			return defaultVal;
		}
	}
	return val;
}

// gets cookies ready to be used elsewhere
export function initCookies() {
	$("input[name=postname]").val(getCookie("name", ""));
	$("input[name=postemail]").val(getCookie("email", ""));
	$("input[name=postpassword]").val(getCookie("password", ""));
	$("input[name=delete-password]").val(getCookie("password", ""));
}

export function setCookie(name, value, expires) {
	let expiresStr = "";
	if(expires) {
		expiresStr = ";expires="
		let d = new Date();
		d.setTime(d.getTime() + 1000*60*60*24*expires)
		expiresStr += d.toUTCString();
	}
	document.cookie = `${name}=${escape(value)}${expiresStr};path=${webroot};sameSite=strict`;
}