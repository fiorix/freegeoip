---
layout: post
title: New config file
user: fiorix
---

Since freegeoip was born back in 2008, one thing is intrinsic about it:
*simplicity*.

Simplicity is in fact one of its major attributes and the reason why so many
people like and use it. Not only the simplicity is in the API itself, but also
can be seen in the server code, and obviously the main page, public site.

For that reason, the server has never had a configuration file and all of its
settings were constants at the very top of the code so anyone could easily
spot them and adjust for their needs.
However, to support the upcoming changes in the infrastructure it is now
required to have a configuration file in order to simplify the deployment in
multiple servers. I didn't really want it, but there's no other way.

Furthermore, Go won't help on that matter too. Albeit the standard library
has pretty much all we really need, it still lacks a *standard* configuration
file format, such as Python's
[ConfigParser](http://docs.python.org/2/library/configparser.html) and
apparently there are no plans to have one.
There's obviously other options by using 3rd party libraries, but that would
introduce more dependencies. Writing my own would be an option, but no. I've
passed that phase already.

Guess what's left? Effing XML. If you only knew how much I hate XML in general,
and speficically for configuration files... The fact that it requires a minimum
of 7 characters to comment a line (9 with spaces) makes no sense at all.
Anyway, that's what's on the table for now so let's focus on functionality.

Here's the default `freegeoip.conf` that ships with the server:

{% highlight xml %}
<?xml version="1.0" encoding="UTF-8"?>
<Server debug="true" xheaders="false" addr=":8080">
	<DocumentRoot>./static</DocumentRoot>
	<IPDB File="./db/ipdb.sqlite" CacheSize="51200"/>
	<Limit MaxRequests="10000" Expire="3600"/>
	<Redis>
		<Addr>127.0.0.1:6379</Addr>
		<!-- Balance between multiple servers: -->
		<!-- <Addr>10.0.0.1:6379</Addr> -->
		<!-- <Addr>10.0.0.2:6379</Addr> -->
		<!-- Or use unix socket: -->
		<!-- <Addr>/tmp/redis.sock</Addr> -->
		<!-- <Addr>/var/run/redis/redis.sock</Addr> -->
	</Redis>
</Server>
{% endhighlight %}

Now we can change settings without having to recompile the server and support
the upcoming multi-server architecture. Also, it's possible balance quota usage
between multiple redis servers by just adding more &lt;Addr&gt; tags under
the &lt;Redis&gt; config.

One new and very important feature that comes with the addition of the config
file is the ability to configure the cache size for the IP database. Previously
it would just use the default, which is not bad however not optimized.

A quick intro to how caching works in SQLite (the IP database is an SQLite):
when an SQLite file is created it has a default page size (in our case 4096),
and the cache size is the number of pages that SQLite will hold in memory.
Check out the [PRAGMA cache_size](http://www.sqlite.org/pragma.html#pragma_cache_size)
for more details.

The default configuration of 51200 holds up to 200MB of cache in memory.

It's now ready to roll out.
