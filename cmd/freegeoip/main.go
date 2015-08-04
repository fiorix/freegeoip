// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/fiorix/freegeoip"
	"github.com/fiorix/go-redis/redis"
	"github.com/gorilla/context"
)

// Version tag.
var Version = "3.0.5"

var maxmindFile = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"

func main() {
	addr := flag.String("addr", ":8080", "Address in form of ip:port to listen on")
	certFile := flag.String("cert", "", "X.509 certificate file")
	keyFile := flag.String("key", "", "X.509 key file")
	public := flag.String("public", "", "Public directory to serve at the / endpoint")
	ipdb := flag.String("db", maxmindFile, "IP database file or URL")
	updateIntvl := flag.Duration("update", 24*time.Hour, "Database update check interval")
	retryIntvl := flag.Duration("retry", time.Hour, "Max time to wait before retrying update")
	useXFF := flag.Bool("use-x-forwarded-for", false, "Use the X-Forwarded-For header when available")
	silent := flag.Bool("silent", false, "Do not log requests to stderr")
	redisAddr := flag.String("redis", "127.0.0.1:6379", "Redis address in form of ip:port for quota")
	redisTimeout := flag.Duration("redis-timeout", 500*time.Millisecond, "Redis read/write timeout")
	quotaMax := flag.Int("quota-max", 0, "Max requests per source IP per interval; Set 0 to turn off")
	quotaIntvl := flag.Duration("quota-interval", time.Hour, "Quota expiration interval")
	version := flag.Bool("version", false, "Show version and exit")
	pprof := flag.String("pprof", "", "Address in form of ip:port to listen on for pprof")
	flag.Parse()

	if *version {
		fmt.Printf("freegeoip v%s\n", Version)
		return
	}

	rc, err := redis.Dial(*redisAddr)
	if err != nil {
		log.Fatal(err)
	}
	rc.Timeout = *redisTimeout

	db, err := openDB(*ipdb, *updateIntvl, *retryIntvl)
	if err != nil {
		log.Fatal(err)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	encoders := map[string]http.Handler{
		"/csv/":  freegeoip.NewHandler(db, &freegeoip.CSVEncoder{UseCRLF: true}),
		"/xml/":  freegeoip.NewHandler(db, &freegeoip.XMLEncoder{Indent: true}),
		"/json/": freegeoip.NewHandler(db, &freegeoip.JSONEncoder{}),
	}

	if *quotaMax > 0 {
		seconds := int((*quotaIntvl).Seconds())
		for path, f := range encoders {
			encoders[path] = userQuota(rc, *quotaMax, seconds, f, *silent)
		}
	}

	mux := http.NewServeMux()
	for path, handler := range encoders {
		mux.Handle(path, handler)
	}

	if len(*public) > 0 {
		mux.Handle("/", http.FileServer(http.Dir(*public)))
	}

	handler := CORS(mux, "GET", "HEAD")

	if !*silent {
		log.Println("freegeoip server starting on", *addr)
		go logEvents(db)
		handler = logHandler(handler)
	}

	if *useXFF {
		handler = freegeoip.ProxyHandler(handler)
	}

	if len(*pprof) > 0 {
		go func() {
			log.Fatal(http.ListenAndServe(*pprof, nil))
		}()
	}

	if len(*certFile) > 0 && len(*keyFile) > 0 {
		err = http.ListenAndServeTLS(*addr, *certFile, *keyFile, handler)
	} else {
		err = http.ListenAndServe(*addr, handler)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// openDB opens and returns the IP database.
func openDB(dsn string, updateIntvl, maxRetryIntvl time.Duration) (db *freegeoip.DB, err error) {
	u, err := url.Parse(dsn)
	if err != nil || len(u.Scheme) == 0 {
		db, err = freegeoip.Open(dsn)
	} else {
		db, err = freegeoip.OpenURL(dsn, updateIntvl, maxRetryIntvl)
	}
	return
}

// CORS is an http handler that checks for allowed request methods (verbs)
// and adds CORS headers to all http responses.
//
// See http://en.wikipedia.org/wiki/Cross-origin_resource_sharing for details.
func CORS(f http.Handler, allow ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			strings.Join(allow, ", ")+", OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		for _, method := range allow {
			if r.Method == method {
				f.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Allow", strings.Join(allow, ", ")+", OPTIONS")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
	})
}

// userQuota is a handler that provides a rate limiter to the freegeoip API.
// It allows qmax requests per qintvl, in seconds.
//
// If redis is not available it responds with service unavailable.
func userQuota(rc *redis.Client, qmax int, qintvl int, f http.Handler, silent bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ip string
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
			ip = r.RemoteAddr[:idx]
		} else {
			ip = r.RemoteAddr
		}
		sreq, err := rc.Get(ip)
		if err != nil {
			if !silent {
				context.Set(r, "log", err.Error())
			}
			http.Error(w, "Try again later",
				http.StatusServiceUnavailable)
			return
		}
		if len(sreq) == 0 {
			err = rc.SetEx(ip, qintvl, "1")
			if err != nil {
				if !silent {
					context.Set(r, "log", err.Error())
				}
				http.Error(w, "Try again later",
					http.StatusServiceUnavailable)
				return
			}
			f.ServeHTTP(w, r)
			return
		}
		nreq, _ := strconv.Atoi(sreq)
		if nreq >= qmax {
			http.Error(w, "Quota exceeded", http.StatusForbidden)
			return
		}
		_, err = rc.Incr(ip)
		if err != nil && !silent {
			context.Set(r, "log", err.Error())
		}
		f.ServeHTTP(w, r)
	})
}

// logEvents logs database events.
func logEvents(db *freegeoip.DB) {
	for {
		select {
		case file := <-db.NotifyOpen():
			log.Println("database loaded:", file)
		case err := <-db.NotifyError():
			log.Println("database error:", err)
		case <-db.NotifyClose():
			return
		}
	}
}

// logHandler logs http requests.
func logHandler(f http.Handler) http.Handler {
	empty := ""
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := responseWriter{w, http.StatusOK, 0}
		start := time.Now()
		f.ServeHTTP(&resp, r)
		elapsed := time.Since(start)
		extra := context.Get(r, "log")
		if extra != nil {
			defer context.Clear(r)
		} else {
			extra = empty
		}
		log.Printf("%q %d %q %q %s %q %db in %s %q",
			r.Proto,
			resp.status,
			r.Method,
			r.URL.Path,
			remoteIP(r),
			r.Header.Get("User-Agent"),
			resp.bytes,
			elapsed,
			extra,
		)
	})
}

// remoteIP returns the client's address without the port number.
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// responseWriter is an http.ResponseWriter that records the returned
// status and bytes written to the client.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

// Write implements the http.ResponseWriter interface.
func (f *responseWriter) Write(b []byte) (int, error) {
	n, err := f.ResponseWriter.Write(b)
	if err != nil {
		return 0, err
	}
	f.bytes += n
	return n, nil
}

// WriteHeader implements the http.ResponseWriter interface.
func (f *responseWriter) WriteHeader(code int) {
	f.status = code
	f.ResponseWriter.WriteHeader(code)
}
