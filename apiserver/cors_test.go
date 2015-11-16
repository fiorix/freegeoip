// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCORS(t *testing.T) {
	// set up the test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}
	mux := http.NewServeMux()
	mux.Handle("/", cors(http.HandlerFunc(handler), "*", "GET"))
	ts := httptest.NewServer(mux)
	defer ts.Close()
	// create and issue an OPTIONS request and
	// validate response status and headers.
	req, err := http.NewRequest("OPTIONS", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Origin", ts.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected response status: %s", resp.Status)
	}
	if resp.ContentLength != 0 {
		t.Fatalf("Unexpected Content-Length. Want 0, have %d",
			resp.ContentLength)
	}
	want := []struct {
		Name  string
		Value string
	}{
		{"Access-Control-Allow-Origin", ts.URL},
		{"Access-Control-Allow-Methods", "GET, OPTIONS"},
		{"Access-Control-Allow-Credentials", "true"},
	}
	for _, th := range want {
		if v := resp.Header.Get(th.Name); v != th.Value {
			t.Fatalf("Unexpected value for %q. Want %q, have %q",
				th.Name, th.Value, v)
		}
	}
	// issue a GET request and validate response headers and body
	resp, err = http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	want[0].Value = "*" // Origin
	for _, th := range want {
		if v := resp.Header.Get(th.Name); v != th.Value {
			t.Fatalf("Unexpected value for %q. Want %q, have %q",
				th.Name, th.Value, v)
		}
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	wb := []byte("hello world")
	if !bytes.Equal(b, wb) {
		t.Fatalf("Unexpected response body. Want %q, have %q", b, wb)
	}
	// issue a POST request and validate response status
	resp, err = http.PostForm(ts.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("Unexpected response status: %s", resp.Status)
	}
}
