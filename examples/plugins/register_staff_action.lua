local manage = require("manage")

manage.register_staff_action({
	id = "register-staff-action-lua/:param1",
	title = "Registered Staff Action (Lua)",
	permissions = "janitor",
	json = "sometimes",
	callback = function(writer, request, staff, wants_json, logger)
		-- access at http://<site>/manage/register-staff-action-lua/<some_param>
		-- or http://<site>/manage/register-staff-action-lua/<some_param>?json=1
		local params = manage.get_action_request_params(request)
		local param1, ok = params:Get("param1")
		if wants_json then
			return {
				message = "Hello from Lua!",
				method = request.Method,
				staff = staff.Username,
				wants_json = wants_json,
				param1 = param1,
				param_exists = ok
			}
		end
		out = string.format([[Hello from Lua!<br/>
Request method: %q<br/>
Staff user: %s<br/>
Wants JSON: %s<br/>
Request param: %s<br/>
Param exists: %s<br/>
]],
request.Method, staff.Username, wants_json, param1, ok)
		return out
	end
}, {
	"GET", "POST", "PUT"
})