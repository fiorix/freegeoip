// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	// embed pprof server.
	_ "net/http/pprof"

	"github.com/fiorix/freegeoip"
	"github.com/fiorix/go-redis/redis"
	gorilla "github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/http2"
)

// Version tag.
var Version = "3.0.10"

var maxmindDB = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"

var (
	flAPIPrefix      = flag.String("api-prefix", "/", "Prefix for API endpoints")
	flCORSOrigin     = flag.String("cors-origin", "*", "CORS origin API endpoints")
	flHTTPAddr       = flag.String("http", ":8080", "Address in form of ip:port to listen on for HTTP")
	flHTTPSAddr      = flag.String("https", "", "Address in form of ip:port to listen on for HTTPS")
	flCertFile       = flag.String("cert", "cert.pem", "X.509 certificate file")
	flKeyFile        = flag.String("key", "key.pem", "X.509 key file")
	flReadTimeout    = flag.Duration("read-timeout", 30*time.Second, "Read timeout for HTTP and HTTPS client conns")
	flWriteTimeout   = flag.Duration("write-timeout", 15*time.Second, "Write timeout for HTTP and HTTPS client conns")
	flPublicDir      = flag.String("public", "", "Public directory to serve at the {prefix}/ endpoint")
	flDB             = flag.String("db", maxmindDB, "IP database file or URL")
	flUpdateIntvl    = flag.Duration("update", 24*time.Hour, "Database update check interval")
	flRetryIntvl     = flag.Duration("retry", time.Hour, "Max time to wait before retrying to download database")
	flUseXFF         = flag.Bool("use-x-forwarded-for", false, "Use the X-Forwarded-For header when available (e.g. when running behind proxies)")
	flSilent         = flag.Bool("silent", false, "Do not log HTTP or HTTPS requests to stderr")
	flLogToStdout    = flag.Bool("logtostdout", false, "Log to stdout instead of stderr")
	flRedisAddr      = flag.String("redis", "127.0.0.1:6379", "Redis address in form of ip:port[,ip:port] for quota")
	flRedisTimeout   = flag.Duration("redis-timeout", time.Second, "Redis read/write timeout")
	flQuotaMax       = flag.Int("quota-max", 0, "Max requests per source IP per interval; set 0 to turn off")
	flQuotaIntvl     = flag.Duration("quota-interval", time.Hour, "Quota expiration interval per source IP querying the API")
	flVersion        = flag.Bool("version", false, "Show version and exit")
	flInternalServer = flag.String("internal-server", "", "Address in form of ip:port to listen on for /metrics and /debug/pprof")
)

// Run is the entrypoint for the freegeoip daemon tool.
func Run() error {
	flag.Parse()

	if *flVersion {
		fmt.Printf("freegeoip %s\n", Version)
		return nil
	}

	if *flLogToStdout {
		log.SetOutput(os.Stdout)
	}

	log.SetPrefix("[freegeoip] ")

	addrs := strings.Split(*flRedisAddr, ",")
	rc, err := redis.NewClient(addrs...)
	if err != nil {
		return err
	}
	rc.SetTimeout(*flRedisTimeout)

	db, err := openDB(*flDB, *flUpdateIntvl, *flRetryIntvl)
	if err != nil {
		return err
	}
	go watchEvents(db)

	ah := NewHandler(&HandlerConfig{
		Prefix:    *flAPIPrefix,
		Origin:    *flCORSOrigin,
		PublicDir: *flPublicDir,
		DB:        db,
		RateLimiter: RateLimiter{
			Redis:    rc,
			Max:      *flQuotaMax,
			Interval: *flQuotaIntvl,
		},
		UseXForwardedFor: *flUseXFF,
	})

	if !*flSilent {
		ah = gorilla.CombinedLoggingHandler(os.Stderr, ah)
	}

	if *flUseXFF {
		ah = freegeoip.ProxyHandler(ah)
	}

	if len(*flInternalServer) > 0 {
		http.Handle("/metrics", prometheus.Handler())
		log.Println("freegeoip internal server starting on", *flInternalServer)
		go func() { log.Fatal(http.ListenAndServe(*flInternalServer, nil)) }()
	}

	if *flHTTPAddr != "" {
		log.Println("freegeoip http server starting on", *flHTTPAddr)
		srv := &http.Server{
			Addr:         *flHTTPAddr,
			Handler:      ah,
			ReadTimeout:  *flReadTimeout,
			WriteTimeout: *flWriteTimeout,
			ConnState:    ConnStateMetrics("http"),
		}
		go func() { log.Fatal(srv.ListenAndServe()) }()
	}

	if *flHTTPSAddr != "" {
		log.Println("freegeoip https server starting on", *flHTTPSAddr)
		srv := &http.Server{
			Addr:         *flHTTPSAddr,
			Handler:      ah,
			ReadTimeout:  *flReadTimeout,
			WriteTimeout: *flWriteTimeout,
			ConnState:    ConnStateMetrics("https"),
		}
		http2.ConfigureServer(srv, nil)
		go func() { log.Fatal(srv.ListenAndServeTLS(*flCertFile, *flKeyFile)) }()
	}

	select {}
}
