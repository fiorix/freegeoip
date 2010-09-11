#!/usr/bin/env python
# coding: utf-8
#
# Copyright 2010 Alexandre Fiori
# freegeoip.net
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.

import simplejson
import os, sys, csv, sqlite3

dbname = "ipdb.sqlite"

class import_city:
    def __init__(self, fd, conn, curs):
        count = 0
        entities = []

        reader = csv.reader(fd, delimiter=";", quotechar='"')
        headers = reader.next()

        sys.stdout.write("ip_group_city")
        sys.stdout.flush()
        for row in reader:
            data = {}
            for k, v in zip(headers[1:], row[1:]):
                data[k] = v

            entities.append([row[0], simplejson.dumps(data)])

            count += 1
            if count % 10000 == 0:
                self.insert(conn, curs, entities)
                entities = []
        
        if entities:
            self.insert(conn, curs, entities)

        print("\n%s records imported" % count)

    def insert(self, conn, curs, data):
        curs.executemany("INSERT INTO ip_group_city VALUES (?, ?)", data)
        conn.commit()
        sys.stdout.write(".")
        sys.stdout.flush()


class import_timezone:
    def __init__(self, fd, conn, curs):
        count = 0
        ignore = 0
        entities = []
        insert = None

        sys.stdout.write("tzdata")
        sys.stdout.flush()

        for row in csv.reader(fd, delimiter=";", quotechar='"'):
            if row == ["id", "country_code", "code", "name", "timezone"]:
                self.flush(insert, conn, curs, entities)
                entities = []
                insert = self.insert_fips_regions
                continue

            if row == ["code", "name"]:
                self.flush(insert, conn, curs, entities)
                entities = []
                insert = None
                continue

            if row == ["id", "name"]:
                self.flush(insert, conn, curs, entities)
                entities = []
                insert = self.insert_timezones
                continue

            if row == ["timezone", "start", "gmtoff", "abbreviation", "isdst"]:
                self.flush(insert, conn, curs, entities)
                entities = []
                insert = self.insert_timezones_data
                continue

            if insert is None:
                ignore += 1
                continue

            entities.append(row)

            count += 1
            if count%10000 == 0:
                self.flush(insert, conn, curs, entities)
                entities = []

        print("\n%s records imported, %s ignored, %s total" % (count, ignore, count+ignore))

    def insert_timezones(self, conn, curs, data):
        curs.executemany("INSERT INTO timezones VALUES (?, ?)", data)
        conn.commit()

    def insert_timezones_data(self, conn, curs, data):
        curs.executemany("INSERT INTO timezones_data VALUES (?, ?, ?, ?, ?)", data)
        conn.commit()

    def insert_fips_regions(self, conn, curs, data):
        curs.executemany("INSERT INTO fips_regions VALUES (?, ?, ?, ?, ?)", data)
        conn.commit()

    def flush(self, method, conn, curs, data):
        if callable(method) and data:
            method(conn, curs, data)
            conn.commit()
            sys.stdout.write(".")
            sys.stdout.flush()

if __name__ == '__main__':
    try:
        city = open("ip_group_city.csv")
        timezone = open("tz.csv")
    except:
        print("""
        In order to create the GeoIP database, you have to place two
        files in the current working directory.

        1. ip_group_city.csv
        See http://ipinfodb.com/ip_database.php for details.
        Download from http://ipinfodb.com/download.php?file=ipinfodb_one_table_full.csv.zip

        2. tz.csv
        See http://ipinfodb.com/timezonedatabase.php for details.
        Download from http://mirrors.ipinfodb.com/ipinfodb/timezonedatabase/tz.csv.zip   
        """)
        sys.exit(1)

    if os.path.exists(dbname):
        os.unlink(dbname)

    conn = sqlite3.connect(dbname)
    curs = conn.cursor()
    curs.execute("CREATE TABLE ip_group_city (ip_start INTEGER PRIMARY KEY, data TEXT)")
    curs.execute("CREATE TABLE timezones (id INTEGER PRIMARY KEY, name TEXT)")
    curs.execute("""
        CREATE TABLE timezones_data (
            timezone INTEGER,
            start INTEGER,
            gmtoff INTEGER,
            abbreviation TEXT,
            isdst INTEGER)""")
    curs.execute("""
        CREATE TABLE fips_regions (
            id INTEGER PRIMARY KEY,
            country_code TEXT,
            region_code TEXT,
            name TEXT,
            timezone INTEGER)""")

    print("importing data into %s" % dbname)
    import_timezone(timezone, conn, curs)
    import_city(city, conn, curs)
