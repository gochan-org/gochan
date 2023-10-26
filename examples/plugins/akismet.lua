local config = require("config")
local events = require("events")
local http = require("http")
local log = require("gclog")
local url = require("url")

local check_key_url = "https://rest.akismet.com/1.1/verify-key"

local base_headers = {}
base_headers["User-Agent"] = "gochan/3.8 | Akismet/0.1"
base_headers["Content-Type"] = "application/x-www-form-urlencoded"

local key = "b78cd8a0ba8c"

local function check_api_key()
	local resp, err = http.request("POST", check_key_url, {
		body = "blog=" .. url.query_escape("http://" .. config.system_critical_config().SiteDomain) ..
				"&key=" .. key,
		headers = base_headers
	})
	if(err ~= nil) then
		log.error_log(err):Str("url", check_key_url):Send()
		return err
	end
	if(resp.body ~= "valid") then
		log.error_log():Str("key", key):Msg("invalid Akismet API key or request")
		return "Invalid API key"
	end
	return nil
end

local function check_akismet(post, user_agent, referrer)
	local comment_type = "reply"
	if post.IsTopPost then
		comment_type = "forum-post"
	end

	local form = "blog=" .. url.query_escape("http://" .. config.system_critical_config().SiteDomain) ..
		"&user_ip=" .. url.query_escape(post.IP) ..
		"&user_agent=" .. url.query_escape(user_agent) ..
		"&referrer=" .. url.query_escape(referrer) ..
		"&comment_type=" .. comment_type ..
		"&comment_author=" .. url.query_escape(post.Name) ..
		"&comment_author_email=" .. url.query_escape(post.Email) ..
		"&comment_content=" .. url.query_escape(post.MessageRaw)
	local resp, err = http.request("POST", "https://" .. key .. ".rest.akismet.com/1.1/comment-check", {
		body = form,
		headers = base_headers
	})
	if(err ~= nil) then
		log.error_log(err):Caller()
			:Str("subject", "akismet")
			:Msg("Unable to check Akismet")
		return err
	end
	local body = resp.body

	local warn_ev = log.warn_log()
		:Str("akismet", body)
		:Str("name", post.Name)
		:Str("IP", post.IP)
	if(body == "true") then
		warn_ev:Msg("Blocked spam message")
		return "Your post looks like spam"
	elseif(body == "invalid") then
		warn_ev:Msg("Invalid Akismet request")
		return "Unable to check post for spam (invalid request)"
	end
	warn_ev:Discard()
	return nil
end

local err = check_api_key()
if(err ~= nil) then
	error(err)
end

events.register_event({"message-pre-format"}, function(tr, post, req)
	local err = check_akismet(post, req.Header:Get("User-Agent"), req:Referer())
	return err
end)