// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	// embed pprof server.
	_ "net/http/pprof"

	"github.com/fiorix/go-listener/listener"
	"github.com/prometheus/client_golang/prometheus"
)

// Version tag.
var Version = "3.4.1"

// Run is the entrypoint for the freegeoip server.
func Run() {
	c := NewConfig()
	c.AddFlags(flag.CommandLine)
	sv := flag.Bool("version", false, "Show version and exit")
	flag.Parse()
	if *sv {
		fmt.Printf("freegeoip %s\n", Version)
		return
	}
	if c.LogToStdout {
		log.SetOutput(os.Stdout)
	}
	if !c.LogTimestamp {
		log.SetFlags(0)
	}
	f, err := NewHandler(c)
	if err != nil {
		log.Fatal(err)
	}
	if c.ServerAddr != "" {
		go runServer(c, f)
	}
	if c.TLSServerAddr != "" {
		go runTLSServer(c, f)
	}
	if c.InternalServerAddr != "" {
		go runInternalServer(c)
	}
	select {}
}

// connStateFunc is a function that can handle connection state.
type connStateFunc func(c net.Conn, s http.ConnState)

// connStateMetrics collect metrics per connection state, per protocol.
// e.g. new http, closed http.
func connStateMetrics(proto string) connStateFunc {
	return func(c net.Conn, s http.ConnState) {
		switch s {
		case http.StateNew:
			clientConnsGauge.WithLabelValues(proto).Inc()
		case http.StateClosed:
			clientConnsGauge.WithLabelValues(proto).Dec()
		}
	}
}

func listenerOpts(c *Config) []listener.Option {
	opts := []listener.Option{}
	if c.FastOpen {
		opts = append(opts, listener.FastOpen())
	}
	if c.Naggle {
		opts = append(opts, listener.Naggle())
	}
	return opts
}

func runServer(c *Config, f http.Handler) {
	log.Println("freegeoip http server starting on", c.ServerAddr)
	ln, err := listener.New(c.ServerAddr, listenerOpts(c)...)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Handler:      f,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		ErrorLog:     c.errorLogger(),
		ConnState:    connStateMetrics("http"),
	}
	log.Fatal(srv.Serve(ln))
}

func runTLSServer(c *Config, f http.Handler) {
	log.Println("freegeoip https server starting on", c.TLSServerAddr)
	opts := listenerOpts(c)
	if c.HTTP2 {
		opts = append(opts, listener.HTTP2())
	}
	if c.LetsEncrypt {
		if c.LetsEncryptHosts == "" {
			log.Fatal("must set at least one host using --letsencrypt-hosts")
		}
		opts = append(opts, listener.LetsEncrypt(
			c.LetsEncryptCacheDir,
			c.LetsEncryptEmail,
			strings.Split(c.LetsEncryptHosts, ",")...,
		))
	} else {
		opts = append(opts, listener.TLS(c.TLSCertFile, c.TLSKeyFile))
	}
	ln, err := listener.New(c.TLSServerAddr, opts...)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:         c.TLSServerAddr,
		Handler:      f,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		ErrorLog:     c.errorLogger(),
		ConnState:    connStateMetrics("https"),
		TLSConfig:    ln.TLSConfig(),
	}
	log.Fatal(srv.Serve(ln))
}

func runInternalServer(c *Config) {
	http.Handle("/metrics", prometheus.Handler())
	log.Println("freegeoip internal server starting on", c.InternalServerAddr)
	log.Fatal(http.ListenAndServe(c.InternalServerAddr, nil))
}
