# freegeoip

This is the source code of the freegeoip software. It contains both
the web server that empowers freegeoip.net, and a package for the
[Go](http://golang.org) programming language that enables any web server
to support IP geolocation with a simple and clean API.

See http://en.wikipedia.org/wiki/Geolocation for details about geolocation.

## Web Server

The freegeoip web server is a standalone program that serves an HTTP API
for searching the geolocation of IP addresses. To serve the API, it uses
an IP database that is automatically downloaded and auto-updated from
the internet when the server is running.

The API returns data encoded in popular formats such as CSV, XML, JSON
and JSONP.

### Download

If you're not a developer and is only looking for the web server, you
can download binaries directly.

See https://github.com/fiorix/freegeoip/releases for tarballs for your
platform.

### Usage

Run the server:

	./freegeoip

Wait for it to download the IP database file for the first time. It does
it in background and writes a message to the console when ready. If you'd
like to use an alternative database source, see the `-db` command line
flag.

If the server is queried when there is no database available, including
this initial first run, it returns *HTTP 503 (Service Unavailable)*, since
it can't service requests before a proper database is in place.

### Querying

You can use any HTTP client to test the server. The examples below use
curl and the environment variable $freegeoip, which must be set to the
address of your server, like http://localhost:8080 for example:

	export freegeoip=http://localhost:8080

Querying the API is very straightforward: you just have to pick a format
of your choice and provide either the IP address or hostname that you'd
like to search for. The syntax is as follows:

	$freegeoip/{format}/{IP_or_hostname}

Examples:

	curl -i $freegeoip/csv/8.8.8.8

	curl -i $freegeoip/xml/4.2.2.2

	curl -i $freegeoip/json/github.com

If a domain or hostname is passed in the URL, the server will resolve that
name to its IP address and lookup the IP instead. If the hostname contains
multiple IPs associated to it, the server picks one randomly, which means
it could be either IPv4 or IPv6.

If no IP or hostname is provided, then the server queries the IP address
of the HTTP client.

Example:

	curl -i $freegeoip/json/   (this queries your own IP address)

The JSON endpoint also supports JSONP, by adding a `callback` argument
to the request query.

Example:

	curl -i $freegeoip/json/8.8.8.8?callback=f

See http://en.wikipedia.org/wiki/JSONP for details on how JSONP works.

### Docker image

Build the docker image:

	docker build -t my/freegeoip .

If you want just the API, not the front-end, edit the Dockerfile and
comment out the `-public` command line flag.

Or use the official image:

	docker run -d --name freegeoip -p 8080:8080 fiorix/freegeoip

If you need quota then link the container to a Redis container:

	docker run -d --name redis -p 6379:6379 dockerfile/redis
	docker run -d --name freegeoip --link redis:redis -p 8080:8080 fiorix/freegeoip -redis redis:6379 -quota-max 10000

You can use `redis-cli monitor` to assure things are working as expected.

## freegeoip package for Go

The freegeoip package for the Go programming language provides two APIs:

- A database API that requires zero maintenance of the IP database;
- A geolocation `http.Handler` that can be used/served by any http server.

tl;dr if all you want is code then see the `example_test.go` file.

Otherwise check out the godoc reference.

[![GoDoc](https://godoc.org/github.com/fiorix/freegeoip?status.svg)](https://godoc.org/github.com/fiorix/freegeoip)
[![Build Status](https://secure.travis-ci.org/fiorix/freegeoip.png)](http://travis-ci.org/fiorix/freegeoip)

### Features

- Zero maintenance

The DB object alone can download an IP database file from the internet and
service lookups to your program right away. It will auto-update the file in
background and always magically work.

- DevOps friendly

If you do care about the database and have the commercial version of the
MaxMind database, you can update the database file with your program running
and the DB object will load it in background. You can focus on your stuff.

- Extensible

Besides the database part, the package provides an `http.Handler` object
that you can add to your HTTP server to service IP geolocation lookups with
the same simplistic API of freegeoip.net. There's also an interface for
crafting your own HTTP responses encoded in any format.

### Install

Install the package:

	go get github.com/fiorix/freegeoip

Install the web server:

	go get github.com/fiorix/freegeoip/cmd/freegeoip

Test coverage is quite good and may help you find the stuff you need.
