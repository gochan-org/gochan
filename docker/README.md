To run docker you have several choices.

If you want the docker container to use the host's database, copy `docker-compose.yml.default` to `docker-compose.yml` and edit the new file with your host's information.

If you want docker to manage the databse, use `docker-compose-[database].yml`.

If you are using MacOS and need better file sync between the host and the container, use `docker-compose-syncForMac.yml`.

To use from the root gochan directory, run `make docker`, or `make docker-macos` if you are using MacOS. This will use the MariaDB docker-compose file. If you want to specify which docker-compose file to use, run `docker-compose -f [docker-compose.yml file you chose] up --build` from this directory. To stop, simply use control+c to send a stop signal. This stops the docker containers but it does not delete them. They are merely frozen.

To delete the containers run `docker-compose -f [file you chose] down`. If you have a container that has a database (for example, if you chose `docker-compose-mariadb.yml`), this command will delete the database too.

If you want to use a specific docker-compose file as the default for your own computer, or you want to edit one of the default configurations given here (to change the database type, for example), copy the file and name it `docker-compose.yml`. This way, you can omit specifying the file when using docker-compose. For example, `docker-compose down` is the same as `docker-compose -f docker-compose.yml down`. The file is added to .gitignore so that your local config won't be accidentally commited.

Docker caches builds. When files change, it has to rebuild from whenever that file was added to the docker image. For example, the docker file adds `Makefile` at first and ignores the rest of the files. It uses it to download the dependencies, which can take a while. After that, it adds the rest of the files. This means that if a file is changed in a source file, docker won't have to rebuild. But if the Makefile changes, it will be forced to rebuild. This can cause Docker to bloat up after a while. Periodically remember to run `docker image prune` (also search for other deletion commands) to keep docker's storage usage relatively low. All images used thus far use Alpine, which is a small OS compared to Ubuntu or other much larger builds.
