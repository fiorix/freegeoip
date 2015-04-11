---
layout: post
title: Enabled IPv6
user: fiorix
---

<div class="row-fluid">
  <div class="span12 pagination-centered">
    <img src="/img/ipv6.png" alt="">
  </div>
</div>
<br>

This is a long due change that many people wanted. Today I finally enabled
IPv6 support for [freegeoip.net](https://freegeoip.net).

Although this may sound like a simple change, it actually required a
combination of things in order to make this happen. First, the IP database
needs to support IPv6, and this was check marked a while ago when the
database was migrated to the binary version of
[GeoLite2](http://dev.maxmind.com/geoip/geoip2/geolite2/).

Besides the database, there's also the network setup. Because freegeoip.net
runs on a little cluster at [Digital Ocean](https://www.digitalocean.com),
each virtual machine needed IPv6. Finally, those IPs were added to the CDN
controller that protects (and speeds up) the site and HTTP API,
[Cloud Flare](https://www.cloudflare.com/), to enable full IPv6 support.

So if you're on v6 already, the automatic query when you access the website
or query the API with `curl freegeoip.net/json/` works for you now. For
those using freegeoip.net on their website to query for their visitor's IP,
it now works when your visitors are coming from IPv6.

Enjoy!
