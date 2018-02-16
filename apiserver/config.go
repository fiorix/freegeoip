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

	"github.com/kelseyhightower/envconfig"

	"github.com/apilayer/freegeoip"
)

// Config is the configuration of the freegeoip server.
type Config struct {
	FastOpen            bool          `envconfig:"TCP_FAST_OPEN"`
	Naggle              bool          `envconfig:"TCP_NAGGLE"`
	ServerAddr          string        `envconfig:"HTTP"`
	HTTP2               bool          `envconfig:"HTTP2"`
	HSTS                string        `envconfig:"HSTS"`
	TLSServerAddr       string        `envconfig:"HTTPS"`
	TLSCertFile         string        `envconfig:"CERT"`
	TLSKeyFile          string        `envconfig:"KEY"`
	LetsEncrypt         bool          `envconfig:"LETSENCRYPT"`
	LetsEncryptCacheDir string        `envconfig:"LETSENCRYPT_CACHE_DIR"`
	LetsEncryptEmail    string        `envconfig:"LETSENCRYPT_EMAIL"`
	LetsEncryptHosts    string        `envconfig:"LETSENCRYPT_HOSTS"`
	APIPrefix           string        `envconfig:"API_PREFIX"`
	CORSOrigin          string        `envconfig:"CORS_ORIGIN"`
	ReadTimeout         time.Duration `envconfig:"READ_TIMEOUT"`
	WriteTimeout        time.Duration `envconfig:"WRITE_TIMEOUT"`
	PublicDir           string        `envconfig:"PUBLIC"`
	DB                  string        `envconfig:"DB"`
	UpdateInterval      time.Duration `envconfig:"UPDATE_INTERVAL"`
	RetryInterval       time.Duration `envconfig:"RETRY_INTERVAL"`
	UseXForwardedFor    bool          `envconfig:"USE_X_FORWARDED_FOR"`
	Silent              bool          `envconfig:"SILENT"`
	LogToStdout         bool          `envconfig:"LOGTOSTDOUT"`
	LogTimestamp        bool          `envconfig:"LOGTIMESTAMP"`
	RedisAddr           string        `envconfig:"REDIS"`
	RedisTimeout        time.Duration `envconfig:"REDIS_TIMEOUT"`
	MemcacheAddr        string        `envconfig:"MEMCACHE"`
	MemcacheTimeout     time.Duration `envconfig:"MEMCACHE_TIMEOUT"`
	RateLimitBackend    string        `envconfig:"QUOTA_BACKEND"`
	RateLimitLimit      uint64        `envconfig:"QUOTA_MAX"`
	RateLimitInterval   time.Duration `envconfig:"QUOTA_INTERVAL"`
	InternalServerAddr  string        `envconfig:"INTERNAL_SERVER"`
	UpdatesHost         string        `envconfig:"UPDATES_HOST"`
	LicenseKey          string        `envconfig:"LICENSE_KEY"`
	UserID              string        `envconfig:"USER_ID"`
	ProductID           string        `envconfig:"PRODUCT_ID"`
	NewrelicName        string        `envconfig:"NEWRELIC_NAME"`
	NewrelicKey         string        `envconfig:"NEWRELIC_KEY"`

	errorLog  *log.Logger
	accessLog *log.Logger
}

// NewConfig creates and initializes a new Config with default values.
func NewConfig() *Config {
	return &Config{
		FastOpen:            false,
		Naggle:              false,
		ServerAddr:          ":8080",
		HTTP2:               true,
		HSTS:                "",
		TLSCertFile:         "cert.pem",
		TLSKeyFile:          "key.pem",
		LetsEncrypt:         false,
		LetsEncryptCacheDir: ".",
		LetsEncryptEmail:    "",
		LetsEncryptHosts:    "",
		APIPrefix:           "/",
		CORSOrigin:          "*",
		ReadTimeout:         30 * time.Second,
		WriteTimeout:        15 * time.Second,
		DB:                  freegeoip.MaxMindDB,
		UpdateInterval:      24 * time.Hour,
		RetryInterval:       2 * time.Hour,
		LogTimestamp:        true,
		RedisAddr:           "localhost:6379",
		RedisTimeout:        time.Second,
		MemcacheAddr:        "localhost:11211",
		MemcacheTimeout:     time.Second,
		RateLimitBackend:    "redis",
		RateLimitInterval:   time.Hour,
		UpdatesHost:         "updates.maxmind.com",
		ProductID:           "GeoIP2-City",
	}
}

// AddFlags adds configuration flags to the given FlagSet.
func (c *Config) AddFlags(fs *flag.FlagSet) {
	defer envconfig.Process("freegeoip", c)
	fs.BoolVar(&c.Naggle, "tcp-naggle", c.Naggle, "Enable TCP Nagle's algorithm (disables NO_DELAY)")
	fs.BoolVar(&c.FastOpen, "tcp-fast-open", c.FastOpen, "Enable TCP fast open")
	fs.StringVar(&c.ServerAddr, "http", c.ServerAddr, "Address in form of ip:port to listen on for HTTP")
	fs.BoolVar(&c.HTTP2, "http2", c.HTTP2, "Enable HTTP/2 when TLS is enabled")
	fs.StringVar(&c.HSTS, "hsts", c.HSTS, "Set HSTS to the value provided on all responses")
	fs.StringVar(&c.TLSServerAddr, "https", c.TLSServerAddr, "Address in form of ip:port to listen on for HTTPS")
	fs.StringVar(&c.TLSCertFile, "cert", c.TLSCertFile, "X.509 certificate file for HTTPS server")
	fs.StringVar(&c.TLSKeyFile, "key", c.TLSKeyFile, "X.509 key file for HTTPS server")
	fs.BoolVar(&c.LetsEncrypt, "letsencrypt", c.LetsEncrypt, "Enable automatic TLS using letsencrypt.org")
	fs.StringVar(&c.LetsEncryptEmail, "letsencrypt-email", c.LetsEncryptEmail, "Optional email to register with letsencrypt (default is anonymous)")
	fs.StringVar(&c.LetsEncryptHosts, "letsencrypt-hosts", c.LetsEncryptHosts, "Comma separated list of hosts for the certificate (required)")
	fs.StringVar(&c.LetsEncryptCacheDir, "letsencrypt-cache-dir", c.LetsEncryptCacheDir, "Letsencrypt cache dir (for storing certs)")
	fs.StringVar(&c.APIPrefix, "api-prefix", c.APIPrefix, "URL prefix for API endpoints")
	fs.StringVar(&c.CORSOrigin, "cors-origin", c.CORSOrigin, "Comma separated list of CORS origin API endpoints")
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
	fs.StringVar(&c.UpdatesHost, "updates-host", c.UpdatesHost, "MaxMind Updates Host")
	fs.StringVar(&c.LicenseKey, "license-key", c.LicenseKey, "MaxMind License Key (requires user-id)")
	fs.StringVar(&c.UserID, "user-id", c.UserID, "MaxMind User ID (requires license-key)")
	fs.StringVar(&c.ProductID, "product-id", c.ProductID, "MaxMind Product ID (e.g GeoIP2-City)")
	fs.StringVar(&c.NewrelicName, "newrelic-name", c.NewrelicName, "Newrepic APM application name")
	fs.StringVar(&c.NewrelicKey, "newrelic-key", c.NewrelicKey, "Nerelic API key")
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
