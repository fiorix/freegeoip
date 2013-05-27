---
layout: post
title: More servers, more capacity
---

Today is a very important day for the freegeoip: it's the day we doubled the
capacity of the service, once again.

It's worth to note that earlier this year the code base has suffered drastic
changes, including a complete rewrite of the server from
[Python](http://python.org) to [Go](http://golang.org), and a switch from plain
[jQuery](http://jquery.org) to [AngularJS](http://angularjs.org) on the main
page, as well as a redesign of the main page itself. Also, the service has been
moved to a new virtual machine on [DigitalOcean](https://www.digitalocean.com),
to leverage Linux running on SSD and faster response times.

This is all part of undergoing experiments with this enthusiastic language,
Go, and the amazing AngularJS. With this change, the service went from a limit
of 1,000 queries per hour to 5,000 and a few weeks later with little
adjustments, to the current limit of **10,000** queries per hour!

In other words, this service has been constantly improved.

One of the main reasons to maintain this service is because it allows me to
exercise some of my programming skills with no rules, no limits, no bullshit.
If it's got to change, it will change, no matter how much effort and
dedication it takes.

The only real compromise is with the community that use and support the
service, and love it! At least that's what I can conclude from the number of
feedback emails and donations from people all over the world. I won't let you
down, it's a promise.

As proof, all recent donations have turned into two brand new servers and
a much better load balancer. The roll out was very smooth and if things go
well there should be yet another increase in the limit of queries per hour
soon.

SSL is in the future plans, too.

Thanks everyone for your support, it's a pleasure to maintain this free service.
