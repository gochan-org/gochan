local events = require("events")
local http = require("http")
local json = require("json")
local log = require("gclog")

local url_prefix = "https://api.vpndetection.io/"
-- refuse privacy relays (iCloud Private Relay and the like) too
local block_relays = false
-- https://vpndetection.io/api, no API key needed

local function check_vpndetection(ip)
	local resp, err = http.get(url_prefix .. ip)
	if(err ~= nil) then
		log.error_log(err):
			Str("IP", ip):
			Msg("VPNDetection lookup failed")
		return nil
	end
	if(resp.status_code ~= 200) then
		log.warn_log():
			Str("IP", ip):
			Int("status", resp.status_code):
			Msg("VPNDetection returned an error")
		return nil
	end
	local data = json.decode(resp.body)
	if(data == nil) then
		return nil
	end
	if(data["is_vpn"] or data["is_tor"] or (block_relays and data["is_relay"])) then
		log.warn_log():
			Str("IP", ip):
			Bool("vpn", data["is_vpn"] == true):
			Bool("tor", data["is_tor"] == true):
			Bool("relay", data["is_relay"] == true):
			Msg("IP determined as a VPN, Tor exit or relay according to VPNDetection")
		return "Posting through a VPN or Tor is not allowed"
	end

	return nil
end

events.register_event({"message-pre-format"}, function(tr, post, req)
	return check_vpndetection(post.IP)
end)
