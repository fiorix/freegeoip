freegeoip.net web server
========================

This is the current web server of freegeoip.net, a public HTTP API for
searching [Geolocation](http://en.wikipedia.org/wiki/Geolocation) of IP
addresses.

It is the result of a research project that started in 2009 using
[Google App Engine](http://en.wikipedia.org/wiki/Geolocation)'s Python API.
A year later it moved to its own server infrastructure, built on the
[cyclone](http://cyclone.io) web framework and backed by pypy.

It's been rewritten in Go using [go-web](https://github.com/fiorix/go-web),
another experimental web anti-framework similar to web.py, tornado and
cyclone. It's a tool to speed up the development and avoid repeating the same
code over and over on web applications.

Database
--------

The database is composed of multiple files, from multiple sources. It's a
combination of IP networks, country codes, city names, etc.

There's a helper script under the _db_ directory that tries to download all
files, process and combine them to build the database. It might eventually
fail.

Make sure the db exists before starting the server. In the _db_ directory,
execute *updatedb* - it's a Python script. Should look like this:

	$ ./updatedb
	Downloading http://dev.maxmind.com/static/csv/codes/maxmind/region.csv
	Downloading http://musta.sh/files/all_cities_in_the_world.csv.zip
	Extracting all_cities_in_the_world.csv -> all_cities_in_the_world.csv
	Checking http://geolite.maxmind.com/download/geoip/database/GeoLiteCity_CSV/GeoLiteCity-latest.zip
	Downloading http://geolite.maxmind.com/download/geoip/database/GeoLiteCity_CSV/GeoLiteCity-latest.zip
	Extracting GeoLiteCity_20130305/GeoLiteCity-Blocks.csv -> GeoLiteCity-Blocks.csv
	Extracting GeoLiteCity_20130305/GeoLiteCity-Location.csv -> GeoLiteCity-Location.csv
	Downloading http://geolite.maxmind.com/download/geoip/database/GeoIPCountryCSV.zip
	Extracting GeoIPCountryWhois.csv -> GeoIPCountryWhois.csv
	Importing GeoIPCountryWhois.csv: .178306 records!
	Importing region.csv: 4053 records!
	Importing GeoLiteCity-Blocks.csv: ......................2256115 records!
	Importing GeoLiteCity-Location.csv: ....403753 records!
	Updating region names: 322 names updated.

This service includes GeoLite data created by MaxMind, available from
maxmind.com.

Usage
-----

Run the dev server with *go run freegeoip.go* and point your browser to it.
Use curl to query the API, like this:

	$ curl http://localhost:8080/xml/ip_or_hostname

It supports CSV, JSON and XML. If *ip_or_hostname* is omitted, the IP of the
client making the request is used.

If the server is listening on unix sockets, use *nc* to test:

	echo -ne 'GET /json/ HTTP/1.0\r\nX-Real-IP: pwnz\r\n\r\n' | nc -U /tmp/freegeoip

Command line
------------

To query the API from the command line, add this to *~/.bash_profile*:

	function geoip_curl_xml { curl -D - http://freegeoip.net/xml/$1; }
	alias geoip=geoip_curl_xml

Credits
-------

Thanks to (in no particular order):

- google.com: Because it wouldn't look so good without the map
- twitter.com: Bootstrap makes dirty programmers feel like artists
- ipinfodb.com: For providing both GeoIP and Timezones database (2010 and 2011)
- maxmind.com: For the current DB
