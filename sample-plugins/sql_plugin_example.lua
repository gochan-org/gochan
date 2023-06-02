event_register({"db-initialized"}, function(tr)
	print("Testing SELECT query from Lua plugin")
	rows, err = db_query("SELECT id, dir, title FROM DBPREFIXboards where id > ?", {1})
	if(err ~= nil) then
		print(err.Error(err))
		return
	end

	print("Boards (id > 1):")
	while rows.Next(rows) do
		rows_table = {}
		err = db_scan_rows(rows, rows_table)
		if(err ~= nil) then
			print(err.Error(err))
			return
		end
		print(string.format("rows_table.id: %#v, rows_table.dir: %#v, rows_table.title = %#v",
			rows_table.id, rows_table.dir, rows_table.title))
	end
	rows.Close(rows)
end)
