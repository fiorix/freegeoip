// Copyright 2009-2014 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"net"
	"sync"
	"testing"
	"time"
)

var (
	testCf *configFile
	testDB *DB
)

func TestLoadConfig(t *testing.T) {
	var err error
	testCf, err = loadConfig("freegeoip.conf")
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenDB(t *testing.T) {
	var err error
	testDB, err = openDB(testCf)
	if err != nil {
		t.Log("Make sure the DB exists: cd db && ./updatedb")
		t.Fatal(err)
	}
}

func TestQueryDB1(t *testing.T) {
	if testDB == nil {
		t.Skip("DB is not available")
	}
	ip := net.ParseIP("127.0.0.1")
	nIP, _ := ip2int(ip)
	record, err := testDB.Lookup(ip, nIP)
	if err != nil {
		t.Fatal(err)
	}
	if record.CountryName != "Reserved" {
		t.Fatal("Unexpected value:", record.CountryName)
	}
}

func TestQueryDB2(t *testing.T) {
	if testDB == nil {
		t.Skip("DB is not available")
	}
	ip := net.ParseIP("8.8.8.8")
	nIP, _ := ip2int(ip)
	record, err := testDB.Lookup(ip, nIP)
	if err != nil {
		t.Fatal(err)
	}
	if record.CountryCode != "US" {
		t.Fatal("Unexpected value:", record.CountryCode)
	}
}

func TestMapQuota(t *testing.T) {
	testCf.Limit.MaxRequests = 1
	testCf.Limit.Expire = 1
	rl := new(mapQuota)
	rl.Setup(testCf)
	nIP, _ := ip2int(net.ParseIP("127.0.0.1"))
	if ok, _ := rl.Ok(nIP); !ok {
		t.Fatal("Unexpected value:", ok)
	}
	if ok, _ := rl.Ok(nIP); ok {
		t.Fatal("Unexpected value:", ok)
	}
}

func TestRedisQuota(t *testing.T) {
	if len(testCf.Redis) < 1 {
		t.Skip("Redis is not configured")
	}
	testCf.Limit.MaxRequests = 1
	testCf.Limit.Expire = 1
	rl := new(redisQuota)
	rl.Setup(testCf)
	nIP, _ := ip2int(net.ParseIP("127.0.0.1"))
	if ok, err := rl.Ok(nIP); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("Unexpected value:", ok)
	}
	if ok, err := rl.Ok(nIP); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("Unexpected value:", ok)
	}
}

func TestDNSLookup(t *testing.T) {
	dh := &dnsHandler{
		Timeout:       1000 * time.Millisecond,
		MaxConcurrent: 1,
	}
	if ip := dh.LookupHost("localhost"); ip == nil {
		t.Fatal("Could not resolve host name")
	} else if !isLocalhostIP(ip.String()) {
		t.Fatal("Unexpected IP: " + ip.String())
	}
}

func TestConcurrentDNSLookup1(t *testing.T) {
	dh := &dnsHandler{
		Timeout:       1000 * time.Millisecond,
		MaxConcurrent: 1,
	}
	go dh.LookupHost("invalid.host.name1")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if ip := dh.LookupHost("localhost"); ip != nil {
			t.Error("Unexpected IP: " + ip.String())
		}
		wg.Done()
	}()
	wg.Wait()
}

func TestConcurrentDNSLookup2(t *testing.T) {
	dh := &dnsHandler{
		Timeout:       1000 * time.Millisecond,
		MaxConcurrent: 2,
	}
	go dh.LookupHost("invalid.host.name2")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if ip := dh.LookupHost("localhost"); ip == nil {
			t.Error("Could not resolve host name")
		} else if !isLocalhostIP(ip.String()) {
			t.Error("Unexpected IP: " + ip.String())
		}
		wg.Done()
	}()
	wg.Wait()
}

func isLocalhostIP(ip string) bool {
	for _, v := range []string{"::1", "fe80::1", "127.0.0.1"} {
		if ip == v {
			return true
		}
	}
	return false
}
