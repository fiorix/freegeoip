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
	[]string{"event"},
)

var clientIPProtoCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "client_ipproto_version",
		Help: "IP version of clients",
	},
	[]string{"ip"},
)

var clientConnsGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "client_connections",
		Help: "Number of client connections per protocol",
	},
	[]string{"proto"},
)

func init() {
	prometheus.MustRegister(dbEventCounter)
	prometheus.MustRegister(clientConnsGauge)
	prometheus.MustRegister(clientIPProtoCounter)
}
