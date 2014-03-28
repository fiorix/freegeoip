// Copyright 2009-2014 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
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
	record := testQuery(t, "127.0.0.1")
	if record.CountryName != "Reserved" {
		t.Fatal("Unexpected value:", record.CountryName)
	}
}

func TestQueryDB2(t *testing.T) {
	record := testQuery(t, "8.8.8.8")
	if record.CountryCode != "US" {
		t.Fatal("Unexpected value:", record.CountryCode)
	}
}

func TestRecordJSON(t *testing.T) {
	record := testQuery(t, "127.0.0.1")
	b := bytes.NewBuffer(nil)
	record.JSON(b)
	if len(b.Bytes()) != 180 {
		t.Fatal("Unexpected value:", b.String())
	}
}

func TestRecordJSONP(t *testing.T) {
	record := testQuery(t, "127.0.0.1")
	b := bytes.NewBuffer(nil)
	record.JSONP(b, "f")
	if len(b.Bytes()) != 184 {
		t.Fatal("Unexpected value:", b.String())
	}
}

func TestRecordXML(t *testing.T) {
	record := testQuery(t, "127.0.0.1")
	b := bytes.NewBuffer(nil)
	record.XML(b)
	if len(b.Bytes()) != 338 {
		t.Fatal("Unexpected value:", b.String())
	}
}

func TestRecordCSV(t *testing.T) {
	record := testQuery(t, "127.0.0.1")
	b := bytes.NewBuffer(nil)
	record.CSV(b)
	if len(b.Bytes()) != 65 {
		t.Fatal("Unexpected value:", b.String())
	}
}

func TestMapQuota(t *testing.T) {
	testCf.Limit.MaxRequests = 1
	testCf.Limit.Expire = 1
	rl := new(mapQuota)
	rl.init(testCf)
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
	rl.init(testCf)
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
	dp := new(dnsPool)
	dp.init(1, 1000*time.Millisecond)
	if ip := dp.LookupHost("localhost"); ip == nil {
		t.Fatal("Could not resolve host name")
	} else if !isLocalhostIP(ip.String()) {
		t.Fatal("Unexpected IP: " + ip.String())
	}
}

func TestConcurrentDNSLookup1(t *testing.T) {
	dp := new(dnsPool)
	dp.init(1, 1000*time.Millisecond)
	go dp.LookupHost("invalid.host.name1")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if ip := dp.LookupHost("localhost"); ip != nil {
			t.Error("Unexpected IP: " + ip.String())
		}
		wg.Done()
	}()
	wg.Wait()
}

func TestConcurrentDNSLookup2(t *testing.T) {
	dp := new(dnsPool)
	dp.init(2, 1000*time.Millisecond)
	go dp.LookupHost("invalid.host.name2")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if ip := dp.LookupHost("localhost"); ip == nil {
			t.Error("Could not resolve host name")
		} else if !isLocalhostIP(ip.String()) {
			t.Error("Unexpected IP: " + ip.String())
		}
		wg.Done()
	}()
	wg.Wait()
}

func BenchmarkQueryDB(b *testing.B) {
	if testDB == nil {
		b.Skip("DB is not available")
	}
	ip := net.ParseIP("8.8.8.8")
	nIP, _ := ip2int(ip)
	for i := 0; i < b.N; i++ {
		_, err := testDB.Lookup(ip, nIP)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func testQuery(t *testing.T, addr string) *geoipRecord {
	if testDB == nil {
		t.Skip("DB is not available")
	}
	ip := net.ParseIP(addr)
	nIP, _ := ip2int(ip)
	record, err := testDB.Lookup(ip, nIP)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func isLocalhostIP(ip string) bool {
	for _, v := range []string{"::1", "fe80::1", "127.0.0.1"} {
		if ip == v {
			return true
		}
	}
	return false
}
