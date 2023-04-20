# Events
This is a list of events that gochan may trigger at some point, that can be used in the plugin system.

- **db-connected**
	- Triggered after gochan successfully connects to the database but before it is checked and initialized (db version checking, provisisioning, etc)

- **db-initialized**
	- Triggered after the database is successfully initialized (db version checking, provisioning, etc)

- **incoming-upload**
	- Triggered by the `gcsql` package when an upload is attached to a post. It is triggered before the upload is entered in the database

- **shutdown**
	- Triggered when gochan is about to shut down, in `main()` as a deferred call

- **startup**
	- Triggered when gochan first starts after its plugin system is initialized. This is (or at least should be) only triggered once.

- **upload-saved**
	- Triggered by the `posting` package when an upload is saved to the disk but before thumbnails are generated.