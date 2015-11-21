// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fiorix/go-redis/redis"
)

func TestRateLimiter(t *testing.T) {
	counter := struct {
		sync.Mutex
		n int
	}{}
	hf := func(w http.ResponseWriter, r *http.Request) {
		counter.Lock()
		counter.n++
		counter.Unlock()
	}
	kmf := func(r *http.Request) string {
		return "rate-limiter-test"
	}
	rl := &RateLimiter{
		Redis:    redis.New(),
		Max:      2,
		Interval: time.Second,
		KeyMaker: KeyMakerFunc(kmf),
		Handler:  http.HandlerFunc(hf),
	}
	mux := http.NewServeMux()
	mux.Handle("/", rl)
	s := httptest.NewServer(mux)
	defer s.Close()
	for i := 0; i < 3; i++ {
		resp, err := http.Get(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusServiceUnavailable {
			t.Skip("Redis unavailable, cannot proceed")
		}
		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusForbidden && i != 2 {
				t.Fatal(resp.Status)
			}
		}
		lim, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Limit"))
		rem, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
		res, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Reset"))
		switch {
		case i == 0 && lim == 2 && rem == 1 && res > 0:
		case (i == 1 || i == 2) && lim == 2 && rem == 0 && res > 0:
		default:
			log.Fatalf("Unexpected values: limit=%d, remaining=%d, reset=%d",
				lim, rem, res)
		}
	}
}
