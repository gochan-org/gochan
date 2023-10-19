local events = require("events")
local gcsql = require("gcsql")

events.register_event({"incoming-upload"}, function(tr, upload)
	rows, err = gcsql.query_rows("SELECT COUNT(*) FROM DBPREFIXfiles WHERE original_filename = ?", {upload.OriginalFilename})
	if(err ~= nil) then
		return err:Error()
	end
	while rows:Next() do
		rows_table = {}
		err = gcsql.scan_rows(rows, rows_table)
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