// Copyright 2009-2014 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"bytes"
	"encoding/csv"

	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestQueryRemoteAddr(t *testing.T) {
	want := net.ParseIP("8.8.8.8")
	// No query argument, so we query the remote IP.
	r := http.Request{
		URL:        &url.URL{Path: "/"},
		RemoteAddr: "8.8.8.8:8888",
		Header:     http.Header{},
	}
	f := &Handler{}
	if ip := f.queryIP(&r); !bytes.Equal(ip, want) {
		t.Errorf("Unexpected IP: %s", ip)
	}
}

func TestQueryDNS(t *testing.T) {
	want4 := net.ParseIP("8.8.8.8")
	want6 := net.ParseIP("2001:4860:4860::8888")
	r := http.Request{
		URL:        &url.URL{Path: "/google-public-dns-a.google.com"},
		RemoteAddr: "127.0.0.1:8080",
		Header:     make(map[string][]string),
	}
	f := &Handler{}
	ip := f.queryIP(&r)
	if ip == nil {
		t.Fatal("Failed to resolve", r.URL.Path)
	}
	if !bytes.Equal(ip, want4) && !bytes.Equal(ip, want6) {
		t.Errorf("Unexpected IP: %s", ip)
	}
}

// Test the server.

func runServer(pattern string, f Encoder) (*Handler, *httptest.Server, error) {
	db, err := Open(testFile)
	if err != nil {
		return nil, nil, err
	}
	select {
	case <-db.NotifyOpen():
	case err := <-db.NotifyError():
		if err != nil {
			return nil, nil, err
		}
	}
	mux := http.NewServeMux()
	handle := NewHandler(db, f)
	mux.Handle(pattern, ProxyHandler(handle))
	return handle, httptest.NewServer(mux), nil
}

func TestLookupUnavailable(t *testing.T) {
	handle, srv, err := runServer("/csv/", &CSVEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	handle.db.mu.Lock()
	reader := handle.db.reader
	handle.db.reader = nil
	handle.db.mu.Unlock()
	defer func() {
		handle.db.mu.Lock()
		handle.db.reader = reader
		handle.db.mu.Unlock()
	}()
	resp, err := http.Get(srv.URL + "/csv/8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatalf("Unexpected query worked: %s\n%s", resp.Status, b)
	}
}

func TestLookupNotFound(t *testing.T) {
	_, srv, err := runServer("/csv/", &CSVEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/csv/fail-me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatalf("Unexpected query worked: %s\n%s", resp.Status, b)
	}
}

func TestLookupXForwardedFor(t *testing.T) {
	_, srv, err := runServer("/csv/", &CSVEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	req, err := http.NewRequest("GET", srv.URL+"/csv/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	resp, err := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	row, err := csv.NewReader(resp.Body).Read()
	if err != nil {
		t.Fatal(err)
	}
	if row[1] != "US" {
		t.Fatalf("Unexpected country code in record: %#v", row)
	}
}

func TestLookupDatabaseDate(t *testing.T) {
	_, srv, err := runServer("/csv/", &CSVEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/csv/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	if len(resp.Header.Get("X-Database-Date")) == 0 {
		t.Fatal("Header X-Database-Date is missing")
	}
}
