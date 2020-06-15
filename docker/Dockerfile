FROM golang:1.13.9-alpine3.11

RUN apk --no-cache add \
						postgresql-client	\
						mariadb-client		\
						nginx 				\
						ffmpeg				\
						make 				\
						git					\
						gcc					\
						musl-dev			\
						bash 				

RUN mkdir -p /root/bin \
	&& ln -s /usr/lib/go-1.13/bin/* /root/bin/ \
	&& export PATH=$PATH:/root/bin \
	&& echo "export PATH=$PATH:/root/bin" >> /root/.bashrc \
	&& rm -f /etc/nginx/sites-enabled/* /etc/nginx/sites-available/* \
	&& mkdir -p /var/lib/nginx \
	&& mkdir -p /var/lib/nginx/tmp \
	&& mkdir -p /run/nginx/

WORKDIR /opt/gochan

# Get dependencies
COPY Makefile .
RUN make dependencies

RUN rm /etc/nginx/conf.d/default.conf
COPY sample-configs/gochan-fastcgi.nginx /etc/nginx/conf.d/gochan.conf
COPY sample-configs/gochan.example.json /etc/gochan/gochan.json

# Get all
COPY . .

EXPOSE 9000

CMD ls -la /opt/gochan && ls -la && ls -la .. && sed -i /etc/gochan/gochan.json \
	-e 's/"Port": 8080/"Port": 9000/' \
	-e 's/"UseFastCGI": false/"UseFastCGI": true/' \
	-e 's/"DomainRegex": ".*"/"DomainRegex": "(https|http):\\\/\\\/(.*)\\\/(.*)"/' \
	-e 's#"DocumentRoot": "html"#"DocumentRoot": "/srv/gochan"#' \
	-e 's#"TemplateDir": "templates"#"TemplateDir": "/usr/local/share/gochan/templates"#' \
	-e 's#"LogDir": "log"#"LogDir": "/var/log/gochan"#' \
	-e 's/"Verbosity": 0/"Verbosity": 1/' \
	-e "s/\"DBtype\".*/\"DBtype\": \"${DBTYPE}\",/" \
	-e "s/\"DBhost\".*/\"DBhost\": \"tcp(${DATABASE_HOST}:${DATABASE_PORT})\",/" \
	-e "s/\"DBname\".*/\"DBname\": \"${DATABASE_NAME}\",/" \
	-e "s/\"DBusername\".*/\"DBusername\": \"${DATABASE_USER}\",/" \
	-e "s/\"DBpassword\".*/\"DBpassword\": \"${DATABASE_PASSWORD}\",/" \
	&& make \
	&& make install \
	&& nginx \
	&& echo "pinging db" \
	&& docker/wait-for.sh $DATABASE_HOST:$DATABASE_PORT -t 30 \
	&& /opt/gochan/gochan
