// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"net"
	"net/http"
	"path/filepath"

	"github.com/fiorix/freegeoip"
	"github.com/prometheus/client_golang/prometheus"
)

// HandlerConfig holds configuration for freegeoip http handlers.
type HandlerConfig struct {
	Prefix      string
	Origin      string
	PublicDir   string
	DB          *freegeoip.DB
	RateLimiter RateLimiter
}

// NewHandler creates a freegeoip http handler.
func NewHandler(conf *HandlerConfig) http.Handler {
	ah := &apiHandler{conf}
	mux := http.NewServeMux()
	ah.RegisterPublicDir(mux)
	ah.RegisterEncoder(mux, "csv", &freegeoip.CSVEncoder{UseCRLF: true})
	ah.RegisterEncoder(mux, "xml", &freegeoip.XMLEncoder{Indent: true})
	ah.RegisterEncoder(mux, "json", &freegeoip.JSONEncoder{})
	return mux
}

type ConnStateFunc func(c net.Conn, s http.ConnState)

func ConnStateMetrics(g prometheus.Gauge) ConnStateFunc {
	return func(c net.Conn, s http.ConnState) {
		switch s {
		case http.StateNew:
			g.Inc()
		case http.StateClosed:
			g.Dec()
		}
	}
}

type apiHandler struct {
	conf *HandlerConfig
}

func (ah *apiHandler) prefix(path string) string {
	p := filepath.Clean(filepath.Join("/", ah.conf.Prefix, path))
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
