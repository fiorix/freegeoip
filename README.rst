=============
freegeoip.net
=============
:Info: See `github <http://github.com/fiorix/freegeoip>`_ for the latest source.
:Author: Alexandre Fiori <fiorix@gmail.com>

About
=====

This is the software running behind the FREE IP Geolocation Web Service at `freegeoip.net <http://freegeoip.net>`_.
The database is not shipped with the package. However, there are instructions for downloading and generating a local version of the database, using the ``database/createdb.py`` python script.

Using
-----

The web service supports three different formats: CSV, XML and JSON (with callbacks).

- For querying GeoIP information (using curl)::

    curl http://freegeoip.net/csv/google.com
    curl http://freegeoip.net/xml/69.63.189.16
    curl http://freegeoip.net/json/74.200.247.59
    curl http://freegeoip.net/json/github.com?callback=doit

- For querying GeoIP information about your own IP::

    curl http://freegeoip.net/csv/
    curl http://freegeoip.net/xml/
    curl http://freegeoip.net/json/

- For querying Timezone information (/tz/``country_code``/``region_code``)::

    curl http://freegeoip.net/tz/xml/BR/27
    curl http://freegeoip.net/tz/json/US/10
    curl http://freegeoip.net/tz/json/CA/10?callback=doit


Running
=======

There is a wrapper script ``freegeoip-server`` to start the server. It is also used as a configuration file for basic settings like the port number and network interface to listen on, and the path to the local version of the geoip database.

On some systems, it's required to set the environment variable PYTHONPATH to the directory where ``freegeoip`` is::

    cd /opt/freegeoip
    export PYTHONPATH=`pwd`
    ./freegeoip-server


Requirements
------------

- `Python <http://python.org/>`_ 2.5 or newer (but not 3.x)
- `SQLite3 <http://www.sqlite.org/>`_ (usually ships with Python)
- `Twisted <http://twistedmatrix.com/trac/>`_ 8.2 or newer
- `Cyclone <http://github.com/fiorix/cyclone/>`_

From the Command Line
---------------------

I usually use the following on ~/.bash_profile or ~/.bashrc in order to easily use geoip from the Unix command line::

    # geoip
    function geoip_curl_xml {
        curl -D - http://freegeoip.net/xml/$1
    }
    alias geoip=geoip_curl_xml


Credits
=======

Thanks to (in no particular order):

- ipinfodb.com

    - For providing both GeoIP and Timezones database

- maxmind.com

    - For creating the database
