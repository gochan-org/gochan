FROM golang

COPY . /opt/gochan

EXPOSE 80

ENV DBTYPE=postgresql
ENV GCVERSION=v2.9.1
ENV FROMDOCKER=1

RUN /opt/gochan/vagrant/bootstrap.sh



CMD [ "/opt/gochan/gochan" ]