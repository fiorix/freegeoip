FROM golang:1.7

COPY cmd/freegeoip/public /var/www

ADD . /go/src/github.com/fiorix/freegeoip
RUN cd /go/src/github.com/fiorix/freegeoip/cmd/freegeoip && go get && go install

ENTRYPOINT ["/go/bin/freegeoip"]

EXPOSE 8080

# CMD instructions:
# Add  "-use-x-forwarded-for"      if your server is behind a reverse proxy
# Add  "-public", "/var/www"       to enable the web front-end
# Add  "-internal-server", "8888"  to enable the pprof+metrics server
#
# Example:
# CMD ["-use-x-forwarded-for", "-public", "/var/www", "-internal-server", "8888"]
