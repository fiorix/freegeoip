// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"log"
	"net"
	"time"
)

func ExampleOpen() {
	db, err := Open("./testdata.gz")
	if err != nil {
		log.Fatal(err)
	}
	var result customQuery
	err = db.Lookup(net.ParseIP("8.8.8.8"), &result)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Printf("%#v", result)
}

func ExampleOpenURL() {
	updateInterval := 24 * time.Hour
	maxRetryInterval := time.Hour
	db, err := OpenURL(MaxMindDB, updateInterval, maxRetryInterval)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	select {
	case <-db.NotifyOpen():
		// Wait for the db to be downloaded.
	case err := <-db.NotifyError():
		log.Fatal(err)
	}
	var result customQuery
	err = db.Lookup(net.ParseIP("8.8.8.8"), &result)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%#v", result)
}

// A customQuery is the query executed in the maxmind database for
// every IP lookup request.
type customQuery struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
}
