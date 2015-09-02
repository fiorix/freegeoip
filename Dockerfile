FROM golang:1.5

ADD . /go/src/github.com/fiorix/freegeoip
WORKDIR /go/src/github.com/fiorix/freegeoip/cmd/freegeoip

RUN go get
RUN go install

# Init public web application
RUN cp -r public /var/www

ENTRYPOINT ["/go/bin/freegeoip"]

# CMD instructions:
#   Add     "-use-x-forwarded-for"  if your image is proxied by an HTTP load balancer
#   Add     "-public", "/var/www"   to enable the web application
#
#   Full example:   CMD ["-use-x-forwarded-for", "-public", "/var/www"]
