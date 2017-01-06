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

package redis

// WORK IN PROGRESS
//
// Redis commands
//
// Some commands take an integer timeout, in seconds. It's not a time.Duration
// because redis only supports seconds resolution for timeouts.
//
// Redis allows clients to block indefinitely by setting timeout to 0, but
// it does not work here. All functions below use the timeout not only to
// block the operation in redis, but also as a socket read timeout (+delta)
// to free up system resources.
//
// The default TCP read timeout is 200ms. If a timeout is required to
// be "indefinitely", then set it to something like 86400.
//
// See redis.DefaultTimeout for details.
//
// 🍺

import (
	"errors"
	"strings"
	"time"
)

// Append implements http://redis.io/commands/append.
func (c *Client) Append(key, value string) (int, error) {
	v, err := c.execWithKey(true, "append", key, value)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// BgRewriteAOF implements http://redis.io/commands/bgrewriteaof.
// Cannot be sharded.
func (c *Client) BgRewriteAOF() (string, error) {
	v, err := c.execOnFirst(false, "BGREWRITEAOF")
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// BgSave implements http://redis.io/commands/bgsave.
// Cannot be sharded.
func (c *Client) BgSave() (string, error) {
	v, err := c.execOnFirst(false, "BGSAVE")
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// Ping implements http://redis.io/commands/ping.
// Cannot be sharded.
func (c *Client) Ping() error {
	v, err := c.execOnFirst(false, "PING")
	if err != nil {
		return err
	}
	s, err := iface2str(v)
	if err != nil {
		return err
	} else if s != "PONG" {
		return ErrServerError
	}
	return nil
}

// BitCount implements http://redis.io/commands/bitcount.
// Start and end are ignored if start is a negative number.
func (c *Client) BitCount(key string, start, end int) (int, error) {
	var (
		v   interface{}
		err error
	)
	if start > -1 {
		v, err = c.execWithKey(true, "BITCOUNT", key, start, end)
	} else {
		v, err = c.execWithKey(true, "BITCOUNT", key)
	}
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// BitOp implements http://redis.io/commands/bitop.
// Cannot be sharded.
func (c *Client) BitOp(operation, destkey, key string, keys ...string) (int, error) {
	a := append([]string{"BITOP", operation, destkey, key}, keys...)
	v, err := c.execOnFirst(true, vstr2iface(a)...)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// blbrPop implements both BLPop and BRPop.
func (c *Client) blbrPop(cmd string, timeout int, keys ...string) (k, v string, err error) {
	var r interface{}
	r, err = c.execWithKeyTimeout(
		true,
		timeout,
		cmd,
		keys[0],
		append(vstr2iface(keys[1:]), timeout)...,
	)
	if err != nil {
		return "", "", err
	}
	if r == nil {
		return "", "", ErrTimedOut
	}
	switch r.(type) {
	case []interface{}:
		items := r.([]interface{})
		if len(items) != 2 {
			return "", "", ErrServerError
		}
		// TODO: test types
		k = items[0].(string)
		v = items[1].(string)
		return k, v, nil
	}
	return "", "", ErrServerError
}

// BLPop implements http://redis.io/commands/blpop.
// Cannot be sharded.
//
// If timeout is 0, DefaultTimeout is used.
func (c *Client) BLPop(timeout int, keys ...string) (k, v string, err error) {
	return c.blbrPop("BLPOP", timeout, keys...)
}

// BRPop implements http://redis.io/commands/brpop.
// Cannot be sharded.
//
// If timeout is 0, DefaultTimeout is used.
func (c *Client) BRPop(timeout int, keys ...string) (k, v string, err error) {
	return c.blbrPop("BRPOP", timeout, keys...)
}

// BRPopLPush implements http://redis.io/commands/brpoplpush.
// Cannot be sharded.
//
// If timeout is 0, DefaultTimeout is used.
func (c *Client) BRPopLPush(src, dst string, timeout int) (string, error) {
	v, err := c.execWithKeyTimeout(true, timeout, "BRPOPLPUSH", src, dst, timeout)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", ErrTimedOut
	}
	return iface2str(v)
}

// ClientKill implements http://redis.io/commands/client-kill.
// Cannot be sharded.
func (c *Client) ClientKill(addr string) error {
	v, err := c.execOnFirst(false, "CLIENT KILL", addr)
	if err != nil {
		return err
	}
	switch v.(type) {
	case string:
		return nil
	}
	return ErrServerError
}

// ClientList implements http://redis.io/commands/client-list.
// Cannot be sharded.
func (c *Client) ClientList() ([]string, error) {
	v, err := c.execOnFirst(false, "CLIENT LIST")
	if err != nil {
		return nil, err
	}
	switch v.(type) {
	case string:
		return strings.Split(v.(string), "\n"), nil
	}
	return nil, ErrServerError
}

// ClientSetName implements http://redis.io/commands/client-setname.
// Cannot be sharded.
//
// This driver creates connections on demand, thus naming them is pointless.
func (c *Client) ClientSetName(name string) error {
	v, err := c.execOnFirst(false, "CLIENT SETNAME", name)
	if err != nil {
		return err
	}
	switch v.(type) {
	case string:
		return nil
	}
	return ErrServerError
}

// ConfigGet implements http://redis.io/commands/config-get.
// Cannot be sharded.
func (c *Client) ConfigGet(name string) (map[string]string, error) {
	v, err := c.execOnFirst(false, "CONFIG GET", name)
	if err != nil {
		return nil, err
	}
	return iface2strmap(v), nil
}

// ConfigSet implements http://redis.io/commands/config-set.
// Cannot be sharded.
func (c *Client) ConfigSet(name, value string) error {
	v, err := c.execOnFirst(false, "CONFIG SET", name, value)
	if err != nil {
		return err
	}
	switch v.(type) {
	case string:
		return nil
	}
	return ErrServerError
}

// ConfigResetStat implements http://redis.io/commands/config-resetstat.
// Cannot be sharded.
func (c *Client) ConfigResetStat() error {
	v, err := c.execOnFirst(false, "CONFIG RESETSTAT")
	if err != nil {
		return err
	}
	switch v.(type) {
	case string:
		return nil
	}
	return ErrServerError
}

// DBSize implements http://redis.io/commands/dbsize.
// Cannot be sharded.
func (c *Client) DBSize() (int, error) {
	v, err := c.execOnFirst(false, "DBSIZE")
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// DebugSegfault implements http://redis.io/commands/debug-segfault.
// Cannot be sharded.
func (c *Client) DebugSegfault() error {
	v, err := c.execOnFirst(false, "DEBUG SEGFAULT")
	if err != nil {
		return err
	}
	switch v.(type) {
	case string:
		return nil
	}
	return ErrServerError
}

// Decr implements http://redis.io/commands/decr.
func (c *Client) Decr(key string) (int, error) {
	v, err := c.execWithKey(true, "DECR", key)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// DecrBy implements http://redis.io/commands/decrby.
func (c *Client) DecrBy(key string, decrement int) (int, error) {
	v, err := c.execWithKey(true, "DECRBY", key, decrement)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// Del implements http://redis.io/commands/del.
// Del issues a plain DEL command to redis if the client is connected to
// a single server. On sharded connections, it issues one DEL command per
// key, in the server selected for each given key.
func (c *Client) Del(keys ...string) (n int, err error) {
	if c.selector.Sharding() {
		n, err = c.delMulti(keys...)
	} else {
		n, err = c.delPlain(keys...)
	}
	return n, err
}

func (c *Client) delMulti(keys ...string) (int, error) {
	deleted := 0
	for _, key := range keys {
		count, err := c.delPlain(key)
		if err != nil {
			return 0, err
		}
		deleted += count
	}
	return deleted, nil
}

func (c *Client) delPlain(keys ...string) (int, error) {
	if len(keys) > 0 {
		v, err := c.execWithKey(true, "DEL", keys[0], vstr2iface(keys[1:])...)
		if err != nil {
			return 0, err
		}
		return iface2int(v)
	}
	return 0, nil
}

// http://redis.io/commands/discard
// TODO: Discard

// Dump implements http://redis.io/commands/dump.
func (c *Client) Dump(key string) (string, error) {
	v, err := c.execWithKey(true, "DUMP", key)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// Echo implements http://redis.io/commands/echo.
func (c *Client) Echo(message string) (string, error) {
	v, err := c.execWithKey(true, "ECHO", message)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// Eval implemenets http://redis.io/commands/eval.
// Cannot be sharded.
func (c *Client) Eval(script string, numkeys int, keys, args []string) (interface{}, error) {
	a := []interface{}{
		"EVAL",
		script, // escape?
		numkeys,
		strings.Join(keys, " "),
		strings.Join(args, " "),
	}
	v, err := c.execOnFirst(true, a...)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// EvalSha implements http://redis.io/commands/evalsha.
// Cannot be sharded.
func (c *Client) EvalSha(sha1 string, numkeys int, keys, args []string) (interface{}, error) {
	a := []interface{}{
		"EVALSHA",
		sha1,
		numkeys,
		strings.Join(keys, " "),
		strings.Join(args, " "),
	}
	v, err := c.execOnFirst(true, a...)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// http://redis.io/commands/exec
// TODO: Exec

// Exists implements http://redis.io/commands/exists.
func (c *Client) Exists(key string) (bool, error) {
	v, err := c.execWithKey(true, "EXISTS", key)
	if err != nil {
		return false, err
	}
	return iface2bool(v)
}

// Expire implements http://redis.io/commands/expire.
// Expire returns true if a timeout was set for the given key,
// or false when key does not exist or the timeout could not be set.
func (c *Client) Expire(key string, seconds int) (bool, error) {
	v, err := c.execWithKey(true, "EXPIRE", key, seconds)
	if err != nil {
		return false, err
	}
	return iface2bool(v)
}

// ExpireAt implements http://redis.io/commands/expireat.
// ExpireAt behaves like Expire.
func (c *Client) ExpireAt(key string, timestamp int) (bool, error) {
	v, err := c.execWithKey(true, "EXPIREAT", key, timestamp)
	if err != nil {
		return false, err
	}
	return iface2bool(v)
}

// FlushAll implements http://redis.io/commands/flushall.
// Cannot be sharded.
func (c *Client) FlushAll() error {
	_, err := c.execOnFirst(false, "FLUSHALL")
	return err
}

// FlushDB implements http://redis.io/commands/flushall.
// Cannot be sharded.
func (c *Client) FlushDB() error {
	_, err := c.execOnFirst(false, "FLUSHDB")
	return err
}

// Get implements http://redis.io/commands/get.
func (c *Client) Get(key string) (string, error) {
	v, err := c.execWithKey(true, "GET", key)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// GetBit implements http://redis.io/commands/getbit.
func (c *Client) GetBit(key string, offset int) (int, error) {
	v, err := c.execWithKey(true, "GETBIT", key, offset)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// GetRange implements http://redis.io/commands/getrange.
func (c *Client) GetRange(key string, start, end int) (string, error) {
	v, err := c.execWithKey(true, "GETRANGE", key, start, end)
	if err != nil {
		return "", err
	}
	switch v.(type) {
	case string:
		return v.(string), nil
	}
	return "", ErrServerError
}

// GetSet implements http://redis.io/commands/getset.
func (c *Client) GetSet(key, value string) (string, error) {
	v, err := c.execWithKey(true, "GETSET", key, value)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// Incr implements http://redis.io/commands/incr.
func (c *Client) Incr(key string) (int, error) {
	v, err := c.execWithKey(true, "INCR", key)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// IncrBy implements http://redis.io/commands/incrby.
func (c *Client) IncrBy(key string, increment int) (int, error) {
	v, err := c.execWithKey(true, "INCRBY", key, increment)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// Keys implement http://redis.io/commands/keys.
// Cannot be sharded.
func (c *Client) Keys(pattern string) ([]string, error) {
	keys := []string{}
	v, err := c.execOnFirst(true, "KEYS", pattern)
	if err != nil {
		return keys, err
	}
	return iface2vstr(v), nil
}

// Scan implements http://redis.io/commands/scan.
func (c *Client) Scan(cursor string, options ...interface{}) (string, []string, error) {
	return c.scanCommandList("SCAN", "", cursor, options...)
}

// SScan implements http://redis.io/commands/sscan.
func (c *Client) SScan(set string, cursor string, options ...interface{}) (string, []string, error) {
	return c.scanCommandList("SSCAN", set, cursor, options...)
}

// ZScan implements http://redis.io/commands/zscan.
func (c *Client) ZScan(zset string, cursor string, options ...interface{}) (string, map[string]string, error) {
	return c.scanCommandMap("ZSCAN", zset, cursor, options...)
}

// HScan implements http://redis.io/commands/hscan.
func (c *Client) HScan(hash string, cursor string, options ...interface{}) (string, map[string]string, error) {
	return c.scanCommandMap("HSCAN", hash, cursor, options...)
}

// SCAN and SSCAN
func (c *Client) scanCommandList(cmd string, key string, cursor string, options ...interface{}) (string, []string, error) {
	empty := []string{}
	resp := []interface{}{}
	newCursor := "0"

	var v interface{}
	var err error

	if len(key) > 0 { // SSCAN
		x := []interface{}{cursor}
		v, err = c.execWithKey(true, cmd, key, append(x, options...)...)
	} else { // SCAN
		x := []interface{}{cmd, cursor}
		v, err = c.execOnFirst(true, append(x, options...)...)
	}

	if err != nil {
		return newCursor, empty, err
	}

	switch v.(type) {
	case []interface{}:
		resp = v.([]interface{})
	}

	// New cursor to call
	switch resp[0].(type) {
	case string:
		newCursor = resp[0].(string)
	}

	switch resp[1].(type) {
	case []interface{}:
		return newCursor, iface2vstr(resp[1]), nil
	}

	return newCursor, empty, nil
}

// ZSCAN and HSCAN
func (c *Client) scanCommandMap(cmd string, key string, cursor string, options ...interface{}) (string, map[string]string, error) {
	empty := map[string]string{}
	resp := []interface{}{}
	newCursor := "0"

	x := []interface{}{cursor}
	v, err := c.execWithKey(true, cmd, key, append(x, options...)...)

	if err != nil {
		return newCursor, empty, err
	}

	switch v.(type) {
	case []interface{}:
		resp = v.([]interface{})
	}

	// New cursor to call
	switch resp[0].(type) {
	case string:
		newCursor = resp[0].(string)
	}

	switch resp[1].(type) {
	case []interface{}:
		return newCursor, iface2strmap(resp[1]), nil
	}

	return newCursor, empty, nil
}

// LPush implements http://redis.io/commands/lpush.
func (c *Client) LPush(key string, values ...string) (int, error) {
	v, err := c.execWithKey(true, "LPUSH", key, vstr2iface(values)...)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// LIndex implements http://redis.io/commands/lindex.
func (c *Client) LIndex(key string, index int) (string, error) {
	v, err := c.execWithKey(true, "LINDEX", key, index)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// LPop implements http://redis.io/commands/lpop.
func (c *Client) LPop(key string) (string, error) {
	v, err := c.execWithKey(true, "LPOP", key)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// RPop implements http://redis.io/commands/rpop.
func (c *Client) RPop(key string) (string, error) {
	v, err := c.execWithKey(true, "RPOP", key)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// LLen implements http://redis.io/commands/llen.
func (c *Client) LLen(key string) (int, error) {
	v, err := c.execWithKey(true, "LLEN", key)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// LTrim implements http://redis.io/commands/ltrim.
func (c *Client) LTrim(key string, begin, end int) (err error) {
	_, err = c.execWithKey(true, "LTRIM", key, begin, end)
	return err
}

// LRange implements http://redis.io/commands/lrange.
func (c *Client) LRange(key string, begin, end int) ([]string, error) {
	v, err := c.execWithKey(true, "LRANGE", key, begin, end)
	if err != nil {
		return []string{}, err
	}
	return iface2vstr(v), nil
}

// LRem implements http://redis.io/commands/lrem.
func (c *Client) LRem(key string, count int, value string) (int, error) {
	v, err := c.execWithKey(true, "LREM", key, count, value)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// HGet implements http://redis.io/commands/hget.
func (c *Client) HGet(key, member string) (string, error) {
	v, err := c.execWithKey(true, "HGET", key, member)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// HGetAll implements http://redis.io/commands/hgetall.
func (c *Client) HGetAll(key string) (map[string]string, error) {
	v, err := c.execWithKey(true, "HGETALL", key)
	if err != nil {
		return nil, err
	}
	return iface2strmap(v), nil
}

// HIncrBy implements http://redis.io/commands/hincrby.
func (c *Client) HIncrBy(key string, field string, increment int) (int, error) {
	v, err := c.execWithKey(true, "HINCRBY", key, field, increment)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// HMGet implements http://redis.io/commands/hmget.
func (c *Client) HMGet(key string, field ...string) ([]string, error) {
	v, err := c.execWithKey(true, "HMGET", key, vstr2iface(field)...)
	if err != nil {
		return nil, err
	}
	return iface2vstr(v), nil
}

// HMSet implements http://redis.io/commands/hmset.
func (c *Client) HMSet(key string, items map[string]string) (err error) {
	tmp := make([]interface{}, (len(items) * 2))
	idx := 0
	for k, v := range items {
		n := idx * 2
		tmp[n] = k
		tmp[n+1] = v
		idx++
	}
	_, err = c.execWithKey(true, "HMSET", key, tmp...)
	return
}

// HSet implements http://redis.io/commands/hset.
func (c *Client) HSet(key, field, value string) (err error) {
	_, err = c.execWithKey(true, "HSET", key, field, value)
	return
}

// HDel implements http://redis.io/commands/hdel.
func (c *Client) HDel(key, field string) (err error) {
	_, err = c.execWithKey(true, "HDEL", key, field)
	return
}

// ZIncrBy implements http://redis.io/commands/zincrby.
func (c *Client) ZIncrBy(key string, increment int, member string) (string, error) {
	v, err := c.execWithKey(true, "ZINCRBY", key, increment, member)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// MGet implements http://redis.io/commands/mget.
// Cannot be sharded.
//
// TODO: support sharded connections.
func (c *Client) MGet(keys ...string) ([]string, error) {
	tmp := make([]interface{}, len(keys)+1)
	tmp[0] = "MGET"
	for n, k := range keys {
		tmp[n+1] = k
	}
	v, err := c.execOnFirst(true, tmp...)
	if err != nil {
		return nil, err
	}
	switch v.(type) {
	case []interface{}:
		items := v.([]interface{})
		resp := make([]string, len(items))
		for n, item := range items {
			switch item.(type) {
			case string:
				resp[n] = item.(string)
			}
		}
		return resp, nil
	}
	return nil, ErrServerError
}

// MSet implements http://redis.io/commands/mset.
// Cannot be sharded.
//
// TODO: support sharded connections.
func (c *Client) MSet(items map[string]string) error {
	tmp := make([]interface{}, (len(items)*2)+1)
	tmp[0] = "MSET"
	idx := 0
	for k, v := range items {
		n := idx * 2
		tmp[n+1] = k
		tmp[n+2] = v
		idx++
	}
	_, err := c.execOnFirst(true, tmp...)
	if err != nil {
		return err
	}
	return nil
}

// PFAdd implements http://redis.io/commands/pfadd.
func (c *Client) PFAdd(key string, vs ...interface{}) (int, error) {
	v, err := c.execWithKey(true, "PFADD", key, vs...)

	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// PFCount implements http://redis.io/commands/pfcount.
func (c *Client) PFCount(keys ...string) (int, error) {
	v, err := c.execWithKeys(true, "PFCOUNT", keys)
	if err != nil {
		return 0, err
	}
	sum := 0

	if len(v) == 0 {
		return 0, nil
	}

	for _, value := range v {
		a, err := iface2int(value)
		if err != nil {
			return 0, err
		}
		sum += a
	}
	return iface2int(sum)
}

// PFMerge implements http://redis.io/commands/pfmerge.
func (c *Client) PFMerge(keys ...string) (err error) {
	_, err = c.execWithKeys(true, "PFMERGE", keys)
	return
}

// Publish implements http://redis.io/commands/publish.
func (c *Client) Publish(channel, message string) error {
	_, err := c.execWithKey(true, "PUBLISH", channel, message)
	return err
}

// RPush implements http://redis.io/commands/rpush.
func (c *Client) RPush(key string, values ...string) (int, error) {
	v, err := c.execWithKey(true, "RPUSH", key, vstr2iface(values)...)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// SAdd implements http://redis.io/commands/sadd.
func (c *Client) SAdd(key string, vs ...interface{}) (int, error) {
	v, err := c.execWithKey(true, "SADD", key, vs...)

	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// SRem implements http://redis.io/commands/srem.
func (c *Client) SRem(key string, vs ...interface{}) (int, error) {
	v, err := c.execWithKey(true, "SREM", key, vs...)

	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// ScriptLoad implements http://redis.io/commands/script-load.
// Cannot be sharded.
func (c *Client) ScriptLoad(script string) (string, error) {
	v, err := c.execOnFirst(true, "SCRIPT", "LOAD", script)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// Set implements http://redis.io/commands/set.
func (c *Client) Set(key, value string) (err error) {
	_, err = c.execWithKey(true, "SET", key, value)
	return
}

// SetNx implements http://redis.io/commands/setnx.
func (c *Client) SetNx(key, value string) (int, error) {
	v, err := c.execWithKey(true, "SETNX", key, value)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// SetBit implements http://redis.io/commands/setbit.
func (c *Client) SetBit(key string, offset, value int) (int, error) {
	v, err := c.execWithKey(true, "SETBIT", key, offset, value)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// SetEx implements http://redis.io/commands/setex.
func (c *Client) SetEx(key string, seconds int, value string) (err error) {
	_, err = c.execWithKey(true, "SETEX", key, seconds, value)
	return
}

// SMembers implements http://redis.io/commands/smembers.
func (c *Client) SMembers(key string) ([]string, error) {
	v, err := c.execWithKey(true, "SMEMBERS", key)
	if err != nil {
		return []string{}, err
	}
	return iface2vstr(v), nil
}

// SMove implements http://redis.io/commands/smove.
func (c *Client) SMove(source string, destination string, member string) (int, error) {
	v, err := c.execWithKey(true, "SMOVE", source, destination, member)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// SRandMember implements http://redis.io/commands/srandmember.
func (c *Client) SRandMember(key string, count int) ([]string, error) {
	v, err := c.execWithKey(true, "SRANDMEMBER", key, count)
	if err != nil {
		return []string{}, err
	}
	return iface2vstr(v), nil
}

// SIsMember implements http://redis.io/commands/sismember.
func (c *Client) SIsMember(key string, vs ...interface{}) (int, error) {
	v, err := c.execWithKey(true, "SISMEMBER", key, vs...)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// SCard implements http://redis.io/commands/scard.
func (c *Client) SCard(key string) (int, error) {
	v, err := c.execWithKey(true, "SCARD", key)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// A PubSubMessage carries redis pub/sub messages for clients
// that are subscribed to a topic. See Subscribe for details.
type PubSubMessage struct {
	Error   error
	Value   string
	Channel string
}

// Subscribe implements http://redis.io/commands/subscribe.
func (c *Client) Subscribe(channel string, m chan<- PubSubMessage, stop <-chan bool) error {
	srv, err := c.selector.PickServer("")
	if err != nil {
		return err
	}
	cn, err := c.getConn(srv)
	if err != nil {
		return err
	}
	// we cannot return this connection to the pool
	// because it'll be in subscribe context.
	//defer cn.condRelease(&err)
	_, err = c.execute(cn.rw, "SUBSCRIBE", channel)
	if err != nil {
		return err
	}
	if err = cn.nc.SetDeadline(time.Time{}); err != nil {
		cn.nc.Close()
		return err
	}
	watcher := make(chan struct{})
	go func() {
		select {
		case <-stop:
			cn.nc.Close()
		case <-watcher:
		}
	}()
	go func() {
		defer cn.nc.Close()
		for {
			raw, err := c.parseResponse(cn.rw.Reader)
			if err != nil {
				m <- PubSubMessage{
					Error: err,
				}
				close(watcher)
				return
			}
			switch raw.(type) {
			case []interface{}:
				ret := raw.([]interface{})
				m <- PubSubMessage{
					Value:   ret[2].(string),
					Channel: ret[1].(string),
					Error:   nil,
				}
			default:
				m <- PubSubMessage{
					Error: ErrServerError,
				}
				close(watcher)
				return
			}
		}
	}()
	return err
}

// TTL implements http://redis.io/commands/ttl.
func (c *Client) TTL(key string) (int, error) {
	v, err := c.execWithKey(true, "TTL", key)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// ZAdd implements http://redis.io/commands/zadd.
func (c *Client) ZAdd(key string, vs ...interface{}) (int, error) {
	if len(vs)%2 != 0 {
		return 0, errors.New("Incomplete parameter sequence")
	}
	v, err := c.execWithKey(true, "ZADD", key, vs...)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// ZCard implements http://redis.io/commands/zcard.
func (c *Client) ZCard(key string) (int, error) {
	v, err := c.execWithKey(true, "ZCARD", key)
	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// ZCount implements http://redis.io/commands/zcount.
func (c *Client) ZCount(key string, min int, max int) (int, error) {
	v, err := c.execWithKey(true, "ZCOUNT", key, min, max)

	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// ZRange implements http://redis.io/commands/zrange.
func (c *Client) ZRange(key string, start int, stop int, withscores bool) ([]string, error) {
	var v interface{}
	var err error
	if withscores == true {
		v, err = c.execWithKey(true, "ZRANGE", key, start, stop, "WITHSCORES")
	} else {
		v, err = c.execWithKey(true, "ZRANGE", key, start, stop)
	}
	if err != nil {
		return nil, err
	}
	return iface2vstr(v), nil
}

// ZRevRange implements http://redis.io/commands/zrevrange.
func (c *Client) ZRevRange(key string, start int, stop int, withscores bool) ([]string, error) {
	var v interface{}
	var err error
	if withscores == true {
		v, err = c.execWithKey(true, "ZREVRANGE", key, start, stop, "WITHSCORES")
	} else {
		v, err = c.execWithKey(true, "ZREVRANGE", key, start, stop)
	}
	if err != nil {
		return nil, err
	}
	return iface2vstr(v), nil
}

// ZRangeByScore implements http://redis.io/commands/ZRANGEBYSCORE
func (c *Client) ZRangeByScore(key string, min int, max int, withscores bool, limit bool, offset int, count int) ([]string, error) {
	var v interface{}
	var err error

	if withscores == true {
		if limit {
			v, err = c.execWithKey(true, "ZRANGEBYSCORE", key, min, max, "WITHSCORES", "LIMIT", offset, count)
		} else {
			v, err = c.execWithKey(true, "ZRANGEBYSCORE", key, min, max, "WITHSCORES")
		}
	} else {
		if limit {
			v, err = c.execWithKey(true, "ZRANGEBYSCORE", key, min, max, "LIMIT", offset, count)
		} else {
			v, err = c.execWithKey(true, "ZRANGEBYSCORE", key, min, max)
		}
	}

	if err != nil {
		return nil, err
	}
	return iface2vstr(v), nil
}

// ZRevRangeByScore implements http://redis.io/commands/ZREVRANGEBYSCORE
func (c *Client) ZRevRangeByScore(key string, max int, min int, withscores bool, limit bool, offset int, count int) ([]string, error) {
	var v interface{}
	var err error

	if withscores == true {
		if limit {
			v, err = c.execWithKey(true, "ZREVRANGEBYSCORE", key, max, min, "WITHSCORES", "LIMIT", offset, count)
		} else {
			v, err = c.execWithKey(true, "ZREVRANGEBYSCORE", key, max, min, "WITHSCORES")
		}
	} else {
		if limit {
			v, err = c.execWithKey(true, "ZREVRANGEBYSCORE", key, max, min, "LIMIT", offset, count)
		} else {
			v, err = c.execWithKey(true, "ZREVRANGEBYSCORE", key, max, min)
		}
	}

	if err != nil {
		return nil, err
	}
	return iface2vstr(v), nil
}

// ZScore implements http://redis.io/commands/zscore.
func (c *Client) ZScore(key string, member string) (string, error) {
	v, err := c.execWithKey(true, "ZSCORE", key, member)
	if err != nil {
		return "", err
	}
	return iface2str(v)
}

// ZRem implements http://redis.io/commands/zrem.
func (c *Client) ZRem(key string, vs ...interface{}) (int, error) {
	v, err := c.execWithKey(true, "ZREM", key, vs...)

	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// ZRemRangeByScore implements http://redis.io/commands/zremrangebyscore.
func (c *Client) ZRemRangeByScore(key string, start interface{}, stop interface{}) (int, error) {
	v, err := c.execWithKey(true, "ZREMRANGEBYSCORE", key, start, stop)

	if err != nil {
		return 0, err
	}
	return iface2int(v)
}

// GetMulti is a batch version of Get. The returned map from keys to
// items may have fewer elements than the input slice, due to memcache
// cache misses. Each key must be at most 250 bytes in length.
// If no error is returned, the returned map will also be non-nil.
/*
func (c *Client) GetMulti(keys []string) (map[string]*Item, error) {
	var lk sync.Mutex
	m := make(map[string]*Item)
	addItemToMap := func(it *Item) {
		lk.Lock()
		defer lk.Unlock()
		m[it.Key] = it
	}

	keyMap := make(map[net.Addr][]string)
	for _, key := range keys {
		if !legalKey(key) {
			return nil, ErrMalformedKey
		}
		addr, err := c.selector.PickServer(key)
		if err != nil {
			return nil, err
		}
		keyMap[addr] = append(keyMap[addr], key)
	}

	ch := make(chan error, buffered)
	for addr, keys := range keyMap {
		go func(addr net.Addr, keys []string) {
			//ch <- c.getFromAddr(addr, keys, addItemToMap)
		}(addr, keys)
	}

	var err error
	for _ = range keyMap {
		if ge := <-ch; ge != nil {
			err = ge
		}
	}
	return m, err
}
*/
