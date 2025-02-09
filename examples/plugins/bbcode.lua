local bbcode = require("bbcode")

bbcode.set_tag("rcv", function(node)
	return {name="span", attrs={class="rcv"}}
end)