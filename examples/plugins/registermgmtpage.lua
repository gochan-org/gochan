-- testing manage page registering from Lua plugins
local strings = require("strings")
local manage = require("manage")

manage.register_manage_page("mgmtplugintest",
	"Staff Plugin Testing",
	3, 1,
	function(writer, request, staff, wantsJSON, infoEv, errEv)
		out = string.format("Hello %s from Lua!<br/>'param' url parameter value: %q", staff.Username, request.FormValue(request,"param"))
		return out, ""
	end
)


-- testing template parsing from Lua plugins
manage.register_manage_page("templateplugintest",
	"Template Plugin Testing",
	3, 0,
	function(writer, request, staff, wantsJSON, infoEv, errEv)
		local tmpl, err = parse_template("parse_template_test",
			[[<b>Staff: </b> {{.staff.Username}}<br/>
			This manage page rendered from a template provided by a Lua plugin]])
		if(err ~= nil) then
			print(err:Error())
			return "", err:Error()
		end
		
		buf = strings.new_builder()
		err = minify_template(tmpl, {
			staff = staff
		}, buf, "text/html")

		return buf:string(), err
	end
)