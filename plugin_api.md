# Constants
- **_GOCHAN_VERSION**
	- The version string of the running Gochan server

# Functions
- **error_log([error_message string])**
	- Creates and returns a zerolog [Event](https://pkg.go.dev/github.com/rs/zerolog) object for the error log. If a string is used as the argument, it is used as the error message.

- **info_log()**
	- Creates and returns a zerolog [Event](https://pkg.go.dev/github.com/rs/zerolog) object with an info level.

- **warn_log()**
	- Creates and returns a zerolog [Event](https://pkg.go.dev/github.com/rs/zerolog) object with a warning level.

- **register_upload_handler(ext string, function(upload, post, board, filePath, thumbPath, catalogThumbPath, infoEv, accessEv, errEv))**
	- Registers a function to be called for handling uploaded files with the given extension. See [pdf_thumbnail.lua](./sample-plugins/pdf_thumbnail.lua) for a usage example.

- **get_thumbnail_ext(upload_ext string)**
	- Returns the configured (or built-in) thumbnail file extension to be used for the given upload extension

- **set_thumbnail_ext(upload_ext string, thumbnail_ext string)**
	- Sets the thumbnail extension to be used for the given upload extension

- **system_critical_config()**
	- Returns the [SystemCriticalConfig](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/config#SystemCriticalConfig)

- **site_config()**
	- Returns the [SiteConfig](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/config#SiteConfig)

- **board_config(board string)**
	- Returns the [BoardConfig](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/config#BoardConfig) for the given board, or the default BoardConfig if `board` is an empty string

- **db_query(query string)**
	- Returns a [Rows](https://pkg.go.dev/database/sql#Rows) object for the given SQL query and an error if any occured, or nil if there were no errors.

- **db_scan_rows(rows, scan_table)**
	- scans the value of the current row into `scan_table` and returns an error if any occured, or nil if there were no errors.

- **event_register(events_table, handler_func)**
	- Registers `handler_func` for the events in `events_table`. If any arguments are passed to the event when it is triggered, it will be sent to `handler_func`.

- **event_trigger(event_name string, data...)**
	- Triggers the event registered to `event_name` and passes `data` (if set) to the event handler.

- **register_manage_page(action string, title string, perms int, wants_json int, handler func(writer, request, staff, wants_json, info_ev, err_ev))**
	- Registers the manage page accessible at /manage/`action` to be handled by `handler`. See [manage.RegisterManagePage](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/manage#RegisterManagePage) for info on how `handler` should be used, or [registermgmtpage.lua](./sample-plugins/registermgmtpage.lua) for an example

- **load_template(files...)**
	- Calls [gctemplates.LoadTemplate](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/gctemplates#LoadTemplate) using the given `files` and returns a [Template](https://pkg.go.dev/html/template#Template) and an error object (or nil if there were no errors).

- **parse_template(template_name string, template_data string)**
	- Calls [gctemplates.ParseTemplate](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/gctemplates#ParseTemplate) with the given template name and Go template data, and returns a [Template](https://pkg.go.dev/html/template#Template) and an error object (or nil if there were no errors).

- **minify_template(template, data_table, writer, media_type)**
	- Calls [serverutil.MinifyTemplate](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/server/serverutil#MinifyTemplate) with the given `template` object, `data_table` (as variables passed to the template), `writer`, and `media_type`. See [registermgmtpage.lua](./sample-plugins/registermgmtpage.lua) for an example


# Events
This is a list of events that gochan may trigger at some point and can be used in the plugin system.

- **db-connected**
	- Triggered after gochan successfully connects to the database but before it is checked and initialized (db version checking, provisisioning, etc)

- **db-initialized**
	- Triggered after the database is successfully initialized (db version checking, provisioning, etc)

- **incoming-upload**
	- Triggered by the `gcsql` package when an upload is attached to a post. It is triggered before the upload is entered in the database

- **message-pre-format**
	- Triggered when an incoming post or post edit is about to be formatted

- **shutdown**
	- Triggered when gochan is about to shut down, in `main()` as a deferred call

- **startup**
	- Triggered when gochan first starts after its plugin system is initialized. This is (or at least should be) only triggered once.

- **upload-saved**
	- Triggered by the `posting` package when an upload is saved to the disk but before thumbnails are generated.