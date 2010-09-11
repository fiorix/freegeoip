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

import struct, socket
from twisted.internet import defer, threads

def gethostbyname(hostname):
    return threads.deferToThread(socket.gethostbyname, hostname)

def ip2uint32(address):
    return struct.unpack("!I", socket.inet_aton(address))[0]

@defer.inlineCallbacks
def geoip(db, address):
    if len(address) > 256:
        raise ValueError

    try:
        ip = ip2uint32(address)
    except:
        try:
            address = yield gethostbyname(address)
            ip = ip2uint32(address)
        except:
            raise ValueError

    result = yield db.runQuery("""
        SELECT * FROM ip_group_city 
        WHERE ip_start < ? 
        ORDER BY ip_start DESC LIMIT 1""", (ip,))

    defer.returnValue((address, result))

def timezone(db, country_code, region_code):
    return db.runQuery("""
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
