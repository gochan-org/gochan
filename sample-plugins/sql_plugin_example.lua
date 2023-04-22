event_register({"db-initialized"}, function(tr)
	print("Testing SELECT query from Lua plugin")
	rows, err = db_query("SELECT message_raw FROM DBPREFIXposts where id = ?", {28})
	if(err ~= nil) then
		print(err.Error(err))
		return
	end

	print("rows.Next():")
	while rows.Next(rows) do
		message_raw = "This should be different after rows.Scan"
		rows_table = {}
		db_scan_rows(rows, rows_table)
		print("Message: " .. message_raw)
		for v in rows_table do
			print(string.format("%q", v))
		end
	end
	rows.Close(rows)
	print("Done")
end)
