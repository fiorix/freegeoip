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

import os.path
import cyclone.web

import freegeoip.geoip
import freegeoip.timezone

import sqlite3

class InlineSQLite:
    def __init__(self, filename):
        self.conn = sqlite3.connect(filename)
        self.curs = self.conn.cursor()

    def runQuery(self, query, *args, **kwargs):
        self.curs.execute(query, *args, **kwargs)
        return [row for row in self.curs]


class Application(cyclone.web.Application):
    def __init__(self, xheaders, database):
        db = InlineSQLite(database)

        tzre = r"([A-Z]{,2})/([0-9A-Z]{,2})?"

        handlers = [
            # static content
            (r"/", cyclone.web.RedirectHandler, {"url":"/static/index.html"}),

            # geoip queries
            (r"/csv/(.*)",  freegeoip.geoip.CsvHandler),
            (r"/xml/(.*)",  freegeoip.geoip.XmlHandler),
            (r"/json/(.*)", freegeoip.geoip.JsonHandler),

            # timezone queries
            (r"/tz/csv/"+tzre,  freegeoip.timezone.CsvHandler),
            (r"/tz/xml/"+tzre,  freegeoip.timezone.XmlHandler),
            (r"/tz/json/"+tzre, freegeoip.timezone.JsonHandler),
        ]

        cwd = os.path.dirname(os.path.dirname(__file__))
        settings = {
            "db": db,
            "xheaders": xheaders,
            "static_path": os.path.join(cwd, "files", "static"),
            "template_path": os.path.join(cwd, "files", "template")
        }

        cyclone.web.Application.__init__(self, handlers, **settings)
