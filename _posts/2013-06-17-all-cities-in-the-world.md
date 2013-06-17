---
layout: post
title: All cities in the world
user: fiorix
---

Earlier this year the database of <http://freegeoip.net> was switched back
to [http://dev.maxmind.com/geoip/legacy/geolite](MaxMind GeoLite) and it
required a complete rewrite of the
[https://github.com/fiorix/freegeoip/blob/master/db/updatedb](updatedb)
script. It's the script that automatically downloads a bunch of files from
multiple sources, process and combine them to generate the database file used
by freegeoip.

One of the files downloaded in the process (of generating the final db) is
<http://dev.maxmind.com/static/csv/codes/maxmind/region.csv>, which is a list
of regions (states or provinces) by country. Turns out this list is formatted
as ASCII and therefore lacks accented characters in region names.

I wanted to fix it and after some searching found
[this post](<http://answers.google.com/answers/threadview/id/774429.html>),
in which there's an actual list of all cities, towns, etc. Some
[python-fu](https://gist.github.com/fiorix/4592774) filtered out only the
relevant info, and the output was a CSV with only three columns: country,
region and city.

Since then, our database has proper names for all regions and cities.

The list of all cities in the world was previously hosted somewhere else,
but from now on is hosted on the blog, and the *updatedb* script will fetch it
from here for building the database.

Download: <http://blog.freegeoip.net/files/all_cities_in_the_world.zip>
