event_register({"incoming-upload"}, function(tr, upload)
	rows, err = db_query("SELECT COUNT(*) FROM DBPREFIXfiles WHERE original_filename = ?", {upload.OriginalFilename})
	if(err ~= nil) then
		return err:Error()
	end
	while rows:Next() do
		rows_table = {}
		err = db_scan_rows(rows, rows_table)
		if(err ~= nil) then
			rows:Close()
			return err:Error()
		end
		if(rows_table["COUNT(*)"] > 0) then
			rows:Close()
			return "a file with that filename has already been uploaded"
		end
	end
end)