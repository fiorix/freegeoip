// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"flag"
	"testing"
	"time"
)

func TestCmd(t *testing.T) {
	flag.Set("http", ":0")
	flag.Set("db", "../testdata/db.gz")
	flag.Set("silent", "true")
	errc := make(chan error)
	go func() {
		errc <- Run()
	}()
	select {
	case err := <-errc:
		t.Fatal(err)
	case <-time.After(time.Second):
	}
}
