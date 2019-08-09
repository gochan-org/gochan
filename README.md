Gochan
=======
A semi-standalone imageboard server written in Go

http://gochan.org


# Installation

## Basic installation (from a release)
1. Extract the .tar.gz or the .zip file into a directory (for example, your home directory)
2. Copy gochan.example.json to either gochan.json or (if you're in a UNIX-like OS) /etc/gochan/gochan.json and modify it as needed. See the Configuration section for more info.
3. If you're using nginx, copy gochan-http.nginx, or gochan-fastcgi.nginx if `UseFastCGI` is set to true to /etc/nginx/sites-enabled/, or the appropriate folder in Windows.
4. If you're using a Linux distribution with systemd, you can optionally copy gochan.service to /lib/systemd/system/gochan.service and run `systemctl enable gochan.service` to have it run on startup. Then run `systemctl start gochan.service` to start it as a background service.
	1. If you aren't using a distro with systemd, you can start a screen session and run `/path/to/gochan`
5. Go to http://[gochan url]/manage?action=staff, log in (default username/password is admin/password), and create a new admin user (and any other staff users as necessary). Then delete the admin user for security.

## Configuration
1. Make sure to set `DBtype`, `DBhost`, `DBname`, `DBusername`, and `DBpassword`, since these are required to connect to your SQL database. Valid `DBtype` values are "mysql", "postgres", and "sqlite3".
	1. To connect to a MySQL database, set `DBhost` to "tcp(ip:3306)" or a different port, if necessary.
	2. To connect to a PostgreSQL database, set `DBhost` to the IP address or hostname. Using a UNIX socket may work as well, but it is currently untested.
	3. To connect to a SQLite database, set `DBhost` to the path of the database file. It will be created if it does not already exist.
2. Set `DomainRegex`,`SiteDomain`, since these are necessary in order to post and log in as a staff member.
3. If you want to see debugging info/noncritical warnings, set verbosity to 1.

## For developers (using Vagrant)
1. Install Vagrant and Virtualbox. Vagrant lets you create a virtual machine and run a custom setup/installation script to make installation easier and faster.
2. From the command line, cd into vagrant/ and run `vagrant up`
3. After it finishes installing the Ubuntu VM, follow the printed instructions.

# Theme development
See [`sass/README.md`](sass/README.md) for information on working with Sass and stylesheets.
