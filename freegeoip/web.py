# coding: utf-8

import cyclone.locale
import cyclone.web

from freegeoip import views
from freegeoip import utils

class Application(cyclone.web.Application):
    def __init__(self, settings):
        ip_re = r"/(csv|xml|json)/(.*)"
        tz_re = r"/tz/(csv|xml|json)/([A-Z]{,2})/([0-9A-Z]{,2})?"
        handlers = [
            (r"/",  cyclone.web.RedirectHandler, {"url":"/static/index.html"}),
            (ip_re, views.SearchIpHandler),
            (tz_re, views.SearchTzHandler),
        ]

        utils.DatabaseMixin().setup(settings)
        cyclone.web.Application.__init__(self, handlers, **settings)
