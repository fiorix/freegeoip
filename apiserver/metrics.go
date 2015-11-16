// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import "github.com/prometheus/client_golang/prometheus"

// Experimental metrics for Prometheus, might change in the future.

var dbEventCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "db_event_counter",
		Help: "Counter per DB event",
	},
	[]string{"event", "data"},
)

var httpConnsGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "current_http_conns",
		Help: "Current number of HTTP connections",
	},
)

var httpsConnsGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "current_https_conns",
		Help: "Current number of HTTPS connections",
	},
)

func init() {
	prometheus.MustRegister(dbEventCounter)
	prometheus.MustRegister(httpConnsGauge)
	prometheus.MustRegister(httpsConnsGauge)
}
