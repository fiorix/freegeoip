package httplog

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-web/httpmux"
)

// DefaultFormat returns a middleware that logs http requests
// to the given logger using the default log format.
func DefaultFormat(l *log.Logger) httpmux.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := NewResponseWriter(w)
			next(rw, r)
			b := getBuffer()
			b.WriteString(r.Proto)
			b.Write([]byte(" "))
			b.WriteString(strconv.Itoa(rw.Code()))
			b.Write([]byte(" "))
			b.WriteString(r.Method)
			b.Write([]byte(" "))
			b.WriteString(r.URL.RequestURI())
			b.Write([]byte(" from "))
			b.WriteString(r.RemoteAddr)
			b.Write([]byte(" "))
			fmt.Fprintf(b, "%q", r.Header.Get("User-Agent"))
			b.Write([]byte(" "))
			b.WriteString(strconv.Itoa(rw.Bytes()))
			b.Write([]byte(" bytes in "))
			b.WriteString(time.Since(start).String())
			if err := httpmux.Context(r).Value(ErrorID); err != nil {
				fmt.Fprintf(b, " err: %v", err)
			}
			l.Print(b.String())
			putBuffer(b)
		}
	}
}
