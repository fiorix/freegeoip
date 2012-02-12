# coding: utf-8

import os
import functools
import socket
import struct
import datetime

import cyclone.escape
import cyclone.locale
import cyclone.web

from twisted.internet import defer, threads
from twisted.names.client import getHostByName
from twisted.python import log

from freegeoip.utils import BaseHandler
from freegeoip.utils import DatabaseMixin


def _ip2uint32(address):
    return struct.unpack("!I", socket.inet_aton(address))[0]

def _gethostbyname(hostname):
    #return getHostByName(hostname)
    return threads.deferToThread(socket.gethostbyname, hostname)

def checkQuota(method):
    @defer.inlineCallbacks
    @functools.wraps(method)
    def wrapper(self, *args, **kwargs):
        # quota key: freegeoip:IP:YYYY:MM:DD:HH:MM
        key = "freegeoip:quota:%s:%s" % (self.request.remote_ip,
                datetime.datetime.now().strftime("%Y:%m:%d:%H:%M"))

        try:
            n = yield self.redis.incr(key)
            yield self.redis.expire(key, self.settings.expire)

            if n >= self.settings.max_requests:
                raise cyclone.web.HTTPError(403) # Over quota

        except Exception, e:
            log.msg("Redis errors: key: %s - msg: %s" % (key, str(e)))
            raise cyclone.web.HTTPError(503)

        yield defer.maybeDeferred(method, self, *args, **kwargs)
        defer.returnValue(None)
    return wrapper


class IndexHandler(BaseHandler):
    def get(self):
        self.render("index.html")


class SearchIpHandler(BaseHandler, DatabaseMixin):
    @checkQuota
    @defer.inlineCallbacks
    def get(self, fmt, address):
        address = address or self.request.remote_ip
        if len(address) > 256:
            raise cyclone.web.HTTPError(400)

        try:
            ip = _ip2uint32(address)
        except:
            try:
                address = yield _gethostbyname(address)
                ip = _ip2uint32(address)
            except:
                raise cyclone.web.HTTPError(400)

        rs = self.sqlite.runQuery("""
            SELECT data FROM ip_group_city
            WHERE ip_start < ?
            ORDER BY ip_start DESC LIMIT 1""", (ip,))

        if rs:
            json_data = rs[0][0]
        else:
            raise cyclone.web.HTTPError(404)

        rs = cyclone.escape.json_decode(json_data)
        rs["ip"] = address

        if fmt in ("csv", "xml"):
            self.set_header("Content-Type", "text/%s" % fmt)
            self.render("geoip.%s" % fmt, data=rs)
        else:
            json_data = cyclone.escape.json_encode(rs)
            callback = self.get_argument("callback", None)
            if callback:
                self.set_header("Content-Type", "text/javascript")
                self.finish("%s(%s);" % (callback, json_data))
            else:
                self.finish(json_data)


class SearchTzHandler(BaseHandler, DatabaseMixin):
    @checkQuota
    def get(self, fmt, country_code, region_code):
        try:
            rs = self.sqlite.runQuery("""
                SELECT tzd.gmtoff, tzd.isdst, tz.name
                    FROM timezones_data tzd
                JOIN timezones tz ON tz.id = tzd.timezone
                    WHERE tzd.timezone = (
                        SELECT timezone
                        FROM fips_regions
                        WHERE country_code = ?
                        AND region_code = ? )
                    AND tzd.start < strftime('%s')
                ORDER BY tzd.start DESC LIMIT 1
            """, (country_code, region_code or "00"))
            if rs:
                rs = {"gmtoff":rs[0][0], "isdst":rs[0][1], "timezone":rs[0][2]}
        except Exception, e:
            log.err()
            raise cyclone.web.HTTPError(503)

        if not rs:
            raise cyclone.web.HTTPError(404)

        if fmt in ("csv", "xml"):
            self.set_header("Content-Type", "text/%s" % fmt)
            self.render("timezone.%s" % fmt, data=rs)
        else:
            callback = self.get_argument("callback", None)
            json_data = cyclone.escape.json_encode(rs)
            if callback:
                self.finish("%s(%s);" % (callback, json_data))
            else:
                self.finish(json_data)
