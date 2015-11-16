// Copyright 2009-2015 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"net/http"
	"strings"
)

// cors is an HTTP handler for managing cross-origin resource sharing.
// Ref: https://en.wikipedia.org/wiki/Cross-origin_resource_sharing.
func cors(f http.Handler, origin string, methods ...string) http.Handler {
	ms := strings.Join(methods, ", ") + ", OPTIONS"
	md := make(map[string]struct{})
	for _, method := range methods {
		md[method] = struct{}{}
	}
	cf := func(w http.ResponseWriter, r *http.Request) {
		orig := origin
		if orig == "*" {
			if ro := r.Header.Get("Origin"); ro != "" {
				orig = ro
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", orig)
		w.Header().Set("Access-Control-Allow-Methods", ms)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if _, exists := md[r.Method]; exists {
			f.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Allow", ms)
		http.Error(w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
	}
	return http.HandlerFunc(cf)
}
