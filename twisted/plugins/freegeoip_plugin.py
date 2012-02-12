# coding: utf-8

import freegeoip.config
import freegeoip.web

from twisted.python import usage
from twisted.plugin import IPlugin
from twisted.application import service, internet
from zope.interface import implements

class Options(usage.Options):
    optParameters = [
        ["port", "p", 8888, "TCP port to listen on", int],
        ["listen", "l", "127.0.0.1", "Network interface to listen on"],
        ["config", "c", "freegeoip.conf", "Configuration file with server settings"],
    ]

class ServiceMaker(object):
    implements(service.IServiceMaker, IPlugin)
    tapname = "freegeoip"
    description = "cyclone-based web server"
    options = Options

    def makeService(self, options):
        port = options["port"]
        interface = options["listen"]
        settings = freegeoip.config.parse_config(options["config"])
        return internet.TCPServer(port, freegeoip.web.Application(settings),
                                  interface=interface)

serviceMaker = ServiceMaker()
