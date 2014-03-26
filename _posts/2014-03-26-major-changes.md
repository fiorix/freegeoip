---
layout: post
title: Major changes
user: fiorix
---

Over the past months I've been updating the freegeoip server regularly,
aiming at making it simpler, lighter, faster, and easier to maintain.

One of the most significant improvement is that now the server preloads
tables that can be accessed by a single key, such as the list of
countries, regions and cities. This has eliminated the need of a
somewhat [complex](https://github.com/fiorix/freegeoip/commit/88a9408bbb3839b8a9b92329d0b984cd8f12ccf2#diff-321e6beb79b1e27be7f1e16c03db22edL63)
query that would slow down the server a bit.

While adding this and other features the server code became a bit messy and
grew into multiple files. After all, I took some time to review, optimize and
clean it up, and the outcome was impressive.

<div class="row-fluid">
  <div class="span12 pagination-centered">
    <img src="/img/dc2.jpg" alt="">
  </div>
</div>
<br>

Now, not only the server is about 2 to 3 times faster using the same
computational resources, but also the source code is smaller and fits
in a single file again. The major reorganization happened in
[this](https://github.com/fiorix/freegeoip/commit/78eacdf8e4dd5568e963ccd52acaa246ad16e23b)
commit, which was followed by a number of minor subsequent commits with
fixes and improvements.

Another notable change is how the server is run. A while ago I ended up
writing a wrapper to daemonize Go programs,
[go-daemon](https://github.com/fiorix/go-daemon), inspired by the good
old twistd. It is no longer needed because now the freegeoip server can
save its own log file and recycle it on SIGHUP, playing nicely with logrotate.

Also, the freegeoip software now ships with a lua script that can be used by
[wrk](https://github.com/wg/wrk) to stress test and benchmark the server.

Besides the optimizations in the server code, there's one more, which is
related to [#issue 32](https://github.com/fiorix/freegeoip/issues/32)
affecting the public freegeoip service. Previous Comodo SSL certificates
have been replaced by brand new ones from [startssl.com](https://startssl.com).

<div class="row-fluid">
  <div class="span12 pagination-centered">
    <img src="/img/startcom_ssl.png" alt="">
  </div>
</div>
<br>

I'm very hapy with the growth of the freegeoip community as more and more
users are following and starring the [repository](https://github.com/fiorix/freegeoip).

Last but not least, I feel like I'm becoming a more seasoned Go programmer
after almost one year of hard work and a lot of typing.
