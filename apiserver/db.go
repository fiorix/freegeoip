// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"log"
	"net/url"
	"time"

	"github.com/fiorix/freegeoip"
)

// openDB opens and returns the IP database.
func openDB(dsn string, updateIntvl, maxRetryIntvl time.Duration) (db *freegeoip.DB, err error) {
	u, err := url.Parse(dsn)
	if err != nil || len(u.Scheme) == 0 {
		db, err = freegeoip.Open(dsn)
	} else {
		db, err = freegeoip.OpenURL(dsn, updateIntvl, maxRetryIntvl)
	}
	return
}

// watchEvents logs and collect metrics of database events.
func watchEvents(db *freegeoip.DB) {
	for {
		select {
		case file := <-db.NotifyOpen():
			log.Println("database loaded:", file)
			dbEventCounter.WithLabelValues("loaded").Inc()
		case err := <-db.NotifyError():
			log.Println("database error:", err)
			dbEventCounter.WithLabelValues("failed").Inc()
		case <-db.NotifyClose():
			return
		}
	}
}
