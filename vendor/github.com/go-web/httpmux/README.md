# httpmux

[![GoDoc](https://godoc.org/github.com/go-web/httpmux?status.svg)](http://godoc.org/github.com/go-web/httpmux)
[![GoReportCard](https://goreportcard.com/badge/github.com/go-web/httpmux)](https://goreportcard.com/report/github.com/go-web/httpmux)

httpmux is an http request multiplexer for [Go](https://golang.org) built
on top of the popular [httprouter](https://github.com/julienschmidt/httprouter),
with modern features.

The main motivation is to bring http handlers back to their original
Handler interface as defined by [net/http](https://golang.org/pkg/net/http/)
but leverage the speed and features of httprouter, such as dispatching
handlers by method and handling HTTP 405 automatically.

Another important aspect of httpmux is that it provides request context
for arbitrary data, such as httprouter's URL parameters. Seasoned gophers
migth immediately think this is an overlap of gorilla's
[context](https://github.com/gorilla/context) package, however, their
implementation rely on global variables and mutexes that can cause
contention on heavily loaded systems. We use a different approach, which
was stolen from [httpway](https://github.com/corneldamian/httpway), that
hijacks the http.Request's Body field and replace it with an object that
is an io.ReadCloser but also carries a [net/context](https://godoc.org/golang.org/x/net/context)
object. This works well for middleware that wants to store arbitrary
data in the request, serial, and once it hits your handler, the context
can be passed around to goroutines. It is automatically cleared at the
end of the middleware chain.

There's been discussions for adding net/context to the standard library
but most options require changing or creating a new interface and/or
function signature for http handlers. In httpmux we remain close to
net/http aiming at being more pluggable and composable with existing
code in the wild, and if the Request type ends up getting a Context
[field](https://github.com/golang/go/issues/14660), will be an easy
change in httpmux.

To make contexts more useful, httpmux provides the ability to register and
chain wrapper handlers, middleware. Our implementation is based on blogs
and especially [chi](https://github.com/pressly/chi), but much smaller.

Last but not least, httpmux offers two more features for improving
composability. First, is to configure a global prefix for all handlers
in the multiplexer. This is for cases when you have to run your API
behind a proxy, or mixed with other services, and have to be able to
parse and understand the prefix in your handlers. Second, is to allow
subtrees like [gin](https://github.com/gin-gonic/gin)'s groups, but
in a more composable way. Think of cases where your code is an independent
package that provides an http handler, that is tested and run isolated,
but can be added to a larger API at run time. In chi, this is equivalent
to the Mount function in their router.
