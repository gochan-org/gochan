event_register({"db-initialized"}, function(tr)
	print("Testing SELECT query from Lua plugin")
	rows, err = db_query("SELECT id, username FROM DBPREFIXstaff where id = ?", {1})
	if(err ~= nil) then
		print(err.Error(err))
		return
	end

	print("rows.Next():")
	while rows.Next(rows) do
		rows_table = {}
		err = db_scan_rows(rows, rows_table)
		if(err ~= nil) then
			print(err.Error(err))
			return
		end
		print(string.format("rows_table.id: %#v, rorws_table.username: %#v", rows_table.id, rows_table.username))
	end
	rows.Close(rows)
	print("Done")
end)
