# Docker usage info
To start gochan using Docker with one of the .yml files, run `docker compose -f docker-compose-<dbtype>.yml up`. It will build and spin up a container for the gochan server and a container for the database, with the exception of SQLite, which is loaded from a file.

When the containers are started, they will mount volumes in the volumes directory for access to gochan logs, the document root, and database data files.

## Boring stuff
Previously, gochan's default docker-compose.yml was divided into two services, gochan+nginx and db, which mainly supported MariaDB. Now, there are four options for docker-compose, one for each database provider (with MariaDB and MySQL separated). The SyncForMac container file appears to have been incomplete so it has been removed since I am unable to test its usefulness.

Nginx has also been removed, as it is not really necessary to run a gochan server. It is only really necessary if you want to serve HTTPS (which you should). For a dev environment, you can just use any of the provided docker-compose files. For a production server, you can run nginx outside Docker (or in a separate container) and just forward ports accordingly.
