FROM alpine:3.9

WORKDIR /app
RUN apk --update add mysql && rm -f /var/cache/apk/*

COPY startup.sh /startup.sh
COPY my.cnf /etc/mysql/my.cnf

EXPOSE 3306
CMD ["/startup.sh"]

