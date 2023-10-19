local events = require('events')
local gcsql = require("gcsql")

events.register_event({"db-initialized"}, function(tr)
	print("Testing SELECT query from Lua plugin")
	rows, err = gcsql.query_rows("SELECT id, dir, title FROM DBPREFIXboards where id > ?", {1})
	if(err ~= nil) then
		print(err:Error())
		return
	end

	print("Boards (id > 1):")
	while rows:Next() do
		rows_table = {}
		err = gcsql.scan_rows(rows, rows_table)
		if(err ~= nil) then
			print(err:Error())
			rows:Close()
			return
		end
		print(string.format("rows_table.id: %#v, rows_table.dir: %#v, rows_table.title = %#v",
			rows_table.id, rows_table.dir, rows_table.title))
	end
	rows:Close()

	print("Testing SELECT COUNT(*) query from Lua plugin")
	rows, err = gcsql.query_rows("SELECT COUNT(*) FROM DBPREFIXstaff WHERE id > ?", {1})
	if(err ~= nil) then
		print(err:Error())
		return
	end
	while rows:Next() do
		rows_table = {}
		err = gcsql.scan_rows(rows, rows_table)
		if(err ~= nil) then
			print(err:Error())
			rows:Close()
			return
		end
		print(string.format("Result: %d", rows_table["COUNT(*)"]))
	end
	rows:Close()
end)
