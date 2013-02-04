# coding: utf-8
#
# Copyright 2009-2013 Alexandre Fiori
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


import cyclone.escape
import cyclone.web
import socket

from twisted.internet import defer
from twisted.internet import threads
#from twisted.names.client import getHostByName

from freegeoip.storage import DatabaseMixin
from freegeoip.utils import CheckQuota
from freegeoip.utils import ReservedIPs
from freegeoip.utils import ip2uint32


def gethostbyname(hostname):
    #return getHostByName(hostname)
    return threads.deferToThread(socket.gethostbyname, hostname)


class IpLookupHandler(cyclone.web.RequestHandler, DatabaseMixin):
    @CheckQuota
    @defer.inlineCallbacks
    def get(self, fmt, address):
        address = address or self.request.remote_ip
        if len(address) > 256:
            raise cyclone.web.HTTPError(400)

        try:
            ip = ip2uint32(address)
        except:
            try:
                address = yield gethostbyname(address)
                ip = ip2uint32(address)
            except:
                raise cyclone.web.HTTPError(400)

        if ReservedIPs.test(ip) is True:
            rs = (u"RD", u"Reserved", "", "", "", "", "", "", "", "")
        else:
            rs = self.sqlite.runQuery(
                "SELECT "
                "  city_location.country_code, country_blocks.country_name, "
                "  city_location.region_code, region_names.region_name, "
                "  city_location.city_name, city_location.postal_code, "
                "  city_location.latitude, city_location.longitude, "
                "  city_location.metro_code, city_location.area_code "
                "FROM city_blocks "
                "  NATURAL JOIN city_location "
                "  INNER JOIN country_blocks ON "
                "    city_location.country_code = country_blocks.country_code "
                "  INNER JOIN region_names ON "
                "    city_location.country_code = region_names.country_code "
                "    AND "
                "    city_location.region_code = region_names.region_code "
                "WHERE city_blocks.ip_start < ? "
                "ORDER BY city_blocks.ip_start DESC LIMIT 1", (ip,))

            if rs:
                rs = rs[0]
            else:
                raise cyclone.web.HTTPError(404)

        data = {
            "ip": address,
            "country_code": rs[0],
            "country_name": rs[1],
            "region_code": rs[2],
            "region_name": rs[3],
            "city": rs[4],
            "zipcode": rs[5],
            "latitude": rs[6],
            "longitude": rs[7],
            "metrocode": rs[8],
            "areacode": rs[9]
        }

        self.set_header("Access-Control-Allow-Origin", "*")

        if fmt in ("csv", "xml"):
            self.set_header("Content-Type", "text/%s" % fmt)
            self.render("geoip.%s" % fmt, data=data)
        else:
            json_data = cyclone.escape.json_encode(data)
            callback = self.get_argument("callback", None)
            if callback:
                self.set_header("Content-Type", "text/javascript")
                self.finish("%s(%s);" % (callback, json_data))
            else:
                self.set_header("Content-Type", "application/json")
                self.finish(json_data)
