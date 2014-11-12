FROM google/golang

RUN apt-get install -y build-essential libsqlite3-dev pkg-config file supervisord

WORKDIR /gopath/src/app
ADD . /gopath/src/app/
RUN go get app
RUN cd /gopath/src/app/
RUN go build

#... will download files and process them to create ipdb.sqlite
RUN cd db && ./updatedb
RUN file /gopath/src/app/db/ipdb.sqlite

RUN /usr/bin/install -o www-data -g www-data -m 0755 -d /var/log/freegeoip

EXPOSE 8080

CMD []
ENTRYPOINT ["/gopath/bin/app"]
