# Docker usage info
To start gochan using Docker with one of the .yml files, run `docker compose -f docker-compose-<dbtype>.yml up`. It will build and spin up a container for the gochan server and a container for the database, with the exception of SQLite, which is loaded from a file.

When the containers are started, they will mount volumes in the volumes directory for access to gochan logs, the document root, the configuration, and database data files.

See docker-compose-*.yml files for example usage

## Dockerfile args
Arg              | Default value | Description
-----------------|---------------|-----------------
GOCHAN_PORT      | 80            | The port that the server will listen on. You will want to expose the same port to the host.
GOCHAN_SITE_HOST | 127.0.0.1     | The host that the server will expect incoming requests to be for. for example, if your server is at 1.2.3.4 but is behind Cloudflare using domain example.com, you will need to set this to example.com
GOCHAN_DB_TYPE   | *none*        | This mostly corresponds to the DBtype config value, the SQL driver name to be used. The exception being "mariadb", which should be "mysql" in gochan.json. Here, "mysql" refers specifically to the mainline MySQL implementation.
GOCHAN_DB_HOST   | *none*        | The host and port to connect to the database for MySQL/MariaDB, Postgresql, or the path to the SQLite database file.
