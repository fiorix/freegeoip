// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import "github.com/prometheus/client_golang/prometheus"

// Experimental metrics for Prometheus, might change in the future.

var dbEventCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "freegeoip_db_events_total",
		Help: "Database events",
	},
	[]string{"event"},
)

var clientCountryCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "freegeoip_client_country_code_total",
		Help: "Country ISO code of clients",
	},
	[]string{"country_code"},
)

var clientIPProtoCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "freegeoip_client_ipproto_version_total",
		Help: "IP version (4 or 6) of clients",
	},
	[]string{"ip"},
)

var clientConnsGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "freegeoip_client_connections",
		Help: "Number of active client connections per protocol",
	},
	[]string{"proto"},
)

func init() {
	prometheus.MustRegister(dbEventCounter)
	prometheus.MustRegister(clientCountryCounter)
	prometheus.MustRegister(clientConnsGauge)
	prometheus.MustRegister(clientIPProtoCounter)
}
