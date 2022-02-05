#!/usr/bin/env bash

# used for connecting to gochan's database whether it's using MySQL/MariaDB or PostgreSQL

if [ "$DBTYPE" = "mysql" ] || [ -z "$DBTYPE" ]; then
	mysql -stu gochan -D gochan -pgochan
elif [ "$DBTYPE" = "postgresql" ]; then
	psql -U gochan -h 127.0.0.1 gochan
else
	echo "DB type '$DBTYPE' not supported"
fi
