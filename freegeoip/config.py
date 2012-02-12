# coding: utf-8

import ConfigParser
import os
from cyclone.util import ObjectDict

def xget(func, section, option, default=None):
    try:
        return func(section, option)
    except:
        return default


def parse_config(filename):
    cfg = ConfigParser.RawConfigParser()
    with open(filename) as fp:
        cfg.readfp(fp)

    settings = {}

    # web server settings
    settings["debug"] = xget(cfg.getboolean, "server", "debug", False)
    settings["xheaders"] = xget(cfg.getboolean, "server", "xheaders", False)
    settings["xsrf_cookies"] = xget(cfg.getboolean, "server", "xsrf_cookies", False)
    settings["cookie_secret"] = cfg.get("server", "cookie_secret")

    # service limits
    settings["expire"] = xget(cfg.getint, "limits", "expire", 3600)
    settings["max_requests"] = xget(cfg.getint, "limits", "max_requests", 1000)

    # get project's absolute path
    root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    getpath = lambda k, v: os.path.join(root, xget(cfg.get, k, v))

    # locale, template and static directories' path
    settings["static_path"] = getpath("frontend", "static_path")
    settings["template_path"] = getpath("frontend", "template_path")

    # sqlite support
    if xget(cfg.getboolean, "sqlite", "enabled", False):
        settings["sqlite_settings"] = ObjectDict(
            database=getpath("sqlite", "database")
        )
    else:
        settings["sqlite_settings"] = None

    # redis support
    if xget(cfg.getboolean, "redis", "enabled", False):
        settings["redis_settings"] = ObjectDict(
            host=cfg.get("redis", "host"),
            port=cfg.getint("redis", "port"),
            dbid=cfg.getint("redis", "dbid"),
            poolsize=cfg.getint("redis", "poolsize"),
        )
    else:
        settings["redis_settings"] = None

    # it must always return a dict
    return settings
