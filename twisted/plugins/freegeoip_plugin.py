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

from zope.interface import implements
from twisted.python import usage
from twisted.plugin import IPlugin
from twisted.application import service, internet

import freegeoip

class Options(usage.Options):
    optFlags = [
        ["xheaders", "x", "set this when running behind nginx"],
    ]

    optParameters = [
        ["database", "d", "database/ipdb.sqlite", "set geoip database"],
        ["port", "p", 8888, "port number to listen on"],
        ["listen", "l", "127.0.0.1", "interface to listen on"],
    ]

class ServiceMaker(object):
    implements(service.IServiceMaker, IPlugin)
    tapname = "freegeoip"
    description = "freegeoip web service"
    options = Options

    def makeService(self, options):
        return internet.TCPServer(int(options["port"]),
            freegeoip.Application(options["xheaders"], options["database"]),
            interface=options["listen"])

serviceMaker = ServiceMaker()
