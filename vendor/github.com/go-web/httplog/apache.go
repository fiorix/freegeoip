package httplog

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-web/httpmux"
)

func apacheCommonLog(w *ResponseWriter, r *http.Request, start time.Time) *bytes.Buffer {
	var username string
	if r.URL.User != nil {
		username = r.URL.User.Username()
	}
	if username == "" {
		username = "-"
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	b := getBuffer()
	b.WriteString(host)
	b.Write([]byte(" - "))
	b.WriteString(username)
	b.Write([]byte(" ["))
	b.WriteString(start.Format("02/Jan/2006:15:04:05 -0700"))
	b.Write([]byte("] \""))
	b.WriteString(r.Method)
	b.Write([]byte(" "))
	b.WriteString(r.URL.RequestURI())
	b.Write([]byte(" "))
	b.WriteString(r.Proto)
	b.Write([]byte("\" "))
	b.WriteString(strconv.Itoa(w.Code()))
	b.Write([]byte(" "))
	b.WriteString(strconv.Itoa(w.Bytes()))
	return b
}

// ApacheCommonFormat returns a middleware that logs http requests
// to the given logger using the Apache Common log format.
func ApacheCommonFormat(l *log.Logger) httpmux.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := NewResponseWriter(w)
			next(rw, r)
			b := apacheCommonLog(rw, r, start)
			l.Print(b.String())
			putBuffer(b)
		}
	}
}

// ApacheCombinedFormat returns a middleware that logs http requests
// to the given logger using the Apache Combined log format.
func ApacheCombinedFormat(l *log.Logger) httpmux.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := NewResponseWriter(w)
			next(rw, r)
			b := apacheCommonLog(rw, r, start)
			b.Write([]byte(" "))
			fmt.Fprintf(b, "%q %q",
				r.Header.Get("Referer"),
				r.Header.Get("User-Agent"),
			)
			l.Print(b.String())
			putBuffer(b)
		}
	}
}
