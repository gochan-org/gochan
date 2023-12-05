FROM golang:1.20-alpine3.18

COPY docker/build-image.sh .

RUN ["./build-image.sh"]

WORKDIR /opt/gochan

COPY examples/configs/gochan-fastcgi.nginx /etc/nginx/http.d/gochan.conf
COPY examples/configs/gochan.example.json /etc/gochan/gochan.json

# Get all
COPY . .
RUN ["./build.py"]
RUN ["./build.py", "install"]


EXPOSE 9000


COPY docker/startup.sh /opt/gochan/startup.sh
COPY docker/wait-for.sh /opt/gochan/wait-for.sh
CMD ["/opt/gochan/docker/startup.sh"]
