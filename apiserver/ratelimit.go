// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/fiorix/go-redis/redis"
)

var (
	errQuotaExceeded    = errors.New("Quota exceeded")
	errRedisUnavailable = errors.New("Try again later")
)

// A KeyMaker makes keys from the http.Request object to the RateLimiter.
type KeyMaker interface {
	KeyFor(r *http.Request) string
}

// KeyMakerFunc is an adapter function for KeyMaker.
type KeyMakerFunc func(r *http.Request) string

// KeyFor implements the KeyMaker interface.
func (f KeyMakerFunc) KeyFor(r *http.Request) string {
	return f(r)
}

// DefaultKeyMaker is a KeyMaker that returns the client IP
// address from http.Request.RemoteAddr.
var DefaultKeyMaker = KeyMakerFunc(func(r *http.Request) string {
	addr, _, _ := net.SplitHostPort(r.RemoteAddr)
	return addr
})

// A RateLimiter is an http.Handler that wraps another handler,
// and calls it up to a certain limit, max per interval.
type RateLimiter struct {
	Redis    *redis.Client
	Max      int
	Interval time.Duration
	KeyMaker KeyMaker
	Handler  http.Handler

	secInterval int
	once        sync.Once
}

// ServeHTTP implements the http.Handler interface.
func (rl *RateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rl.once.Do(func() {
		rl.secInterval = int(rl.Interval.Seconds())
		if rl.KeyMaker == nil {
			rl.KeyMaker = DefaultKeyMaker
		}
	})
	status, err := rl.do(w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
}

func (rl *RateLimiter) do(w http.ResponseWriter, r *http.Request) (int, error) {
	k := rl.KeyMaker.KeyFor(r)
	nreq, err := rl.Redis.Incr(k)
	if err != nil {
		return http.StatusServiceUnavailable, errRedisUnavailable
	}
	ttl, err := rl.Redis.TTL(k)
	if err != nil {
		return http.StatusServiceUnavailable, errRedisUnavailable
	}
	if ttl == -1 {
		if _, err = rl.Redis.Expire(k, rl.secInterval); err != nil {
			return http.StatusServiceUnavailable, errRedisUnavailable
		}
		ttl = rl.secInterval
	}
	rem := rl.Max - nreq
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.Max))
	w.Header().Set("X-RateLimit-Reset", strconv.Itoa(ttl))
	if rem < 0 {
		w.Header().Set("X-RateLimit-Remaining", "0")
		return http.StatusForbidden, errQuotaExceeded
	}
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rem))
	rl.Handler.ServeHTTP(w, r)
	return http.StatusOK, nil
}
