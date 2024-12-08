Gochan
=======

Gochan is an imageboard server with a backend written in Go. It works in a manner similar to Kusaba X, Tinyboard and others. As such, Gochan generates static HTML files which can optionally be served by a separate web server.

Demo installation: https://gochan.org

# Installation

## Basic installation (from a release)
1. Extract the .tar.gz or the .zip file into a directory (for example, your home directory)
2. Copy gochan.example.json to either gochan.json or (if you're in a UNIX-like OS) /etc/gochan/gochan.json and modify it as needed. See the Configuration section for more info.
3. If you're using nginx, copy gochan-http.nginx, or gochan-fastcgi.nginx if `UseFastCGI` is set to true to /etc/nginx/sites-enabled/, or the appropriate folder in Windows.
4. If you're using a Linux distribution with systemd, you can optionally copy gochan.service to /lib/systemd/system/gochan.service and run `systemctl enable gochan.service` to have it run on startup. Then run `systemctl start gochan.service` to start it as a background service.
	1. If you aren't using a distro with systemd, you can start a screen session and run `/path/to/gochan`
5. Go to http://[gochan url]/manage/staff, log in (default username/password is admin/password), and create a new admin user (and any other staff users as necessary). Then delete the admin user for security.

## Installation using Docker
See [`docker/README.md`](docker/README.md)

## Configuration
See [config.md](config.md)

## Plugins
Gochan has a built-in [Lua](https://lua.org) interpreter and an event system to allow for extending your Gochan instance's functionality. See [plugin_api.md](./plugin_api.md) for a list of functions and events, and information about when they are used.

## Migration
If you run gochan v3.0 or newer and get a message telling you that your database is out of date, please run gochan-migration -updatedb. If this does not work, please contact the developers.

## For developers (using Vagrant)
1. Install Vagrant and Virtualbox. Vagrant lets you create a virtual machine and run a custom setup/installation script to make installation easier and faster.
2. From the command line, cd into vagrant/ and run `vagrant up`. By default, MariaDB (a MySQL fork that most Linux distributions are defaulting to) is used, but if you want to test with a different SQL type, run `GC_DBTYPE=dbtype vagrant up`, replacing "dbtype" with either mysql or postgresql
	- **Note on MySQL:** While MariaDB and mainline MySQL are very similar, there are a few features that MariaDB has that MySQL lacks that may cause issues. To specifically use the mainline MySQL server, run `GC_MYSQL_MAINLINE=1 vagrant up`
3. After it finishes installing the Ubuntu VM, follow the printed instructions.

## For developers (using VS Code)
1. Install Go, the VS Code Go extention, and gcc
2. Install MariaDB, PostgreSQL, or MySQL. Setup a database with an account and enter the ip:post and login information into the gochan.json config. See "Configuration". (Tools like PG admin highly recommended for easy debugging of the database)
3. Set "DebugMode" to true. This will log all logs to the console and disable some checks.
4. Open the folder containing everything in vscode (named gochan most likely), go to "Run"
	1. Select "gochan" if you wish to run/debug the website itself
	2. Select "gochan-migrate" if you wish to run/debug the migration tool
5. (Optional) Change go extention configs. Examples: save all files on start debugging
6. Press F5 or "Start Debugging" to debug.

## Frontend development - (S)CSS or JavaScript
See [`frontend/README.md`](frontend/README.md) for information on working with Sass and developing gochan's JavaScript frontend.

## Backend development

## Style guide
* For Go source, follow the standard Go [style guide](https://github.com/golang/go/wiki/CodeReviewComments).
* variables and functions exposed to Go templates should be in camelCase, like Go variables
* All exported functions and variables should have a documentation comment explaining their functionality, as per Go style guides.
* Unexported functions are preferred to have a documentation comment explaining it, unless it is sufficiently self explanatory or simple.
* Git commits should be descriptive. Put further explanation in the comment of the commit.
* Function names should not be *too* long.
* Avoid single letter variables except for simple things like iterator ints, use descriptive variables names if possible, within reason.

## Roadmap

### Near future
* Fully implement cyclical threads
* Implement +50
* Add more banners
* Add more plugin support (more event triggers)

### Lower priority
* RSS feeds from boards/specific threads/specific usernames+tripcodes (such as newsanon)
* Pinning a post within a thread even if its not the OP, to prevent its deletion in a cyclical thread.
