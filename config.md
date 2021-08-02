# gochan configuration
See [gochan.example.json](sample-configs/gochan.example.json) for an example gochan.json.

## Server-critical stuff
* You'll need to edit some of the values (like `ListenIP` and `UseFastCGI` based on your system's setup. For an example nginx configuration, see [gochan-fastcgi.nginx](sample-configs/gochan-fastcgi.nginx) for FastCGI and [gochan-http.nginx](sample-configs/gochan-http.nginx) for passing through HTTP.
* `DocumentRoot` refers to the root directory on your filesystem where gochan will look for requested files.
* `TemplateDir` refers to the directory where gochan will load the templates from.
* `LogDir` refers to the directory where gochan will write the logs to.

**Make sure gochan has read-write permission for `DocumentRoot` and `LogDir` and read permission for `TemplateDir`**

## Database configuration
Valid `DBtype` values are "mysql" and "postgres" (sqlite3 is no longer supported for stability reasons, though that may or may not come back).
1. To connect to a MySQL database, set `DBhost` to "x.x.x.x:3306" (replacing x.x.x.x with your database server's IP or domain) or a different port, if necessary. You can also use a UNIX socket if you have it set up, like "unix(/var/run/mysqld/mysqld.sock)".
2. To connect to a PostgreSQL database, set `DBhost` to the IP address or hostname. Using a UNIX socket may work as well, but it is currently untested.
3. Set `SiteDomain`, since these are necessary in order to post and log in as a staff member.
3. If you want to see debugging info/noncritical warnings, set verbosity to 1.
4. If `DBprefix` is set (not required), all gochan table names will be prefixed with the `DBprefix` value. Once you run gochan for the first time, you really shouldn't edit this value, since gochan will assume the tables are missing.

## Website configuration
* `SiteName` is used for the name displayed on the home page.
* `SiteSlogan` is used for the slogan (if set) on the home page.
* `SiteDomain` is used for links throughout the site.
* `WebRoot` is used as the prefix for boards, files, and pretty much everything on the site. If it isn't set, "/" will be used.

## Styles
* `Styles` is an array, with each element representing a theme selectable by the user from the frontend settings screen. Each element should have `Name` string value and a `Filename` string value. Example:
```JSON
"Styles": [
	{ "Name": "Pipes", "Filename": "pipes.css" }
]
```
* If `DefaultStyle` is not set, the first element in `Styles` will be used.

## Misc
* `ReservedTrips` is used for reserving secure tripcodes. It should be an array of strings. For example, if you have `abcd##ABCD` and someone posts with the name ##abcd, their name will instead show up as !!ABCD on the site.
* `BanColors` is used for the color of the text set by `BanMessage`, and can be used for setting per-user colors, if desired. It should be a string array, with each element being of the form `"username:color"`, where color is a valid HTML color (#000A0, green, etc) and username is the staff member who set the ban. If a color isn't set for the user, the style will be used to set the color.
