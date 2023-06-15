local string = require("string")

local recognized_tlds = {"com", "net", "org", "edu", "gov", "us", "uk"}

local function is_new_poster(ip)
	rows, err = db_query("SELECT COUNT(*) FROM DBPREFIXposts WHERE ip = ?", {ip})
	if(err ~= nil) then
		return true, err
	end

	is_new = true
	while rows:Next() do
		rows_table = {}
		err = db_scan_rows(rows, rows_table)
		if(err ~= nil) then
			rows:Close()
			return true, err
		end
		if(rows_table["COUNT(*)"] > 0) then
			is_new = false
			break 
		end
	end
	rows:Close()
	return is_new
end


event_register({"message-pre-format"}, function(tr, post)
	is_new, err = is_new_poster(post.IP)
	if(err ~= nil) then
		error_log(err:Error())
			:Str("lua", "check_links.lua")
			:Str("event", tr)
			:Send()
		return err:Error()
	end
	if(is_new == false) then
		-- Not a new poster, skip TLD check
		return
	end

	for tld in string.gmatch(post.MessageRaw, "%a+://%w+.(%w+)") do
		found = false
		for _, recognized in pairs(recognized_tlds) do
			if(tld == recognized) then
				found = true
				break
			end
		end
		if(found == false) then
			return "post contains one or more untrusted links"
		end
	end
end)
