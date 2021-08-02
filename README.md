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
5. Go to http://[gochan url]/manage?action=staff, log in (default username/password is admin/password), and create a new admin user (and any other staff users as necessary). Then delete the admin user for security.

## Configuration
See [config.md](config.md)

## Installation using Docker
See [`docker/README.md`](docker/README.md)

## Migration
If you run gochan and get a message telling you your database is out of data, please run gochan-migration. If this does not work, please contact the developers.

## For developers (using Vagrant)
1. Install Vagrant and Virtualbox. Vagrant lets you create a virtual machine and run a custom setup/installation script to make installation easier and faster.
2. From the command line, cd into vagrant/ and run `vagrant up`. By default, MySQL/MariaDB is used, but if you want to test with a different SQL type, run `GC_DBTYPE=dbtype vagrant up`, replacing "dbtype" with either mysql or postgresql
3. After it finishes installing the Ubuntu VM, follow the printed instructions.

## For developers (using vscode)
1. Install go, the vs-go extention and gcc (I think, let me know if you need something else)
2. Install either postgreSQL or mariaDB. Setup a database with an account and enter the ip:post and login information into the gochan.json config. See "Configuration". (Tools like PG admin highly recommended for easy debugging of the database)
3. Set "DebugMode" to true. This will log all logs to the console and disable some checks.
4. Open the folder containing everything in vscode (named gochan most likely), go to "Run"
	1. Select "gochan" if you wish to run/debug the website itself
	2. Select "gochan-migrate" if you wish to run/debug the migrator
5. (Optional) Change go extention configs. Examples: save all files on start debugging
6. Press F5 or "Start Debugging" to debug.

# Theme development
See [`sass/README.md`](sass/README.md) for information on working with Sass and stylesheets.

# Development

## Style guide
* For Go source, follow the standard Go [style guide](https://github.com/golang/go/wiki/CodeReviewComments).
* All exported functions and variables should have a documentation comment explaining their functionality, as per go style guides.
* Unexported functions are preferred to have a documentation comment explaining it, unless it is sufficiently self explanatory or simple.
* Git commits should be descriptive. Put further explanation in the comment of the commit.
* Function names should not be *too* long.
* Avoid single letter variables except for simple things like iterator ints, use descriptive variables names if possible, within reason.

## Roadmap

### Near future
All features that are to be realised for the near future are found in the issues tab with the milestone "Next Release"

### Lower priority
* Improve moderation tools heavily
* Rework board creation to correctly use new fields.
* Rework any legacy structs that uses comma separated fields to use a slice instead.
* Replace all occurrences of “interfaceslice(items)” with []interface{}{items} notation, then remove interfaceslice.
* Remove all references/code related to sqlite
* RSS feeds from boards/specific threads/specific usernames+tripcodes (such as newsanon)
* Pinning a post within a thread even if its not the OP, to prevent its deletion in a cyclical thread.

### Later down the line
* Look into the possibility of a plugin system, preferably in go, a scripting language if that is not possible
* Move frontend to its own git to allow easier frontend swapping
* API support for existing chan browing phone apps
* Social credit system to deal with tor/spam posters in a better way
* Better image fingerpringing and banning system (as opposed to a hash)

### Possible experimental features:
* Allow users to be mini-moderators within threads they posted themselves, to prevent spammers/derailers.