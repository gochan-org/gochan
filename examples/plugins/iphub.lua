local events = require("events")
local http = require("http")
local json = require("json")
local log = require("gclog")
local url = require("url")

local url_prefix = "http://v2.api.iphub.info/ip/"
local key = ""
local max_block = 0
-- https://iphub.info/api

local function check_iphub(ip)
	if(key == "") then
		return nil
	end
	local headers = {}
	headers["X-Key"] = key
	local resp, err = http.get(url_prefix .. ip, {
		headers = headers
	})
	if(err ~= nil) then
		return err
	end
	local json_decoded = json.decode(resp.body)
	local err = json_decoded["error"]
	if(err ~= nil) then
		return err
	end
	local block = tonumber(json_decoded["block"])
	if(block > max_block) then
		log.error_log():
			Str("IP", ip):
			Int("block", block):
			Msg("IP determined as high-risk according to IPHub")
		return "Your post looks like spam"
	end
	
	return nil
end


local iphf = assert(io.open("/etc/gochan/iphub_key.txt", "r"))
key = assert(iphf:read("*a")):gsub("%s+", "")
iphf:close()

events.register_event({"message-pre-format"}, function(tr, post, req)
	local ip = post.IP
	return check_iphub(ip)
end)