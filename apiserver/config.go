// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/fiorix/freegeoip"
)

// Config is the configuration of the freegeoip server.
type Config struct {
	ServerAddr         string
	TLSServerAddr      string
	TLSCertFile        string
	TLSKeyFile         string
	APIPrefix          string
	CORSOrigin         string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PublicDir          string
	DB                 string
	UpdateInterval     time.Duration
	RetryInterval      time.Duration
	UseXForwardedFor   bool
	Silent             bool
	LogToStdout        bool
	LogTimestamp       bool
	RedisAddr          string
	RedisTimeout       time.Duration
	MemcacheAddr       string
	MemcacheTimeout    time.Duration
	RateLimitBackend   string
	RateLimitLimit     uint64
	RateLimitInterval  time.Duration
	InternalServerAddr string

	errorLog  *log.Logger
	accessLog *log.Logger
}

// NewConfig creates and initializes a new Config with default values.
func NewConfig() *Config {
	return &Config{
		ServerAddr:        ":8080",
		TLSCertFile:       "cert.pem",
		TLSKeyFile:        "key.pem",
		APIPrefix:         "/",
		CORSOrigin:        "*",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      15 * time.Second,
		DB:                freegeoip.MaxMindDB,
		UpdateInterval:    24 * time.Hour,
		RetryInterval:     2 * time.Hour,
		LogTimestamp:      true,
		RedisAddr:         "localhost:6379",
		RedisTimeout:      time.Second,
		MemcacheAddr:      "localhost:11211",
		MemcacheTimeout:   time.Second,
		RateLimitBackend:  "redis",
		RateLimitInterval: time.Hour,
	}
}

// AddFlags adds configuration flags to the given FlagSet.
func (c *Config) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.ServerAddr, "http", c.ServerAddr, "Address in form of ip:port to listen on for HTTP")
	fs.StringVar(&c.TLSServerAddr, "https", c.TLSServerAddr, "Address in form of ip:port to listen on for HTTPS")
	fs.StringVar(&c.TLSCertFile, "cert", c.TLSCertFile, "X.509 certificate file for HTTPS server")
	fs.StringVar(&c.TLSKeyFile, "key", c.TLSKeyFile, "X.509 key file for HTTPS server")
	fs.StringVar(&c.APIPrefix, "api-prefix", c.APIPrefix, "URL prefix for API endpoints")
	fs.StringVar(&c.CORSOrigin, "cors-origin", c.CORSOrigin, "CORS origin API endpoints")
	fs.DurationVar(&c.ReadTimeout, "read-timeout", c.ReadTimeout, "Read timeout for HTTP and HTTPS client conns")
	fs.DurationVar(&c.WriteTimeout, "write-timeout", c.WriteTimeout, "Write timeout for HTTP and HTTPS client conns")
	fs.StringVar(&c.PublicDir, "public", c.PublicDir, "Public directory to serve at the {prefix}/ endpoint")
	fs.StringVar(&c.DB, "db", c.DB, "IP database file or URL")
	fs.DurationVar(&c.UpdateInterval, "update", c.UpdateInterval, "Database update check interval")
	fs.DurationVar(&c.RetryInterval, "retry", c.RetryInterval, "Max time to wait before retrying to download database")
	fs.BoolVar(&c.UseXForwardedFor, "use-x-forwarded-for", c.UseXForwardedFor, "Use the X-Forwarded-For header when available (e.g. behind proxy)")
	fs.BoolVar(&c.Silent, "silent", c.Silent, "Disable HTTP and HTTPS log request details")
	fs.BoolVar(&c.LogToStdout, "logtostdout", c.LogToStdout, "Log to stdout instead of stderr")
	fs.BoolVar(&c.LogTimestamp, "logtimestamp", c.LogTimestamp, "Prefix non-access logs with timestamp")
	fs.StringVar(&c.RedisAddr, "redis", c.RedisAddr, "Redis address in form of host:port[,host:port] for quota")
	fs.DurationVar(&c.RedisTimeout, "redis-timeout", c.RedisTimeout, "Redis read/write timeout")
	fs.StringVar(&c.MemcacheAddr, "memcache", c.MemcacheAddr, "Memcache address in form of host:port[,host:port] for quota")
	fs.DurationVar(&c.MemcacheTimeout, "memcache-timeout", c.MemcacheTimeout, "Memcache read/write timeout")
	fs.StringVar(&c.RateLimitBackend, "quota-backend", c.RateLimitBackend, "Backend for rate limiter: map, redis, or memcache")
	fs.Uint64Var(&c.RateLimitLimit, "quota-max", c.RateLimitLimit, "Max requests per source IP per interval; set 0 to turn quotas off")
	fs.DurationVar(&c.RateLimitInterval, "quota-interval", c.RateLimitInterval, "Quota expiration interval, per source IP querying the API")
	fs.StringVar(&c.InternalServerAddr, "internal-server", c.InternalServerAddr, "Address in form of ip:port to listen on for metrics and pprof")
}

func (c *Config) logWriter() io.Writer {
	if c.LogToStdout {
		return os.Stdout
	}
	return os.Stderr
}

func (c *Config) errorLogger() *log.Logger {
	if c.LogTimestamp {
		return log.New(c.logWriter(), "[error] ", log.LstdFlags)
	}
	return log.New(c.logWriter(), "[error] ", 0)
}

func (c *Config) accessLogger() *log.Logger {
	return log.New(c.logWriter(), "[access] ", 0)
}
