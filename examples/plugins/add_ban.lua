local events = require("events")
local log = require("gclog")
local manage = require("manage")

events.register_event({"message-pre-format"}, function(tr, post, req)
	if(post.MessageRaw == "ban me pls") then

		log.warn_log()
			:Str("IP", post.IP)
			:Msg("Banning post from Lua event")
		err = manage.ban_ip(post.IP, nil, "banned from Lua plugin", "admin", {
			board = "test"
		})
		if(err ~= nil) then
			return err:Error()
		end
	end
end)