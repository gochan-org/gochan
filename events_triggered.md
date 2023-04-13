# Events
This is a list of events that gochan may trigger at some point, that can be used in the plugin system.

- **incoming-upload**
	- Triggered by the `gcsql` package when an upload is attached to a post. It is triggered before the upload is entered in the database
- **upload-saved**
	- Triggered by the `posting` package when an upload is saved to the disk but before thumbnails are generated.