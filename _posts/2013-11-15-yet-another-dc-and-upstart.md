---
layout: post
title: Yet another datacentre and upstart
user: fiorix
---

Quick update: now running on a dedicated server at
[DataShack](https://www.datashack.net), and no longer using
[supervisord](http://supervisord.org).

The whole deployment at [Digital Ocean](https://www.digitalocean.com) was
becoming too complex. It was running on a location that didn't have private
LAN between the droplets and therefore used the public internet for the
communication between the load balancer and instances.

That's one of the reasons why I decided to move it somewhere else. For the
same price with much less headache and much more horsepower, now it's on its
own bare metal server.

Another important change is how it's being run. Previously, I used supervisor
to run the server and rotate logs, but it has become too expensive using
about 30% of cpu time just for that.

The new server is currently using [go-daemon](https://github.com/fiorix/go-daemon)
for running the service on Ubuntu's upstart. It takes only 2% of cpu time for
the exact same thing, and logs are rotated by logrotate.

I've updated the project's README file with instructions so others can do the
same: <https://github.com/fiorix/freegeoip#running-with-upstart>
