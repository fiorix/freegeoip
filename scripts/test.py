#!/usr/bin/env python
# coding: utf-8
#
# on osx:
# launchctl limit maxfiles 10240 10240
#
# edit freegeoip.conf and increase max_requests
# start freegeoip server:
# $ twistd --reactor=cf -n freegeoip
#
# start the test:
# $ ./test -n 100000 -c 1000
#
#from twisted.internet import cfreactor
#cfreactor.install()

import json
import random
import sys
import time
from cyclone.httpclient import fetch
from twisted.internet import defer
from twisted.internet import reactor
from twisted.internet import task
from twisted.python import usage

humanreadable = lambda s:[(s%1024**i and "%.1f"%(s/1024.0**i) or \
                          str(s/1024**i))+x.strip()+"B" \
                          for i,x in enumerate(' KMGTPEZY') \
                          if s<1024**(i+1) or i==8][0]

class report(object):
    requests = 0
    show_errmsg = False

    total_body = 0
    total_bytes = 0
    total_errors = 0
    total_headers = 0
    total_requests = 0

    bps = 0
    rps = 0
    last = None
    start = None
    last_err = None

    @classmethod
    def update(self, response=None):
        self.rps += 1
        self.total_requests += 1

        if hasattr(response, "body"):
            bodylen = len(response.body)
            self.total_body += bodylen
            headerslen = sum(map(lambda (k, v): len(k)+len(v)+4, # 4=: \r\n
                             response.headers.items())) + 2 # \r\n
            self.total_headers += headerslen

            reqlen = (bodylen + headerslen)
            self.bps += reqlen
            self.total_bytes += reqlen

            if response.code != 200:
                self.total_errors += 1
        else:
            self.total_errors += 1
            self.last_err = response

        now = int(time.time())
        if self.last is None:
            self.last = now
        if self.start is None:
            self.start = now
        if self.last < now:
            pct = self.total_requests*100/self.requests
            print "% 3s%% % 4d req/s @ % 8s/s" % \
                  (pct, self.rps, humanreadable(self.bps))
            if self.show_errmsg and self.last_err:
                print self.last_err.getErrorMessage()
                self.last_err = None
            self.bps = self.rps = 0
            self.last = now

    @classmethod
    def summary(self):
        pct = self.total_requests*100/self.requests
        print "% 3s%% % 4d req/s @ % 8s/s" % \
              (pct, self.rps, humanreadable(self.bps))
        print "--"
        if self.total_errors:
            errpct = self.total_errors*100/self.total_requests
        else:
            errpct = 0
        print "%d requests, %d errors (%d%%)" % \
              (self.total_requests, self.total_errors, errpct)

        hdrpct = self.total_headers*100/self.total_bytes
        bdypct = self.total_body*100/self.total_bytes
        print "%s transferred: %s headers (%d%%), %s body (%d%%)" % \
              (humanreadable(self.total_bytes),
               humanreadable(self.total_headers), hdrpct,
               humanreadable(self.total_body), bdypct)

        elapsed = time.time() - self.start
        total_time = time.strftime("%H:%M:%S", time.gmtime(elapsed))
        avgreq = self.total_requests / elapsed
        avgbps = humanreadable(self.total_bytes / elapsed)
        print "%s to run. avg: %d req/s @ %s/s transfer rate" % \
              (total_time, avgreq, avgbps)


def randrequests(requests):
    for n in xrange(requests):
        ip = ".".join(map(lambda n:str(random.randint(1,254)), xrange(4)))
        d = fetch("http://localhost:8888/json/%s" % ip)
        d.addBoth(report.update)
        yield d

class Options(usage.Options):
    optFlags = [
        ["help", "h", "Show this help."],
        ["errmsg", "e", "Dump some of the error messages during test"],
    ]
    optParameters = [
        ["requests", "n", 1000, "Number of requests", int],
        ["concurrent", "c", 1, "Concurrent requests", int],
    ]

def main():
    config = Options()
    try:
        config.parseOptions()
    except usage.UsageError, errorText:
        print "%s: %s" % (sys.argv[0], errorText)
        print "%s: Try --help for usage details." % sys.argv[0]
        sys.exit(1)

    report.requests = config["requests"]
    report.show_errmsg = config["errmsg"]

    tasks = []
    coop = task.Cooperator()
    work = randrequests(report.requests)
    for n in xrange(config["concurrent"]):
        d = coop.coiterate(work)
        tasks.append(d)

    dl = defer.DeferredList(tasks)
    dl.addCallback(lambda ign: report.summary())
    dl.addCallback(lambda ign: reactor.stop())

if __name__ == "__main__":
    main()
    reactor.run()
