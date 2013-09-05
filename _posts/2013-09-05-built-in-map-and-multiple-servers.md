---
layout: post
title: Built-in map and multiple servers
user: fiorix
---

Due to recent issues with redis management I decided to experiment with a
built-in map for managing quota. It's a [Go map](http://golang.org/doc/effective_go.html#maps)
that uses the source IP of the connection as the key, and its value is the
number of hits on the API.

The map implementation has reduced request processing times from an average
of 800μs down to 300μs in production.

There's one more optimization that I'd like to implement, which is what was
used in one of the latest Python versions of freegeoip: convert the IP
to int32 and use that as the map key to save some memory. The IP is being
converted already for the SQLite query anyway, and is just a matter of
reorganizing the code.

For now, redis is optional and can be enabled or disabled any time in the
configuration file.

Another new feature that has been added to the server is the ability to listen
on multiple HTTP and HTTPS servers on either tcp or unix socket. Now the
configuration file supports multiple ``<Listen>`` tags, each with
individual settings for logging and use of xheaders.

If a ``<Listen>`` tag has a pair of ``<CertFile>`` and ``<KeyFile>`` tags
then HTTPS is enabled.
