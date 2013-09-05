# freegeoip.net

freegeoip.net is a public web service for searching
[geolocation](http://en.wikipedia.org/wiki/Geolocation) of IP addresses.
This is the source code of freegeoip.net's web server and script for building
the IP database.

## Overview

freegeoip.net is the result of a web server research project that started in
2009 hosted at Google's [App Engine](http://en.wikipedia.org/wiki/Geolocation),
using the Python API.
A year later it moved to its own server infrastructure built on the
[Cyclone](http://cyclone.io) web framework, backed by
[Twisted](http://twistedmatrix.com) and [PyPy](http://pypy.org).

The current version is written in Go as the experiments progress with
[go-web](https://github.com/fiorix/go-web) and
[go-redis](https://github.com/fiorix/go-redis).

### Install

List of prerequisites for building and running the server:

- Go compiler - for ``freegeoip.go``
- libsqlite3-dev, gcc or llvm - for dependency ``go-sqlite3``
- Python - for the ``updatedb`` script
- Redis - for API usage quotas
- The IP database

The following instructions are for Debian and Ubuntu servers.

Make sure Go is installed and both $GOROOT and $GOPATH are set, then run:

	apt-get install build-essential libsqlite3-dev pkg-config
	go get github.com/fiorix/freegeoip
	cd $GOPATH/src/github.com/fiorix/freegeoip
	go build

On recent OSX you might have to set the CC=clang before ``go build`` if
the sqlite3 package fails to compile.

Proceed to building the IP database before starting the server.

### Building the IP database

The IP database is composed of multiple files from multiple sources. It's a
combination of IP subnets, country codes, city names, etc.

There's a helper script under the ``db`` directory that automates the process
of building the database, and can be used regularly to update it as well.

It's a Python script called ``updatedb`` that creates ``ipdb.sqlite``:

	$ cd db
	$ ./updatedb
	... will download files and process them to create ipdb.sqlite
	$ file ipdb.sqlite
	ipdb.sqlite: SQLite 3.x database

This service includes GeoLite data created by MaxMind, available from
maxmind.com.

### Running

Use either ``go run freegeoip.go`` or ``go build; ./freegeoip`` to compile and
run the server, then point the browser to http://localhost:8080.

Server needs ``freegeoip.conf`` to be in the current directory but an alternate
config can be specified using the ``-config`` command line option.

If the database is not accessible (e.g. file does not exist, permissions) or
redis-server is unreachable, all queries will result in HTTP 503
(Service Unavailable).

For SSL, use the following command to generate self-signed certificates:

	go run $GOROOT/src/pkg/crypto/tls/generate_cert.go --host localhost

We recommend [supervisor](http://supervisord.org) for running the server in
production. On Ubuntu, install ``apt-get install supervisor`` and drop the
following config in ``/etc/supervisor/conf.d/freegeoip.conf``:

	[program:freegeoip]
	user=www-data
	redirect_stderr=true
	directory=/opt/freegeoip
	command=/opt/freegeoip/freegeoip
	stdout_logfile=/var/log/freegeoip.log
	stdout_logfile_maxbytes=50MB
	stdout_logfile_backups=20

If the server is proxied by Nginx or another HTTP load balancer, edit the
configuration file and set ``xheaders="true"`` and it'll use X-Real-IP or
X-Forwarded-For HTTP headers when available, as the client IP.

For listening on low ports as non-root user (e.g. www-data) on linux, set
file capabilities at least once before running it:

	/sbin/setcap 'cap_net_bind_service=+ep' /opt/freegeoip/freegeoip

### Usage

Point the browser to http://localhost:8080 and search for IPs or hostnames.

Use curl from the command line to query the API:

	$ curl -v http://localhost:8080/{format}/{ip_or_hostname}

It supports csv, json and xml as the output format. JSON supports callbacks
with the ``callback`` query argument. The client (self) IP is used if
**ip_or_hostname** is omitted in the query.

Examples:

	$ curl -v http://localhost:8080/csv/
	$ curl -v http://localhost:8080/xml/
	$ curl -v http://localhost:8080/xml/freegeoip.net
	$ curl -v http://localhost:8080/json/github.com?callback=foobar

If the server is listening on unix sockets, use ``nc`` to test:

	echo -ne 'GET /json/my-domain.abc HTTP/1.0\r\n\r\n' | nc -U /tmp/freegeoip.sock

### Credits

Thanks to (in no particular order):

- [Gleicon](https://github.com/gleicon) for all the drama.
- Google for the map, Go, and AngularJS.
- Twitter for Bootstrap.
- MaxMind for the current database.
- ipinfodb.com for both the IP and timezones database back in 2010 and 2011.
