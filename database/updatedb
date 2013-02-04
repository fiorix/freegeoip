#!/usr/bin/env python
# coding: utf-8


import csv
import urllib
import os
import sqlite3
import sys
import unicodedata
import zipfile

from HTMLParser import HTMLParser

dbname = "ipdb.sqlite"


class ParseFiles(HTMLParser):
    def __init__(self):
        HTMLParser.__init__(self)
        self._parse = False
        self.files = []

    def handle_starttag(self, tag, attrs):
        if tag == "a":
            self._parse = True

    def handle_endtag(self, tag):
        if self._parse is True:
            self._parse = False

    def handle_data(self, data):
        if self._parse is True and data[-4:] == ".zip":
            self.files.append(data)


def download(url, filename=None):
    print "Downloading " + url

    req = urllib.urlopen(url)
    data = req.read()
    req.close()

    if filename:
        with open(filename, "w") as fd:
            fd.write(data)

    return data


def extract(zipfd, zipname, filename=None):
    if filename is None:
        filename = zipname

    print "Extracting " + zipname + " -> " + filename

    try:
        data = zipfd.read(zipname)
    except KeyError:
        print "Could not extract %s from zip archive." % zipname
        sys.exit(1)
    else:
        with open(filename, "w") as fd:
            fd.write(data)


def import_csv(cursor, csvfile, table, skip_lines=0):
    sys.stdout.write("Importing %s: " % csvfile)
    sys.stdout.flush()

    fd = open(csvfile)
    for n in range(skip_lines):
        fd.next()

    q = None
    rows = []
    for n, row in enumerate(csv.reader(fd), 1):
        if q is None:
            question_marks = ",".join(["?"] * len(row))
            q = "insert into %s values (%s)" % (table, question_marks)

        rows.append(map(lambda s: s.decode("latin-1"), row))
        if not n % 100000:
            sys.stderr.write(".")
            cursor.executemany(q, rows)
            rows = []

    if rows:
        curs.executemany(q, rows)

    fd.close()
    print "%d records!" % n


class world_regions(dict):
    """Imports a csv and only store rows that contains accented characters,
    indexing them by their non-accented version::

        country,region (no accents) -> region with accents

    Expected csv columns: country,region,city
    """
    def __init__(self, filename=None):
        self.filename = filename
        if filename:
            with open(filename) as fd:
                for row in csv.reader(fd):
                    v = map(lambda s: s.decode("utf-8"), row[:2])
                    k = self.strip_accents(",".join(v))

                    if k != v:
                        self[k] = v[1]

    def strip_accents(self, s):
        return ''.join((c for c in unicodedata.normalize('NFD', s)
                          if unicodedata.category(c) != 'Mn'))


class world_countries(dict):
    def __init__(self, conn):
        curs = conn.cursor()
        curs.execute("SELECT country_code, country_name from country_blocks")
        for (code, name) in curs:
            self[code] = name
        curs.close()


if __name__ == "__main__":
    region_url = "http://dev.maxmind.com/static/csv/codes/maxmind/region.csv"
    region_csv = os.path.basename(region_url)
    if not os.path.exists(region_csv):
        download(region_url, region_csv)

    wr_url = "http://musta.sh/files/all_cities_in_the_world.csv.zip"
    wr_zip = os.path.basename(wr_url)
    wr_csv = wr_zip[:-4]
    if not os.path.exists(wr_csv):
        if not os.path.exists(wr_zip):
            download(wr_url, wr_zip)
        with zipfile.ZipFile(wr_zip) as zipfd:
            extract(zipfd, wr_csv, wr_csv)

    geolite = "http://geolite.maxmind.com/download/geoip/database/"

    city_url = geolite + "GeoLiteCity_CSV/"
    city_files = ["GeoLiteCity-Blocks.csv", "GeoLiteCity-Location.csv"]
    if not all(map(os.path.exists, city_files)):
        print "Checking " + city_url
        city_html = download(city_url)
        parser = ParseFiles()
        parser.feed(city_html)

        if not len(parser.files):
            print "Could not find any databases:\n" + city_html
            sys.exit(1)

        parser.files.sort(reverse=True)

        # Fetch the most recent city database
        city_zip = os.path.basename(parser.files[0])  # hax0rs foff
        if not os.path.exists(city_zip):
            download(city_url + city_zip, city_zip)

        # Extract city csv files
        zipfd = zipfile.ZipFile(city_zip)
        for filename in city_files:
            if not os.path.exists(filename):
                extract(zipfd, os.path.join(city_zip[:-4], filename), filename)
        zipfd.close()

    # Fetch the country database
    country_url = geolite + "GeoIPCountryCSV.zip"
    country_zip = os.path.basename(country_url)
    if not os.path.exists(country_zip):
        download(country_url, country_zip)

    country_csv = "GeoIPCountryWhois.csv"
    if not os.path.exists(country_csv):
        with zipfile.ZipFile(country_zip) as zipfd:
            extract(zipfd, country_csv)

    # Create the IP database
    tmpdb = "_" + dbname + ".temp"
    if os.path.exists(tmpdb):
        os.unlink(tmpdb)

    conn = sqlite3.connect(tmpdb)
    curs = conn.cursor()

    curs.execute("""\
    create table country_blocks (
        ip_start_str text,
        ip_end_str text,
        ip_start text,
        ip_end text,
        country_code text,
        country_name text,
        primary key(ip_start))""")
    import_csv(curs, country_csv, "country_blocks")
    curs.execute("CREATE INDEX cc_idx ON country_blocks(country_code);")

    curs.execute("""\
    create table region_names (
        country_code text,
        region_code text,
        region_name text,
        unique (country_code, region_code))""")
    import_csv(curs, region_csv, "region_names")

    curs.execute("""\
    create table city_blocks (
        ip_start int,
        ip_end int,
        loc_id int,
        primary key(ip_start))""")
    import_csv(curs, city_files[0], "city_blocks", skip_lines=2)

    curs.execute("""\
    create table city_location (
        loc_id int,
        country_code text,
        region_code text,
        city_name text,
        postal_code text,
        latitude real,
        longitude real,
        metro_code text,
        area_code text,
        primary key(loc_id))""")
    import_csv(curs, city_files[1], "city_location", skip_lines=2)

    curs.close()
    conn.commit()

    # Fix region names
    sys.stdout.write("Updating region names: ")
    sys.stdout.flush()

    world_regions = world_regions("all_cities_in_the_world.csv")
    world_countries = world_countries(conn)

    count = 0
    regions = conn.cursor()
    regions.execute("SELECT rowid, * FROM region_names")

    for region in regions:
        region_name = region[-1]  # rowid,country_code,region_code,region_name
        country_name = world_countries.get(region[1])

        if country_name:
            k = country_name + "," + region_name
            if k in world_regions:
                new_name = world_regions[k]
                if region_name != new_name:
                    update = conn.cursor()
                    update.execute("UPDATE region_names SET region_name=? "
                                   "WHERE rowid=?", (new_name, region[0]))
                    update.close()
                    count += 1

    print "%d names updated." % count
    regions.close()
    conn.commit()
    conn.close()

    # Replace any existing db with the new one
    if os.path.exists(dbname):
        os.unlink(dbname)
    os.rename(tmpdb, dbname)
