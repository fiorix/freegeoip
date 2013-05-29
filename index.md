---
layout: default
title: freegeoip blog
---

<h1>freegeoip blog</h1>
<hr>
<ul class="unstyled">
  {% for post in site.posts %}
    <li>
      <span class="muted date"><small>{{post.date|date_to_string}}</small></span>
      &middot; <a href="{{post.url}}">{{post.title}}</a>
    </li>
  {% endfor %}
</ul>

<hr>
<p>
<a href="http://freegeoip.net">freegeoip.net</a> &middot;
<a href="https://github.com/fiorix/freegeoip">source</a>
</p>
