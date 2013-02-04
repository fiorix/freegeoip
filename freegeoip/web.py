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

import cyclone.web

from freegeoip import API
from freegeoip import config
from freegeoip.storage import DatabaseMixin


class Application(cyclone.web.Application):
    def __init__(self, config_file):
        conf = config.parse_config(config_file)
        handlers = [
            # URIs
            (r"/", cyclone.web.RedirectHandler, {"url":"/static/index.html"}),
            (r"/(crossdomain.xml)", cyclone.web.StaticFileHandler,
                                        dict(path=conf["static_path"])),

            # API
            (r"/(csv|xml|json)/(.*)", API.IpLookupHandler),
        ]

        DatabaseMixin.setup(conf)
        cyclone.web.Application.__init__(self, handlers, **conf)
