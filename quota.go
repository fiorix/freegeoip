// Copyright 2013-2014 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/fiorix/go-redis/redis"
)

// Quota interface for limiting access to the API.
type Quota interface {
	Setup(args ...string)          // Initialize quota backend
	Ok(ipkey uint32) (bool, error) // Returns true if under quota
}

// MapQuota implements the Quota interface using a map as the backend.
type MapQuota struct {
	mu sync.Mutex
	ip map[uint32]int
}

func (q *MapQuota) Setup(args ...string) {
	q.ip = make(map[uint32]int)
}

func (q *MapQuota) Ok(ipkey uint32) (bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if n, ok := q.ip[ipkey]; ok {
		if n < conf.Limit.MaxRequests {
			q.ip[ipkey]++
			return true, nil
		}
		return false, nil
	}
	q.ip[ipkey] = 1
	go func() {
		time.Sleep(time.Duration(conf.Limit.Expire) * time.Second)
		q.mu.Lock()
		defer q.mu.Unlock()
		delete(q.ip, ipkey)
	}()
	return true, nil
}

// RedisQuota implements the Quota interface using Redis as the backend.
type RedisQuota struct {
	c *redis.Client
}

func (q *RedisQuota) Setup(args ...string) {
	q.c = redis.New(args...)
	q.c.Timeout = time.Duration(1500) * time.Millisecond
}

func (q *RedisQuota) Ok(ipkey uint32) (bool, error) {
	k := fmt.Sprintf("%d", ipkey) // "numeric" key
	if ns, err := q.c.Get(k); err != nil {
		return false, fmt.Errorf("redis get: %s", err.Error())
	} else if ns == "" {
		if err = q.c.SetEx(k, conf.Limit.Expire, "1"); err != nil {
			return false, fmt.Errorf("redis setex: %s", err.Error())
		}
	} else if n, _ := strconv.Atoi(ns); n < conf.Limit.MaxRequests {
		if n, err = q.c.Incr(k); err != nil {
			return false, fmt.Errorf("redis incr: %s", err.Error())
		} else if n == 1 {
			q.c.Expire(k, conf.Limit.Expire)
		}
	} else {
		return false, nil
	}
	return true, nil
}
