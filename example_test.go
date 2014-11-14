// Copyright 2009-2014 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

var maxmindFile = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"

func ExampleDatabaseQuery() {
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

func ExampleRemoteDatabaseQuery() {
	updateInterval := 24 * time.Hour
	maxRetryInterval := time.Hour
	db, err := OpenURL(maxmindFile, updateInterval, maxRetryInterval)
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

func ExampleServer() {
	db, err := OpenURL(maxmindFile, 24*time.Hour, time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/csv/", NewHandler(db, &CSVEncoder{}))
	http.Handle("/xml/", NewHandler(db, &XMLEncoder{}))
	http.Handle("/json/", NewHandler(db, &JSONEncoder{}))
	http.ListenAndServe(":8080", nil)
}

func ExampleServerWithCustomEncoder() {
	db, err := Open("./testdata/db.gz")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/custom/json/", NewHandler(db, &customEncoder{}))
	http.ListenAndServe(":8080", nil)
}

// A customEncoder writes a custom JSON object to an http response.
type customEncoder struct{}

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

// A customResponse is what gets written to the http response as JSON.
type customResponse struct {
	IP          string
	CountryCode string
	CountryName string
	Latitude    float64
	Longitude   float64
	TimeZone    string
}

// NewQuery implements the freegeoip.Encoder interface.
func (f *customEncoder) NewQuery() Query {
	return &customQuery{}
}

// Encode implements the freegeoip.Encoder interface.
func (f *customEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := q.(*customQuery)
	out := &customResponse{
		IP:          ip.String(),
		CountryCode: record.Country.ISOCode,
		CountryName: record.Country.Names["en"], // Set to client lang.
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
		TimeZone:    record.Location.TimeZone,
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(&out)
}
