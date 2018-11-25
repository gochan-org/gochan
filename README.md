# Gochan
A semi-standalone imageboard server written in Go

http://gochan.org


## Installation

### Basic installation (from a release)
1. Extract the .tar.gz or the .zip file into a directory (for example, your home directory)
2. Copy gochan.example.json to gochan.json and modify it to your liking.
	1. If you want to see debugging info/noncritical warnings, set verbosity to 1. If you want to see benchmarks as well, set it to 2.
	2. Make sure to set `DBname`, `DBusername`, and `DBpassword`, since these are required to connect to your MySQL database. Set `DomainRegex`,`SiteDomain`, since these are necessary in order to post and log in as a staff member without being rejected.
3. If you're using nginx, copy gochan-fastcgi.nginx, or gochan-http.nginx if `UseFastCGI` is set to true to /etc/nginx/sites-enabled/, or the appropriate folder in Windows.
4. If you're in Linux, you can optionally copy gochan.service to ~/.config/systemd/user/gochan.service and run `systemctl enable gochan.service` to have it run on login and `systemctl start gochan.service` to start it as a background service.
	1. If you aren't using a distro with systemd, you can start a screen session and run `./gochan`
5. Go to http://[gochan url]/manage?action=boards, log in (default username/password is admin/password), create a board, and go to http://[gochan url]/manage?action=rebuildall
	1. For security reasons, you should probably go to http://[gochan url]/manage?action=staff to create a new admin user account and delete admin.

### For developers (using Vagrant)
1. Install Vagrant and Virtualbox. Vagrant lets you create a virtual machine and run a custom setup/installation script to make installation easier and faster.
2. From the command line, cd into vagrant/ and run `vagrant up`
3. After it finishes installing the Ubuntu VM, follow the printed instructions.

### Theme development
See `sass/README.md` for information on working with Sass and stylesheets.

