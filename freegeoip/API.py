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
import types

from xml.sax.saxutils import escape as xml_escape
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
                "WHERE city_blocks.ip_start <= ? "
                "ORDER BY city_blocks.ip_start DESC LIMIT 1", (ip,))

            if rs:
                rs = rs[0]
            else:
                raise cyclone.web.HTTPError(404)

        self.set_header("Access-Control-Allow-Origin", "*")

        if fmt == "csv":
            self.set_header("Content-Type", "text/csv")
            rs = (address,) + rs
            self.finish(",".join(map(lambda s: '"%s"' %
                        unicode(s).encode("utf-8") if s else "", rs)) + "\n")

        elif fmt == "xml":
            self.set_header("Content-Type", "text/xml")
            rs = map(lambda s: xml_escape(s)
                     if isinstance(s, types.StringTypes) else s,
                     ((address,) + rs))
            self.finish("""<?xml version="1.0" encoding="UTF-8"?>\n"""
                        "<Response>\n"
                        "  <Ip>%s</Ip>\n"
                        "  <CountryCode>%s</CountryCode>\n"
                        "  <CountryName>%s</CountryName>\n"
                        "  <RegionCode>%s</RegionCode>\n"
                        "  <RegionName>%s</RegionName>\n"
                        "  <City>%s</City>\n"
                        "  <ZipCode>%s</ZipCode>\n"
                        "  <Latitude>%s</Latitude>\n"
                        "  <Longitude>%s</Longitude>\n"
                        "  <MetroCode>%s</MetroCode>\n"
                        "  <AreaCode>%s</AreaCode>\n"
                        "</Response>\n" % tuple(rs))

        elif fmt == "json":
            json_data = cyclone.escape.json_encode({
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
            })

            callback = self.get_argument("callback", None)
            if callback:
                self.set_header("Content-Type", "text/javascript")
                self.finish("%s(%s);\n" % (callback, json_data))
            else:
                self.set_header("Content-Type", "application/json")
                self.finish(json_data + "\n")
