// Package redisrl is a redis client wrapper for rate limiting.
package redisrl

import "github.com/fiorix/go-redis/redis"

// Client is a redis client wrapper suitable for rate limiting.
type Client struct {
	rc *redis.Client
}

// New creates and initializes a new Client.
func New(rc *redis.Client) *Client {
	return &Client{rc}
}

// Hit implements the httprl.Backend interface.
func (c *Client) Hit(key string, ttlsec int32) (count uint64, remttl int32, err error) {
	rem, err := c.rc.TTL(key)
	if err != nil {
		return 0, 0, err
	}
	if rem <= 0 {
		return 1, ttlsec, c.rc.SetEx(key, int(ttlsec), "1")
	}
	n, err := c.rc.Incr(key)
	if err != nil {
		return 0, 0, err
	}
	return uint64(n), int32(rem), nil
}
