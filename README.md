# Gochan

A muti-threaded Imageboard software project in Go

At the moment, regular users can:

- Log in as the initial admin account (password is "password"

- Create new threads

- Post in a thread

- Upload an image with a post


Staff can:

- View announcements (announcment editing coming soon)

- create and delete users (if they are logged into an administrator account)

- Log out

- Use various other half implemented functions

- Delete posts without needing to put in a password

## To-do list:

+ Important
	* General
		- Set up daemonization
		- add delete post functionality on the inline post dropdown
		- add similar dropdown to the postbox for staff with mod name, mod rank, raw html, sticky, and lock
		- make dropdowns close by clicking anywhere outside them
		- set up board pagination
		- make jquery stuff in manage pages more consistent (no reloading the whole page if in a lightbox)
		- set up board creation
	* Security
		- Add banning functionality
		- add mod tools (delete, search IP, permaban, etc) to the dropdown for staff
		- check for user-agent on post submission/staff login	
		
+ Bugs
	- fix execute sql page
	- fix "multiple response.WriteHeader calls" bug
	- fix cross-browser compatibility issues

+ Features
	- Load error html pages into memory and use templating
	- Set up load balancing
	- Set up HTTPS for management
	- Set up timezone adjusting
	- Give administrator server control options (restart/shutdown daemon, etc)
	- add edit post functionality, both for staff and regular posters
	- set up video embeds
	- set up optional tor exit node blocking
	- set up international board (geoip + flags)
	- set up board pagination
	- set up client-side watched threads list
	- set up Ponychan/4chan-X style javascript features
	- generate robots.txt
	- generate post rss, to be used for recent posts on the front page