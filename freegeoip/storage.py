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

try:
    sqlite_ok = True
    import cyclone.sqlite
except ImportError, sqlite_err:
    sqlite_ok = False

import cyclone.redis
import functools
import os
import sys

from twisted.internet import defer
from twisted.internet import reactor
from twisted.internet import task
from twisted.python import log


def DatabaseSafe(method):
    @defer.inlineCallbacks
    @functools.wraps(method)
    def run(self, *args, **kwargs):
        try:
            r = yield defer.maybeDeferred(method, self, *args, **kwargs)
        except cyclone.redis.ConnectionError, e:
            m = "redis.ConnectionError: %s" % e
            log.msg(m)
            raise cyclone.web.HTTPError(503, m)  # Service Unavailable
        else:
            defer.returnValue(r)

    return run


class DatabaseMixin(object):
    redis = None
    sqlite = None
    sqlite_info = None

    @classmethod
    def setup(cls, conf):
        if "sqlite_settings" in conf:
            if sqlite_ok:
                DatabaseMixin.sqlite = \
                cyclone.sqlite.InlineSQLite(conf["sqlite_settings"].database)

                t = task.LoopingCall(cls.sqlite_autoreload,
                                     conf["sqlite_settings"].database)
                reactor.callWhenRunning(t.start, conf["sqlite_autoreload"])
            else:
                log.err("SQLite is currently disabled: %s" % sqlite_err)
                sys.exit(1)

        if "redis_settings" in conf:
            if conf["redis_settings"].get("unixsocket"):
                DatabaseMixin.redis = \
                cyclone.redis.lazyUnixConnectionPool(
                              conf["redis_settings"].unixsocket,
                              conf["redis_settings"].dbid,
                              conf["redis_settings"].poolsize)
            else:
                DatabaseMixin.redis = \
                cyclone.redis.lazyConnectionPool(
                              conf["redis_settings"].host,
                              conf["redis_settings"].port,
                              conf["redis_settings"].dbid,
                              conf["redis_settings"].poolsize)

    @classmethod
    def sqlite_autoreload(cls, dbname):
        try:
            current = os.stat(dbname).st_mtime
        except Exception, e:
            log.msg("SQLite autoreload task failed: %s" % e)
            return

        if cls.sqlite_info is None:
            cls.sqlite_info = current
            log.msg("SQLite autoreload task initialized.")
        else:
            if current != cls.sqlite_info:
                try:
                    newdb = \
                    cyclone.sqlite.InlineSQLite(dbname)
                except Exception, e:
                    log.msg("SQLite autoreload failed: %s" % e)
                else:
                    curdb = DatabaseMixin.sqlite
                    DatabaseMixin.sqlite = newdb
                    curdb.close()

                    cls.sqlite_info = current
                    log.msg("SQLite autoreload complete.")
