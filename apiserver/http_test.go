// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fiorix/freegeoip"
	"github.com/fiorix/go-redis/redis"
)

func newTestHandler(db *freegeoip.DB) http.Handler {
	return NewHandler(&HandlerConfig{
		Prefix:    "/api",
		PublicDir: ".",
		DB:        db,
		RateLimiter: RateLimiter{
			Redis:    redis.New(),
			Max:      5,
			Interval: time.Second,
			KeyMaker: KeyMakerFunc(func(r *http.Request) string {
				return "handler-test"
			}),
		},
	})
}

func TestHandler(t *testing.T) {
	db, err := freegeoip.Open("../testdata/db.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := httptest.NewServer(newTestHandler(db))
	defer s.Close()
	// query some known location...
	resp, err := http.Get(s.URL + "/api/json/200.1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusServiceUnavailable:
		t.Skip("Redis available?")
	default:
		t.Fatal(resp.Status)
	}
	m := struct {
		Country string `json:"country_name"`
		City    string `json:"city"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m.Country != "Venezuela" && m.City != "Caracas" {
		t.Fatalf("Query data does not match: want Caracas,Venezuela, have %q,%q",
			m.City, m.Country)
	}
}
