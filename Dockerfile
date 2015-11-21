FROM golang:1.5

ADD . /go/src/github.com/fiorix/freegeoip
WORKDIR /go/src/github.com/fiorix/freegeoip/cmd/freegeoip

RUN go get
RUN go install

# Init public web application
RUN cp -r public /var/www

ENTRYPOINT ["/go/bin/freegeoip"]

# CMD instructions:
# Add  "-use-x-forwarded-for"      if your server is behind a reverse proxy
# Add  "-public", "/var/www"       to enable the web front-end
# Add  "-internal-server", "8888"  to enable the pprof+metrics server
#
# Example:
# CMD ["-use-x-forwarded-for", "-public", "/var/www", "-internal-server", "8888"]
