local geoip = require("geoip")
local log = require("gclog")

local geoip_header = "CF-IPCountry"

CFGeoIP = {}

function CFGeoIP:new()
	local t = setmetatable({}, {__index = CFGeoIP})
	return t
end

function CFGeoIP:init()
	log.info_log():Str("dbType", "cloudflare"):Msg("GeoIP initialized")
end

function CFGeoIP:close()
	return ">:("
end

function CFGeoIP.get_country(request, board, errEv)
	local abbr = request.Header:Get(geoip_header)
	local name, err = geoip.country_name(abbr)
	if(err ~= nil) then
		errEv:Err(err):Caller():Send()
		return nil, err
	end
	return {
		flag = abbr,
		name = name
	}, nil
end

-- local cf = CFGeoIP:new()
geoip.register_handler("cloudflare", CFGeoIP)