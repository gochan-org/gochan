# Constants
- **_GOCHAN_VERSION**
	- The version string of the running Gochan server

# Modules
The following are modules that can be loaded via `require("modulename")`. See [./examples/plugins/](./examples/plugins/) for usage examples.
## External modules
- [async](https://pkg.go.dev/github.com/CuberL/glua-async@v0.0.0-20190614102843-43f22221106d)
- [filepath](https://pkg.go.dev/github.com/vadv/gopher-lua-libs@v0.5.0/filepath)
- [json](https://pkg.go.dev/layeh.com/gopher-json@v0.0.0-20201124131017-552bb3c4c3bf)
- [strings](https://pkg.go.dev/github.com/vadv/gopher-lua-libs@v0.5.0/strings)

## bbcode
- **set_tag(tag string, handler [bbcode.TagCompilerFunc](https://pkg.go.dev/github.com/frustra/bbcode@v0.0.0-20201127003707-6ef347fbe1c8#TagCompilerFunc)))**
	- Registers a new BBCode function to handle the given tag. Struct-table fields are expected to use snake case (e.g., name instead of Name)

## config
- **config.system_critical_config()**
  - Returns the [SystemCriticalConfig](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/config#SystemCriticalConfig)
- **config.site_config()**
	- Returns the [SiteConfig](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/config#SiteConfig)
- **config.board_config(board string)**
	- Returns the [BoardConfig](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/config#BoardConfig) for the given board, or the default BoardConfig if `board` is an empty string

## events
- **events.register_event(events_table, handler_func)**
	- Registers `handler_func` for the events in `events_table`. If any arguments are passed to the event when it is triggered, it will be sent to `handler_func`.
- **events.trigger_event(event_name string, data...)**
	- Triggers the event registered to `event_name` and passes `data` (if set) to the event handler.

## gclog
- **gclog.info_log()**
	- Creates and returns a zerolog [Event](https://pkg.go.dev/github.com/rs/zerolog) object with an info level.
- **gclog.warn_log()**
	- Creates and returns a zerolog [Event](https://pkg.go.dev/github.com/rs/zerolog) object with a warning level.
- **gclog.error_log([error_message string])**
	- Creates and returns a zerolog [Event](https://pkg.go.dev/github.com/rs/zerolog) object for the error log. If a string is used as the argument, it is used as the error message.

## gcsql
- **gcsql.query_rows(query string, args...)**
	- Returns a [Rows](https://pkg.go.dev/database/sql#Rows) object for the given SQL query and an error if any occured, or nil if there were no errors. `args` if given will be used for a parameterized query.
- **gcsql.execute_sql(query string, args...)**
  - Executes the SQL string `query` with the optional `args` as parameters and returns a [Result](https://pkg.go.dev/database/sql#Result) object and an error (or nil if there were no errors)
- **gcsql.scan_rows(rows, scan_table)**
	- scans the value of the current row into `scan_table` and returns an error if any occured, or nil if there were no errors.

## gctemplates
- **gctemplates.load_template(files...)**
    - Loads the given file paths into a [Template](https://pkg.go.dev/html/template#Template), using the base filename as the name, and returns the template and an error if one occured.
- **gctemplates.parse_template(template_name string, template_data string)**
	- Calls [gctemplates.ParseTemplate](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/gctemplates#ParseTemplate) with the given template name and Go template data, and returns a [Template](https://pkg.go.dev/html/template#Template) and an error object (or nil if there were no errors).

## geoip
- **geoip.country_name(abbr string) (string, error)**
	- Returns the country name, given its abbreviation, and an error if any occured

- **geoip.register_handler(name string, handler table) error**
	- Calls [posting.RegisterGeoIPHandler](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/posting/geoip#RegisterGeoIPHandler) with the given handler info and returns an error if any occured. The table is expected to have the following fields/values:

Key         | Type | Explanation
------------|------|-------------
init        | func(options map[string]any) error | The function to initialize the GeoIP handler with options. If it needs no initialization, the function can return null
get_country | func(request http.Request, board string, errEv zerolog.Event) geoip.Country, error | The function to get the requesting IP's country, returning it and any errors that occured
close       | func() error | The function to close any network or file handles, if any were opened, returning an error if any occured


## manage
- **manage.ban_ip(ip string, duration string, reason string, staff string|int, options table)**
  - Bans the given IP for the given duration and gets other optional ban data from the `options` table below

Key           | Type             | Explanation
--------------|------------------|--------------
board         | string\|int\|nil | The board directory or ID that the IP will be banned from. If this is nil or omitted, it will be a global ban
post          | int              | The post ID
is_thread_ban | bool             | If true, the user will be able to post but unable to create threads
appeal_after  | string           | User can appeal after this duration. If unset, the user can appeal immediately.
appealable    | bool             | Sets whether or not the user can appeal the ban. If unset, the user is able to appeal.
staff_note    | string           | A private note attached to the ban that only staff can see

- **manage.register_manage_page(action string, title string, perms int, wants_json int, handler func(writer, request, staff, wants_json, info_ev, err_ev))**
	- Registers the manage page accessible at /manage/`action` to be handled by `handler`. See [manage.RegisterManagePage](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/manage#RegisterManagePage) for info on how `handler` should be used, or [registermgmtpage.lua](./examples/plugins/registermgmtpage.lua) for an example

## server
- **server.register_ext_headers(ext string, headers table) error**
	- Registers the file extension headers, allowing it to be recognized when serving static files. Each key in the table corresponds to the header name, and the values should be strings. The table can have custom/non-standard headers, but a Content-Type header is required.
	**Note**: If you have it set up so that Gochan is not serving static files (i.e., they are being handled by a reverse proxy or another server), you should not need to call this function.


## serverutil
- **serverutil.minify_template(template, data_table, writer, media_type)**
	- Calls [serverutil.MinifyTemplate](https://pkg.go.dev/github.com/gochan-org/gochan/pkg/server/serverutil#MinifyTemplate) with the given `template` object, `data_table` (as variables passed to the template), `writer`, and `media_type`. See [registermgmtpage.lua](./examples/plugins/registermgmtpage.lua) for an example

## uploads
- **uploads.register_handler(ext string, function(upload, post, board, filePath, thumbPath, catalogThumbPath, infoEv, accessEv, errEv))**
	- Registers a function to be called for handling uploaded files with the given extension. See [pdf_thumbnail.lua](./examples//plugins/pdf_thumbnail.lua) for a usage example.
- **uploads.get_thumbnail_ext(upload_ext string)**
	- Returns the configured (or built-in) thumbnail file extension to be used for the given upload extension
- **uploads.set_thumbnail_ext(upload_ext string, thumbnail_ext string)**
	- Sets the thumbnail extension to be used for the given upload extension

## url
- **url.join_path(base string, ext...string)**
  - Returns a string representing a URL-compatible path, with `ext` joined to `base`
- **path_escape(path string)**
  - Returns a string with any special characters escaped to be compatible with URL paths
- **path_unescape(escaped string)**
  - Attempts to unescape the given string, and returns the result and any errors (or nil if it was successful)
- **query_escape(query string)**
  - Escapes the given string so that it can be used in a URL query
- **query_unescape(escaped string)**
  - Attempts to unescape the given query-escaped string, and returns the result and any errors (or nil if it was successful)


# Events
This is a list of events that gochan may trigger at some point and can be used in the plugin system.

- **db-connected**
	- Triggered after gochan successfully connects to the database but before it is checked and initialized (db version checking, provisisioning, etc)

- **db-initialized**
	- Triggered after the database is successfully initialized (db version checking, provisioning, etc)

- **db-views-reset**
	- Triggered after the SQL views have been successfully reset, either immediately after the database is initialized, or by a staff member

- **incoming-upload**
	- Triggered by the `gcsql` package when an upload is attached to a post. It is triggered before the upload is entered in the database

- **message-pre-format**
	- Triggered when an incoming post or post edit is about to be formatted, event data includes the post object and the HTTP request

- **reset-boards-sections**
	- Triggered when the boards and sections array needs to be refreshed

- **shutdown**
	- Triggered when gochan is about to shut down, in `main()` as a deferred call

- **startup**
	- Triggered when gochan first starts after its plugin system is initialized. This is (or at least should be) only triggered once.

- **upload-saved**
	- Triggered by the `posting` package when an upload is saved to the disk but before thumbnails are generated.