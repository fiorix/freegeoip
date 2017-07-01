// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fiorix/freegeoip"
)

// Config is the configuration of the freegeoip server.
type Config struct {
	FastOpen            bool   // TCP Fast Open
	Naggle              bool   // TCP Naggle (buffered, disables TCP_NODELAY)
	ServerAddr          string // HTTP server addr
	TLSServerAddr       string // HTTPS server addr
	TLSCertFile         string
	TLSKeyFile          string
	LetsEncrypt         bool
	LetsEncryptCacheDir string
	LetsEncryptEmail    string
	LetsEncryptHosts    string
	APIPrefix           string
	CORSOrigin          string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	PublicDir           string
	DB                  string
	UpdateInterval      time.Duration
	RetryInterval       time.Duration
	UseXForwardedFor    bool
	Silent              bool
	LogToStdout         bool
	LogTimestamp        bool
	RedisAddr           string
	RedisTimeout        time.Duration
	MemcacheAddr        string
	MemcacheTimeout     time.Duration
	RateLimitBackend    string
	RateLimitLimit      uint64
	RateLimitInterval   time.Duration
	InternalServerAddr  string
	UpdatesHost         string
	LicenseKey          string
	UserID              string
	ProductID           string
	NewrelicName        string
	NewrelicKey         string

	errorLog  *log.Logger
	accessLog *log.Logger
}

// NewConfig creates and initializes a new Config with default values.
func NewConfig() *Config {
	return &Config{
		FastOpen:            false,
		Naggle:              false,
		ServerAddr:          ":8080",
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
	fs.BoolVar(&c.Naggle, "tcp-naggle", c.Naggle, "Enable TCP Nagle's algorithm (disables NO_DELAY)")
	fs.BoolVar(&c.FastOpen, "tcp-fast-open", c.FastOpen, "Enable TCP fast open")
	fs.StringVar(&c.ServerAddr, "http", c.ServerAddr, "Address in form of ip:port to listen on for HTTP")
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

	err := setFlagsFromEnv("FREEGEOIP", fs)
	if err != nil {
    //c.errorLogger(err)
	}
}

// SetFlagsFromEnv parses all registered flags in the given flagset,
// and if they are not already set it attempts to set their values from
// environment variables. Environment variables take the name of the flag but
// are UPPERCASE, have the given prefix  and any dashes are replaced by
// underscores - for example: some-flag => FREEGEOIP_SOME_FLAG
func setFlagsFromEnv(prefix string, fs *flag.FlagSet) error {
	var err error
	alreadySet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		alreadySet[flagToEnv(prefix, f.Name)] = true
	})
	usedEnvKey := make(map[string]bool)
	fs.VisitAll(func(f *flag.Flag) {
		err = setFlagFromEnv(fs, prefix, f.Name, usedEnvKey, alreadySet, true)
	})

	verifyEnv(prefix, usedEnvKey, alreadySet)

	return err
}

func verifyEnv(prefix string, usedEnvKey, alreadySet map[string]bool) {
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 {
			//plog.Warningf("found invalid env %s", env)
		}
		if usedEnvKey[kv[0]] {
			continue
		}
		if alreadySet[kv[0]] {
			//plog.Infof("recognized environment variable %s, but unused: shadowed by corresponding flag ", kv[0])
			continue
		}
		if strings.HasPrefix(env, prefix+"_") {
			//plog.Warningf("unrecognized environment variable %s", env)
		}
	}
}

// FlagToEnv converts flag string to upper-case environment variable key string.
func flagToEnv(prefix, name string) string {
	return prefix + "_" + strings.ToUpper(strings.Replace(name, "-", "_", -1))
}

type flagSetter interface {
	Set(fk string, fv string) error
}

func setFlagFromEnv(fs flagSetter, prefix, fname string, usedEnvKey, alreadySet map[string]bool, log bool) error {
	key := flagToEnv(prefix, fname)
	if !alreadySet[key] {
		val := os.Getenv(key)
		if val != "" {
			usedEnvKey[key] = true
			if serr := fs.Set(fname, val); serr != nil {
				//return fmt.Errorf("invalid value %q for %s: %v", val, key, serr)
			}
			if log {
				//plog.Infof("recognized and used environment variable %s=%s", key, val)
			}
		}
	}
	return nil
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
