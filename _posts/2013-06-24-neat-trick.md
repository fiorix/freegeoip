---
layout: post
title: Neat caching trick
user: lxfontes
---

API usage is limited to 10,000 requests per hour per IP and that should be enough for most cases. However, you might be doing log crunching or even slowing down your batch process due to roundtrip time to FreeGeoIP.

So here is a little trick: Run a local cache !

The API is based on HTTP GET and is cache friendly. By doing this, you shave down precious time:

* No roundtrip to FreeGeoIP servers
* No DB query on FreeGeoIP side
* Those 10,000 requests per hour will become 10,000 unique ip addresses per hour

To demonstrate I'm not just blowing smoke here; using a Ubuntu droplet @ DigitalOcean:

Hitting FreeGeoIP directly:

    $ time curl -s -o /dev/null -i http://freegeoip.net/json/200.200.200.200

    real0m0.071s
    user0m0.008s
    sys0m0.004s


Installing [Squid](http://www.squid-cache.org/):

    sudo apt-get install squid

Forcing requests through Squid and running same query twice:

    $ export http_proxy=http://127.0.0.1:3128
    $ time curl -s -o /dev/null -i http://freegeoip.net/json/200.200.200.200

    real0m0.049s
    user0m0.004s
    sys0m0.004s

    $ time curl -s -o /dev/null -i http://freegeoip.net/json/200.200.200.200

    real0m0.018s
    user0m0.004s
    sys0m0.004s



After first request/response, Squid will reply the request directly from cache. Almost 3x faster :)


