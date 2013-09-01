---
layout: post
title: New server, transition complete
user: fiorix
---

Last night I started getting alarm emails from [mongu.ru](https://mongu.ru),
the service that monitors [freegeoip](http://freegeoip.net) servers, as well
as from some users. Service was in critical state and responding very
slow because the load balancer couldn't keep up with the traffic due to a
recent change _on the Internet_.

<div class="row-fluid">
  <div class="span12 pagination-centered">
    <img src="/img/traffic_monitor.jpg" alt="">
  </div>
</div>
<br>

About two weeks ago one of the most popular Wordpress plugins, the
[AdRotate](http://www.adrotateplugin.com) has switched its default
geolocation provider from [geoPlugin](http://www.geoplugin.com) to us. As a
result, our servers were flooded with new requests and the poor load balancer
cried loud becoming a bottleneck for the service.

For the past 6 months [freegeoip](http://freegeoip.net) has been serving
an average of 70 million queries per day. In just two weeks it went from that
to 220 million per day! Good thing is that I was contacted by Arnan, the
author of [AdRotate](http://www.adrotateplugin.com) plugin before the change
so I could keep an eye on it. I'd be very puzzled otherwise.

The old load balancer was a cheap $5/mo droplet from
[DigitalOcean](https://www.digitalocean.com), spraying requests to 3 workers
of the same type running the
[freegeoip software](https://github.com/fiorix/freegeoip). Now it's a much
better server, the $20/mo one with 2GB of RAM and 2 CPU cores. It's expected
to handle the traffic for the next months, at least.

I would like to thank all donators that help me keep this service up and
running, because it's _our_ service not mine. I just happen to write the
software and run it for us.
