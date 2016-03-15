// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"flag"
	"testing"
)

func TestConfig(t *testing.T) {
	c := NewConfig()
	c.AddFlags(flag.NewFlagSet("freegeoip", flag.ContinueOnError))
}
