-- a simple demonstration of using the event system from a Lua plugin to modify an incoming upload
event_register({"incoming-upload"}, function(tr, upload)
	print("Received upload, making the original filename upper case")
	before = upload.OriginalFilename
	upload.OriginalFilename = string.upper(upload.OriginalFilename)
	print(string.format("Before: %q, after: %q", before, upload.OriginalFilename))
end)