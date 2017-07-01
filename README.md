# freegeoip

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

This is the source code of the freegeoip software. It contains both
the web server that empowers freegeoip.net, and a package for the
[Go](http://golang.org) programming language that enables any web server
to support IP geolocation with a simple and clean API.

See http://en.wikipedia.org/wiki/Geolocation for details about geolocation.

Developers looking for the Go API can skip to the [Package freegeoip](#packagefreegeoip)
section below.

## Running

This section is for people who desire to run the freegeoip web server
on their own infrastructure. The easiest and most generic way of doing
this is by using Docker.

See the [Server Options](#serveroptions) below for more information on configuring
the server.

### Docker

Install Docker on Ubuntu 14.04 LTS:

```bash
sudo apt-get install docker.io
```

Install Docker on CentOS 7:

```bash
yum install docker
```

Run the freegeoip web server:

```bash
docker run --restart=always -p 8080:8080 -d fiorix/freegeoip
```

Test:

```bash
curl localhost:8080/json/1.2.3.4
```

See the **API** section below for details.

### Other Linux, OS X, or FreeBSD

There are [pre-compiled binaries](https://github.com/fiorix/freegeoip/releases) available. You'll have to set up your own init scripts for your system.

There is also a [Chef cookbook](https://supermarket.chef.io/cookbooks/freegeoip) to deploy it automatically.

<a name="serveroptions">

### Server Options

You can configure the freegeoip web server to listen on a port
other than the default 8080, and also listen on HTTPS by passing
an ip:port and X.509 certificate and key files.

For example, to have freegeoip listen on port 12904, run the following command:

```
docker run --restart=always -p 12904:12904 -d fiorix/freegeoip -http 0.0.0.0:12904
```

These and many other options are described in the help. If you're
using Docker, you can see them like this:

```bash
docker run --rm -it fiorix/freegeoip --help
```

By default, the Docker image of freegeoip does not provide the
web page from freegeiop.net, it only provides the API.

If you want to serve that page, you can pass the `-public=/var/www`
parameter in the command line. You can also tell Docker to mount that
directory as a volume on the host machine and have it serve your own
page, using Docker's `-v` parameter.

If the freegeoip web server is running behind a proxy or load
balancer, you have to run it passing the `-use-x-forwarded-for`
parameter and provide the `X-Forwarded-For` HTTP header so the web
server is capable of using the source IP address of the connection
to perform geolocation lookups when an IP is not provided to
the API, e.g. `/json/` vs `/json/1.2.3.4`.

## Database

The current implementation uses the free [GeoLite2](http://dev.maxmind.com/geoip/geoip2/geolite2/)
database from MaxMind.

In the past we had databases from other providers, and at some point
even our own database comprised of different sources. This means it
might change in the future.

If you have purchased the commercial database from MaxMind, you can
point the freegeoip web server or Go API to the URL of it, or local
file, and the server will use it.

In case of files on disk, you can replace with a new version and the
freegeoip software will load it automatically. URLs are frequently
checked in background, and if a new version of the database is
available it is loaded automatically also.

## API

The freegeoip API is served by endpoints that encode the response
in different formats.

Example:

```bash
curl freegeoip.net/json/
```

Returns the geolocation information of your own IP address, the source
IP address of the connection.

You can pass a different IP or hostname:

```bash
curl freegeoip.net/json/github.com
```

To lookup the geolocation of `github.com` after resolving its IP address,
which might be IPv4 or IPv6.

Responses can also be encoded as JSONP, by adding the `callback` parameter:

```bash
curl freegeoip.net/json/?callback=foobar
```

Same semantics are available for the `/xml/{ip}` and `/csv/{ip}` endpoints
except the callback parameter.

## Metrics and profiling

The freegeoip web server can provide metrics about its usage, and also
supports runtime profiling.

Both are disabled by default, but can be enabled by passing the
`-internal-server=:8081` parameter in the command line. Metrics are
generated for [Prometheus](http://prometheus.io) and can be queried
at `/metrics` even with curl.

HTTP pprof is available at `/debug/pprof` and the examples from
the [pprof](https://golang.org/pkg/net/http/pprof/) package work.

<a name="packagefreegeoip">
## Package freegeoip

The freegeoip package for the Go programming language provides two APIs:

- A database API that requires zero maintenance of the IP database;
- A geolocation `http.Handler` that can be used/served by any http server.

tl;dr if all you want is code then see the `example_test.go` file.

Otherwise check out the godoc reference.

[![GoDoc](https://godoc.org/github.com/fiorix/freegeoip?status.svg)](https://godoc.org/github.com/fiorix/freegeoip)
[![Build Status](https://secure.travis-ci.org/fiorix/freegeoip.png)](http://travis-ci.org/fiorix/freegeoip)
[![GoReportCard](https://goreportcard.com/badge/github.com/fiorix/freegeoip)](https://goreportcard.com/report/github.com/fiorix/freegeoip)

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

Download the package:

	go get -d github.com/fiorix/freegeoip/...

Install the web server:

	go install github.com/fiorix/freegeoip/cmd/freegeoip

Test coverage is quite good and tests may help you find the stuff you need.
