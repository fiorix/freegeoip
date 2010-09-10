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
#
# download ip_group_city.csv from:
# http://ipinfodb.com/download.php?file=ipinfodb_one_table_full.csv.zip

import simplejson
import os, sys, csv, sqlite3

dbname = "ipdb.sqlite"

def insert(conn, curs, data):
    curs.executemany('INSERT INTO ip_group_city VALUES (?, ?)', data)
    conn.commit()

if __name__ == '__main__':
    try:
        fd = open("ip_group_city.csv")
    except:
        print "download ip_group_city.csv from"
        print "http://ipinfodb.com/download.php?file=ipinfodb_one_table_full.csv.zip"
        sys.exit(1)

    if os.path.exists(dbname):
        os.unlink(dbname)

    conn = sqlite3.connect(dbname)
    curs = conn.cursor()
    curs.execute("CREATE TABLE ip_group_city (ip_start INTEGER PRIMARY KEY, data TEXT)")

    count = 0
    entities = []

        
    reader = csv.reader(fd, delimiter=";", quotechar='"')
    headers = reader.next()

    sys.stdout.write("importing data into %s" % dbname)
    sys.stdout.flush()
    for row in csv.reader(fd, delimiter=';', quotechar='"'):
        data = {}
        for k, v in zip(headers[1:], row[1:]):
            data[k] = v

        entities.append([row[0], simplejson.dumps(data)])

        count += 1
        if count % 10000 == 0:
            insert(conn, curs, entities)
            sys.stdout.write(".")
            sys.stdout.flush()
            entities = []
    
    if entities:
        insert(conn, curs, entities)

    print "\n%s records imported" % count
    conn.close()
