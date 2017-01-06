// Copyright 2013-2015 go-redis authors.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//
// This is a modified version of gomemcache adapted to redis.
// Original code and license at https://github.com/bradfitz/gomemcache/

package redis

import (
	"errors"
	"fmt"
	"hash/crc32"
	"net"
	"strings"
	"sync"
)

// ServerSelector is the interface that selects a redis server as a function
// of the item's key.
//
// All ServerSelector implementations must be threadsafe.
type ServerSelector interface {
	// PickServer returns the server address that a given item
	// should be shared onto, or the first listed server if an
	// empty key is given.
	PickServer(key string) (ServerInfo, error)

	// Sharding indicates that the client can connect to multiple servers.
	Sharding() bool
}

// ServerInfo stores parsed the server information.
type ServerInfo struct {
	Addr   net.Addr
	DB     string
	Passwd string
}

// ServerList is a simple ServerSelector. Its zero value is usable.
type ServerList struct {
	lk       sync.RWMutex
	servers  []ServerInfo
	sharding bool
}

func parseOptions(srv *ServerInfo, opts []string) error {
	for _, opt := range opts {
		items := strings.Split(opt, "=")
		if len(items) != 2 {
			return errors.New("Unknown option " + opt)
		}
		switch items[0] {
		case "db":
			srv.DB = items[1]
		case "passwd":
			srv.Passwd = items[1]
		default:
			return errors.New("Unknown option " + opt)
		}
	}
	return nil
}

// SetServers changes a ServerList's set of servers at runtime and is
// threadsafe.
//
// Each server is given equal weight. A server is given more weight
// if it's listed multiple times.
//
// SetServers returns an error if any of the server names fail to
// resolve. No attempt is made to connect to the server. If any error
// is returned, no changes are made to the ServerList.
func (ss *ServerList) SetServers(servers ...string) error {
	var err error
	var fs, addr net.Addr
	nsrv := make([]ServerInfo, len(servers))
	for i, server := range servers {
		// addr db=N passwd=foobar
		items := strings.Split(server, " ")
		if strings.Contains(items[0], "/") {
			addr, err = net.ResolveUnixAddr("unix", items[0])
		} else {
			addr, err = net.ResolveTCPAddr("tcp", items[0])
		}
		if err != nil {
			return fmt.Errorf(
				"Invalid redis server '%s': %s",
				server, err)
		}
		nsrv[i].Addr = addr
		// parse connection options
		if len(items) > 1 {
			if err := parseOptions(&nsrv[i], items[1:]); err != nil {
				return fmt.Errorf(
					"Invalid redis server '%s': %s",
					server, err)
			}
		}
		if i == 0 {
			fs = addr
		} else if fs != addr && !ss.sharding {
			ss.sharding = true
		}
	}
	ss.lk.Lock()
	defer ss.lk.Unlock()
	ss.servers = nsrv
	return nil
}

// Sharding implements the ServerSelector interface.
func (ss *ServerList) Sharding() bool {
	return ss.sharding
}

// PickServer implements the ServerSelector interface.
func (ss *ServerList) PickServer(key string) (srv ServerInfo, err error) {
	ss.lk.RLock()
	defer ss.lk.RUnlock()
	if len(ss.servers) == 0 {
		err = ErrNoServers
		return
	}
	if key == "" {
		return ss.servers[0], nil
	}
	return ss.servers[crc32.ChecksumIEEE([]byte(key))%uint32(len(ss.servers))], nil
}
