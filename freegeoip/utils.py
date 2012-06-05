# coding: utf-8

import cyclone.escape
import cyclone.redis
import cyclone.sqlite
import cyclone.web


class BaseHandler(cyclone.web.RequestHandler):
    pass


class DatabaseMixin(object):
    redis = None
    sqlite = None

    def setup(self, settings):
        conf = settings.get("redis_settings")
        if conf:
            DatabaseMixin.redis = cyclone.redis.lazyConnectionPool(
                            host=conf.host, port=conf.port,
                            dbid=conf.dbid, poolsize=conf.poolsize)
        else:
            raise RuntimeError("Redis support is mandatory.")

        conf = settings.get("sqlite_settings")
        if conf:
            DatabaseMixin.sqlite = cyclone.sqlite.InlineSQLite(conf.database)
        else:
            raise RuntimeError("SQLite support is mandatory.")
