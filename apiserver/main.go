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

	// embed pprof server.
	_ "net/http/pprof"
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

func runServer(c *Config, f http.Handler) {
	log.Println("freegeoip http server starting on", c.ServerAddr)
	s := &http.Server{
		Addr:         c.ServerAddr,
		Handler:      f,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		ErrorLog:     c.errorLogger(),
		ConnState:    connStateMetrics("http"),
	}
	log.Fatal(s.ListenAndServe())
}

func runTLSServer(c *Config, f http.Handler) {
	log.Println("freegeoip https server starting on", c.TLSServerAddr)
	s := &http.Server{
		Addr:         c.TLSServerAddr,
		Handler:      f,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		ErrorLog:     c.errorLogger(),
		ConnState:    connStateMetrics("https"),
	}
	log.Fatal(s.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile))
}
