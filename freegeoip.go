// Copyright 2009-2014 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// A Handler provides http handlers that can process requests and return
// data in multiple formats.
//
// Usage:
//
// 	handle := NewHandler(db)
// 	http.Handle("/json/", handle.JSON())
//
// Note that the url pattern must end with a trailing slash since the
// handler looks for IP addresses or hostnames as parameters, for
// example /json/8.8.8.8 or /json/domain.com.
//
// If no IP or hostname is provided, then the handler will query the
// IP address of the caller. See the ProxyHandler for more.
type Handler struct {
	db  *DB
	enc Encoder
}

// NewHandler creates and initializes a new Handler.
func NewHandler(db *DB, enc Encoder) *Handler {
	return &Handler{db, enc}
}

// ServeHTTP implements the http.Handler interface.
func (f *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := f.queryIP(r)
	if ip == nil {
		http.NotFound(w, r)
		return
	}
	q := f.enc.NewQuery()
	err := f.db.Lookup(ip, q)
	if err != nil {
		http.Error(w, "Try again later.",
			http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Database-Date", f.db.Date().Format(http.TimeFormat))
	err = f.enc.Encode(w, r, q, ip)
	if err != nil {
		f.db.sendError(fmt.Errorf("Failed to encode %#v: %s", q, err))
		http.Error(w, "An unexpected error occurred.",
			http.StatusInternalServerError)
		return
	}
}

func (f *Handler) queryIP(r *http.Request) net.IP {
	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		return f.remoteAddr(r)
	}
	q := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	if ip := net.ParseIP(q); ip != nil {
		return ip
	}
	ip, err := net.LookupIP(q)
	if err != nil {
		return nil // Not found.
	}
	if len(ip) == 0 {
		return nil
	}
	return ip[rand.Intn(len(ip))]
}

func (f *Handler) remoteAddr(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

// ProxyHandler is a wrapper for other http handlers that sets the
// client IP address in request.RemoteAddr to the first value of a
// comma separated list of IPs from the X-Forwarded-For request
// header. It resets the original RemoteAddr back after running the
// designated handler f.
func ProxyHandler(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := r.Header.Get("X-Forwarded-For")
		if len(addr) > 0 {
			remoteAddr := r.RemoteAddr
			r.RemoteAddr = strings.SplitN(addr, ",", 2)[0]
			defer func() { r.RemoteAddr = remoteAddr }()
		}
		f.ServeHTTP(w, r)
	})
}
