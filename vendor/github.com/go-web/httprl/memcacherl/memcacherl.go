// Package memcacherl is a memcache client wrapper for rate limiting.
package memcacherl

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// Client is a memcache client wrapper suitable for rate limiting.
type Client struct {
	mc *memcache.Client
}

// New creates and initializes a new Client.
func New(mc *memcache.Client) *Client {
	return &Client{mc}
}

// Hit implements the httprl.Backend interface.
func (c *Client) Hit(key string, ttlsec int32) (count uint64, remttl int32, err error) {
	item, err := c.mc.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return c.create(key, ttlsec)
		}
		return 0, 0, err
	}
	n, err := c.mc.Increment(key, 1)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return c.create(key, ttlsec)
		}
		return 0, 0, err
	}
	rem := int32(item.Flags) - int32(time.Now().Unix())
	if rem < 0 {
		rem = 0
	}
	return n, rem, nil
}

func (c *Client) create(key string, ttlsec int32) (uint64, int32, error) {
	exp := uint32(time.Now().Unix()) + uint32(ttlsec)
	item := &memcache.Item{
		Key:        key,
		Value:      []byte{'1'},
		Flags:      exp, // unix time of exp date
		Expiration: ttlsec,
	}
	err := c.mc.Set(item)
	if err != nil {
		return 0, 0, err
	}
	return 1, ttlsec, nil
}
