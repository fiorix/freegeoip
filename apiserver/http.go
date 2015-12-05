// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"net"
	"net/http"
	"path"

	"github.com/fiorix/freegeoip"
	"github.com/prometheus/client_golang/prometheus"
)

// HandlerConfig holds configuration for freegeoip http handlers.
type HandlerConfig struct {
	Prefix           string
	Origin           string
	PublicDir        string
	DB               *freegeoip.DB
	RateLimiter      RateLimiter
	UseXForwardedFor bool
}

// NewHandler creates a freegeoip http handler.
func NewHandler(conf *HandlerConfig) http.Handler {
	ah := &apiHandler{conf}
	mux := http.NewServeMux()
	ah.RegisterPublicDir(mux)
	ah.RegisterEncoder(mux, "csv", &freegeoip.CSVEncoder{UseCRLF: true})
	ah.RegisterEncoder(mux, "xml", &freegeoip.XMLEncoder{Indent: true})
	ah.RegisterEncoder(mux, "json", &freegeoip.JSONEncoder{})
	return ah.metricsCollector(mux)
}

// ConnStateFunc is a function that can handle connection state.
type ConnStateFunc func(c net.Conn, s http.ConnState)

// ConnStateMetrics collect metrics per connection state, per protocol.
// e.g. new http, closed http.
func ConnStateMetrics(proto string) ConnStateFunc {
	return func(c net.Conn, s http.ConnState) {
		switch s {
		case http.StateNew:
			clientConnsGauge.WithLabelValues(proto).Inc()
		case http.StateClosed:
			clientConnsGauge.WithLabelValues(proto).Dec()
		}
	}
}

type apiHandler struct {
	conf *HandlerConfig
}

func (ah *apiHandler) prefix(p string) string {
	p = path.Clean(path.Join("/", ah.conf.Prefix, p))
	if p[len(p)-1] != '/' {
		p += "/"
	}
	return p
}

func (ah *apiHandler) RegisterPublicDir(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir(ah.conf.PublicDir))
	fs = prometheus.InstrumentHandler("frontend", fs)
	prefix := ah.prefix("")
	mux.Handle(prefix, http.StripPrefix(prefix, fs))
}

func (ah *apiHandler) RegisterEncoder(mux *http.ServeMux, path string, enc freegeoip.Encoder) {
	f := http.Handler(freegeoip.NewHandler(ah.conf.DB, enc))
	if ah.conf.RateLimiter.Max > 0 {
		rl := ah.conf.RateLimiter
		rl.Handler = f
		f = &rl
	}
	origin := ah.conf.Origin
	if origin == "" {
		origin = "*"
	}
	f = cors(f, origin, "GET", "HEAD")
	f = prometheus.InstrumentHandler(path, f)
	mux.Handle(ah.prefix(path), f)
}

func (ah *apiHandler) metricsCollector(handler http.Handler) http.Handler {
	type query struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	f := func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
		// Collect metrics after serving the request.
		var ip net.IP
		if ah.conf.UseXForwardedFor {
			ip = net.ParseIP(r.RemoteAddr)
		} else {
			addr, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip = net.ParseIP(addr)
		}
		if ip == nil {
			// TODO: increment error count?
			return
		}
		if ip.To4() != nil {
			clientIPProtoCounter.WithLabelValues("4").Inc()
		} else {
			clientIPProtoCounter.WithLabelValues("6").Inc()
		}
		var q query
		err := ah.conf.DB.Lookup(ip, &q)
		if err != nil || q.Country.ISOCode == "" {
			clientCountryCounter.WithLabelValues("unknown").Inc()
			return
		}
		clientCountryCounter.WithLabelValues(q.Country.ISOCode).Inc()
	}
	return http.HandlerFunc(f)
}
