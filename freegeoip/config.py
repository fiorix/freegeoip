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

import os
import sys
import ConfigParser

from cyclone.util import ObjectDict


def tryget(func, section, option, default=None):
    try:
        return func(section, option)
    except ConfigParser.NoOptionError:
        return default


def my_parse_config(filename):
    cp = ConfigParser.RawConfigParser()
    cp.read([filename])

    conf = dict(raw=cp, config_file=filename)

    # server settings
    conf["debug"] = tryget(cp.getboolean, "server", "debug", False)
    conf["xheaders"] = tryget(cp.getboolean, "server", "xheaders", False)
    conf["cookie_secret"] = cp.get("server", "cookie_secret")
    conf["xsrf_cookies"] = tryget(cp.getboolean,
                                  "server", "xsrf_cookies", False)

    # make relative path absolute to this file's parent directory
    root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    getpath = lambda k, v: os.path.join(root, tryget(cp.get, k, v))

    # directory with static files
    conf["static_path"] = getpath("frontend", "static_path")

    # sqlite support
    if tryget(cp.getboolean, "sqlite", "enabled", False) is True:
        conf["sqlite_settings"] = \
            ObjectDict(database=getpath("sqlite", "database"))
        conf["sqlite_autoreload"] = \
            tryget(cp.getint, "sqlite", "autoreload_time", 60)
    else:
        raise ValueError("SQLite is mandatory, but is currently disabled "
                            "in freegeoip.conf. Not running.")

    # redis support
    if tryget(cp.getboolean, "redis", "enabled", False) is True:
        conf["redis_settings"] = ObjectDict(
                    unixsocket=tryget(cp.get, "redis", "unixsocket", None),
                    host=tryget(cp.get, "redis", "host", "127.0.0.1"),
                    port=tryget(cp.getint, "redis", "port", 6379),
                    dbid=tryget(cp.getint, "redis", "dbid", 0),
                    poolsize=tryget(cp.getint, "redis", "poolsize", 10))
    else:
        raise ValueError("Redis is mandatory, but is currently disabled "
                            "in freegeoip.conf. Not running.")

    # service limits
    conf["expire"] = tryget(cp.getint, "limits", "expire", 3600)
    conf["max_requests"] = tryget(cp.getint, "limits", "max_requests", 1000)

    return conf


def parse_config(filename):
    try:
        return my_parse_config(filename)
    except Exception, e:
        print("Error parsing %s: %s" % (filename, e))
        sys.exit(1)
