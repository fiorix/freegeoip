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

import cyclone.web
import cyclone.escape
from twisted.python import log
from twisted.internet import defer

import freegeoip.search

class BaseHandler(cyclone.web.RequestHandler):
    @defer.inlineCallbacks
    def get(self, address):
        try:
            ip, data = yield freegeoip.search.geoip(self.settings.db,
                address or self.request.remote_ip)

            if data:
                data = cyclone.escape.json_decode(data[0][0])
                data["ip"] = ip

        except ValueError, e:
            raise cyclone.web.HTTPError(400)

        except Exception, e:
            log.err("search.geoip('%s') failed: %s" % (address, e))
            raise cyclone.web.HTTPError(503)

        if data:
            self.dump(data)
        else:
            raise cyclone.web.HTTPError(404)

    def dump(self, data):
        raise NotImplementedError


class CsvHandler(BaseHandler):
    def dump(self, data):
        self.set_header("Content-Type", "text/csv")
        self.render("geoip.csv", data=data)

class XmlHandler(BaseHandler):
    def dump(self, data):
        self.set_header("Content-Type", "text/xml")
        self.render("geoip.xml", data=data)

class JsonHandler(BaseHandler):
    def dump(self, data):
        callback = self.get_argument("callback", None)
        self.set_header("Content-Type", "application/json")
        if callback:
            self.finish("%s(%s);" % (callback, cyclone.escape.json_encode(data)))
        else:
            self.finish(cyclone.escape.json_encode(data))
