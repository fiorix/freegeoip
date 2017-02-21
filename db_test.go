// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testFile = "testdata/db.gz"

func TestMaxMindUpdateURL(t *testing.T) {
	UserID := "hello"
	LicenseKey := "world"
	u, err := MaxMindUpdateURL(
		"updates.maxmind.com",
		"GeoIP2-City",
		UserID,
		LicenseKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	uu, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := uu.Query()
	switch {
	case uu.Scheme != "https":
		t.Fatalf("unexpected url scheme: want https, have %q", uu.Scheme)
	case uu.Host != "updates.maxmind.com":
		t.Fatalf("unexpected url host: want updates.maxmind.com, have %q", uu.Host)
	case len(q["db_md5"]) == 0:
		t.Fatal("missing db_md5 param")
	case len(q["challenge_md5"]) == 0:
		t.Fatal("missing challenge_md5 param")
	case len(q["user_id"]) == 0 || q["user_id"][0] != UserID:
		t.Fatalf("unexpected user id: want %q, have %q", UserID, q["user_id"])
	case len(q["edition_id"]) == 0 || q["edition_id"][0] != "GeoIP2-City":
		t.Fatalf("unexpected edition_id: %q", q["edition_id"])
	}
}

func TestDownload(t *testing.T) {
	if _, err := os.Stat(testFile); err == nil {
		t.Skip("Test database already exists:", testFile)
	}
	db := &DB{}
	dbfile, err := db.download(MaxMindDB)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Rename(dbfile, testFile)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNeedUpdateFileMissing(t *testing.T) {
	db := &DB{file: "does-not-exist"}
	yes, err := db.needUpdate("whatever")
	if err != nil {
		t.Fatal(err)
	}
	if !yes {
		t.Fatal("Unexpected: db is supposed to need an update")
	}
}

func TestNeedUpdateSameFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/testdata/", http.FileServer(http.Dir(".")))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	db := &DB{file: testFile}
	yes, err := db.needUpdate(srv.URL + "/" + testFile)
	if err != nil {
		t.Fatal(err)
	}
	if yes {
		t.Fatal("Unexpected: db is not supposed to need an update")
	}
}

func TestNeedUpdateSameMD5(t *testing.T) {
  db := &DB{file: testFile}
  _, checksum, err := db.newReader(db.file)
  if err != nil {
    t.Fatal(err)
  }
  db.checksum = checksum
	mux := http.NewServeMux()
  changeHeaderThenServe := func(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
      w.Header().Add("X-Database-MD5", checksum)
      h.ServeHTTP(w, r)
    }
  }
  mux.Handle("/testdata/", changeHeaderThenServe(http.FileServer(http.Dir("."))))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	yes, err := db.needUpdate(srv.URL + "/" + testFile)
	if err != nil {
		t.Fatal(err)
	}
	if yes {
		t.Fatal("Unexpected: db is not supposed to need an update")
	}
}

func TestNeedUpdateMD5(t *testing.T) {
	mux := http.NewServeMux()
  changeHeaderThenServe := func(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
      w.Header().Add("X-Database-MD5", "9823y5981y2398y1234")
      h.ServeHTTP(w, r)
    }
  }
  mux.Handle("/testdata/", changeHeaderThenServe(http.FileServer(http.Dir("."))))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	db := &DB{file: testFile}
	yes, err := db.needUpdate(srv.URL + "/" + testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !yes {
		t.Fatal("Unexpected: db is supposed to need an update")
	}
}

func TestNeedUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/testdata/", http.FileServer(http.Dir(".")))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	file := testFile + ".tmp"
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(file)
	db := &DB{file: file}
	yes, err := db.needUpdate(srv.URL + "/" + testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !yes {
		t.Fatal("Unexpected: db is supposed to need an update")
	}
}

func TestOpenFile(t *testing.T) {
	db, err := Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	select {
	case <-db.NotifyOpen():
	case <-db.NotifyClose():
	case <-time.After(time.Second):
		t.Fatal("Timed out")
	}
	db.Date() // Test this?
}

func TestOpenBadFile(t *testing.T) {
	db, err := Open("db_test.go")
	if err == nil {
		db.Close()
		t.Fatal("Unexpected bogus db is open")
	}
}

func TestSendError(t *testing.T) {
	db := &DB{notifyError: make(chan error, 1)}
	err1 := errors.New("test")
	db.sendError(err1)
	select {
	case err2 := <-db.NotifyError():
		if err2 != err2 {
			t.Fatalf("Unexpected error: %#v", err2)
		}
	default:
		t.Fatal("An error is expected but it's not available")
	}
}

func TestSkipSendError(t *testing.T) {
	db := &DB{notifyError: make(chan error, 1)}
	db.sendError(nil)
	db.sendError(nil)
	close(db.notifyError)
}

func TestWatchFile(t *testing.T) {
	db, err := Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = os.Rename(testFile, testFile+".bkp")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Rename(testFile+".bkp", testFile)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case file := <-db.NotifyOpen():
		if file != testFile {
			t.Fatal("Unexpected file:", file)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out")
	}
}

func TestWatchMkdir(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/testdata/", http.FileServer(http.Dir(".")))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	tmp := defaultDB
	defaultDB = filepath.Join(os.TempDir(), "foobar", "db.gz")
	defer func() {
		defaultDB = tmp
		time.Sleep(time.Second)
		os.RemoveAll(filepath.Dir(defaultDB))
	}()
	db, err := OpenURL(srv.URL+"/"+testFile, time.Hour, time.Minute)
	if err != nil {
		t.Fatalf("Failed to create %s: %s", filepath.Dir(defaultDB), err)
	}
	db.Close()
}

func TestWatchMkdirFail(t *testing.T) {
	basedir := filepath.Join(os.TempDir(), "freegeoip-test")
	err := os.MkdirAll(basedir, 0444)
	if err != nil {
		t.Fatal(err)
	}
	tmp := defaultDB
	defaultDB = filepath.Join(basedir, "a", "db.gz")
	defer func() {
		defaultDB = tmp
		time.Sleep(time.Second)
		os.Chmod(basedir, 0755)
		os.RemoveAll(basedir)
	}()
	mux := http.NewServeMux()
	mux.Handle("/testdata/", http.FileServer(http.Dir(".")))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	db, err := OpenURL(srv.URL+"/"+testFile, time.Hour, time.Minute)
	if err == nil {
		db.Close()
		t.Fatalf("Unexpected creation of dir %s worked", basedir)
	}
}

func TestLookupOnFile(t *testing.T) {
	db, err := Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var record DefaultQuery
	err = db.Lookup(net.ParseIP("8.8.8.8"), &record)
	if err != nil {
		t.Fatal(err)
	}
	if record.Country.ISOCode != "US" {
		t.Fatal("Unexpected ISO code:", record.Country.ISOCode)
	}
}

func TestLookupOnURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/testdata/", http.FileServer(http.Dir(".")))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	os.Remove(defaultDB) // In case it exists.
	db, err := OpenURL(srv.URL+"/"+testFile, time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	select {
	case file := <-db.NotifyOpen():
		if file != defaultDB {
			t.Fatal("Unexpected db file:", file)
		}
	case err := <-db.NotifyError():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out")
	}
	var record DefaultQuery
	err = db.Lookup(net.ParseIP("8.8.8.8"), &record)
	if err != nil {
		t.Fatal(err)
	}
	if record.Country.ISOCode != "US" {
		t.Fatal("Unexpected ISO code:", record.Country.ISOCode)
	}
}

func TestLookuUnavailable(t *testing.T) {
	db := &DB{}
	err := db.Lookup(net.ParseIP("8.8.8.8"), nil)
	if err == nil {
		t.Fatal("Unexpected lookup worked")
	}
}
