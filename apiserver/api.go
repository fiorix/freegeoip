// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package apiserver

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/fiorix/go-redis/redis"
	"github.com/go-web/httplog"
	"github.com/go-web/httpmux"
	"github.com/go-web/httprl"
	"github.com/go-web/httprl/memcacherl"
	"github.com/go-web/httprl/redisrl"
	newrelic "github.com/newrelic/go-agent"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
	"golang.org/x/text/language"

	"github.com/apilayer/freegeoip"
)

type apiHandler struct {
	db    *freegeoip.DB
	conf  *Config
	cors  *cors.Cors
	nrapp newrelic.Application
}

// NewHandler creates an http handler for the freegeoip server that
// can be embedded in other servers.
func NewHandler(c *Config) (http.Handler, error) {
	db, err := openDB(c)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	cf := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(c.CORSOrigin, ","),
		AllowedMethods:   []string{"GET"},
		AllowCredentials: true,
	})
	f := &apiHandler{db: db, conf: c, cors: cf}
	mc := httpmux.DefaultConfig
	if err := f.config(&mc); err != nil {
		return nil, err
	}
	mux := httpmux.NewHandler(&mc)
	mux.GET("/csv/*host", f.register("csv", csvWriter))
	mux.GET("/xml/*host", f.register("xml", xmlWriter))
	mux.GET("/json/*host", f.register("json", jsonWriter))
	go watchEvents(db)
	return mux, nil
}

func (f *apiHandler) config(mc *httpmux.Config) error {
	mc.Prefix = f.conf.APIPrefix
	mc.NotFound = newPublicDirHandler(f.conf.PublicDir)
	if f.conf.UseXForwardedFor {
		mc.UseFunc(httplog.UseXForwardedFor)
	}
	if !f.conf.Silent {
		mc.UseFunc(httplog.ApacheCombinedFormat(f.conf.accessLogger()))
	}
	if f.conf.HSTS != "" {
		mc.UseFunc(hstsMiddleware(f.conf.HSTS))
	}
	mc.UseFunc(clientMetricsMiddleware(f.db))
	if f.conf.RateLimitLimit > 0 {
		rl, err := newRateLimiter(f.conf)
		if err != nil {
			return fmt.Errorf("failed to create rate limiter: %v", err)
		}
		mc.Use(rl.Handle)
	}
	if f.conf.NewrelicName != "" && f.conf.NewrelicKey != "" {
		config := newrelic.NewConfig(f.conf.NewrelicName, f.conf.NewrelicKey)
		app, err := newrelic.NewApplication(config)
		if err != nil {
			return fmt.Errorf("failed to create newrelic application: {name: %v, key: %v}", f.conf.NewrelicName, f.conf.NewrelicKey)
		}
		f.nrapp = app
	}
	return nil
}

func newPublicDirHandler(path string) http.HandlerFunc {
	handler := http.NotFoundHandler()
	if path != "" {
		handler = http.FileServer(http.Dir(path))
	}
	return prometheus.InstrumentHandler("frontend", handler)
}

func hstsMiddleware(policy string) httpmux.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil {
				return
			}
			w.Header().Set("Strict-Transport-Security", policy)
			next(w, r)
		}
	}
}

func clientMetricsMiddleware(db *freegeoip.DB) httpmux.MiddlewareFunc {
	type query struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
			// Collect metrics after serving the request.
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				return
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return
			}
			if ip.To4() != nil {
				clientIPProtoCounter.WithLabelValues("4").Inc()
			} else {
				clientIPProtoCounter.WithLabelValues("6").Inc()
			}
			var q query
			err = db.Lookup(ip, &q)
			if err != nil || q.Country.ISOCode == "" {
				clientCountryCounter.WithLabelValues("unknown").Inc()
				return
			}
			clientCountryCounter.WithLabelValues(q.Country.ISOCode).Inc()
		}
	}
}

type writerFunc func(w http.ResponseWriter, r *http.Request, d *responseRecord)

func (f *apiHandler) register(name string, writer writerFunc) http.HandlerFunc {
	var h http.Handler
	if f.nrapp == nil {
		h = prometheus.InstrumentHandler(name, f.iplookup(writer))
	} else {
		h = prometheus.InstrumentHandler(newrelic.WrapHandle(f.nrapp, name, f.iplookup(writer)))
	}

	return f.cors.Handler(h).ServeHTTP
}

func (f *apiHandler) iplookup(writer writerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := httpmux.Params(r).ByName("host")
		if len(host) > 0 && host[0] == '/' {
			host = host[1:]
		}
		if host == "" {
			host, _, _ = net.SplitHostPort(r.RemoteAddr)
			if host == "" {
				host = r.RemoteAddr
			}
		}
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			http.NotFound(w, r)
			return
		}
		ip, q := ips[rand.Intn(len(ips))], &geoipQuery{}
		err = f.db.Lookup(ip, &q.DefaultQuery)
		if err != nil {
			http.Error(w, "Try again later.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("X-Database-Date", f.db.Date().Format(http.TimeFormat))
		resp := q.Record(ip, r.Header.Get("Accept-Language"))
		writer(w, r, resp)
	}
}

func csvWriter(w http.ResponseWriter, r *http.Request, d *responseRecord) {
	w.Header().Set("Content-Type", "text/csv")
	io.WriteString(w, d.String())
}

func xmlWriter(w http.ResponseWriter, r *http.Request, d *responseRecord) {
	w.Header().Set("Content-Type", "application/xml")
	x := xml.NewEncoder(w)
	x.Indent("", "\t")
	x.Encode(d)
	w.Write([]byte{'\n'})
}

func jsonWriter(w http.ResponseWriter, r *http.Request, d *responseRecord) {
	if cb := r.FormValue("callback"); cb != "" {
		w.Header().Set("Content-Type", "application/javascript")
		io.WriteString(w, cb)
		w.Write([]byte("("))
		b, err := json.Marshal(d)
		if err == nil {
			w.Write(b)
		}
		io.WriteString(w, ");")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

type geoipQuery struct {
	freegeoip.DefaultQuery
}

func (q *geoipQuery) Record(ip net.IP, lang string) *responseRecord {
	lang = parseAcceptLanguage(lang, q.Country.Names)

	r := &responseRecord{
		IP:          ip.String(),
		CountryCode: q.Country.ISOCode,
		CountryName: q.Country.Names[lang],
		City:        q.City.Names[lang],
		ZipCode:     q.Postal.Code,
		TimeZone:    q.Location.TimeZone,
		Latitude:    roundFloat(q.Location.Latitude, .5, 4),
		Longitude:   roundFloat(q.Location.Longitude, .5, 4),
		MetroCode:   q.Location.MetroCode,
	}
	if len(q.Region) > 0 {
		r.RegionCode = q.Region[0].ISOCode
		r.RegionName = q.Region[0].Names[lang]
	}
	return r
}

func parseAcceptLanguage(header string, dbLangs map[string]string) string {
	// supported languages -- i.e. languages available in the DB
	matchLangs := []language.Tag{
		language.English,
	}

	// parse available DB languages and add to supported
	for name := range dbLangs {
		matchLangs = append(matchLangs, language.Raw.Make(name))
	}

	var matcher = language.NewMatcher(matchLangs)

	// parse header
	t, _, _ := language.ParseAcceptLanguage(header)
	// match most acceptable language
	tag, _, _ := matcher.Match(t...)
	// extract base language
	base, _ := tag.Base()

	return base.String()
}

func roundFloat(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	return round / pow
}

type responseRecord struct {
	XMLName     xml.Name `xml:"Response" json:"-"`
	IP          string   `json:"ip"`
	CountryCode string   `json:"country_code"`
	CountryName string   `json:"country_name"`
	RegionCode  string   `json:"region_code"`
	RegionName  string   `json:"region_name"`
	City        string   `json:"city"`
	ZipCode     string   `json:"zip_code"`
	TimeZone    string   `json:"time_zone"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	MetroCode   uint     `json:"metro_code"`
}

func (rr *responseRecord) String() string {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	w.UseCRLF = true
	w.Write([]string{
		rr.IP,
		rr.CountryCode,
		rr.CountryName,
		rr.RegionCode,
		rr.RegionName,
		rr.City,
		rr.ZipCode,
		rr.TimeZone,
		strconv.FormatFloat(rr.Latitude, 'f', 4, 64),
		strconv.FormatFloat(rr.Longitude, 'f', 4, 64),
		strconv.Itoa(int(rr.MetroCode)),
	})
	w.Flush()
	return b.String()
}

// openDB opens and returns the IP database file or URL.
func openDB(c *Config) (*freegeoip.DB, error) {
	// This is a paid product. Get the updates URL.
	if len(c.UserID) > 0 && len(c.LicenseKey) > 0 {
		var err error
		c.DB, err = freegeoip.MaxMindUpdateURL(
			c.UpdatesHost,
			c.ProductID,
			c.UserID,
			c.LicenseKey,
		)
		if err != nil {
			return nil, err
		}
		log.Println("Using updates URL:", c.DB)
	}

	u, err := url.Parse(c.DB)
	if err != nil || len(u.Scheme) == 0 {
		return freegeoip.Open(c.DB)
	}
	return freegeoip.OpenURL(c.DB, c.UpdateInterval, c.RetryInterval)
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
		case msg := <-db.NotifyInfo():
			log.Println("database info:", msg)
		case <-db.NotifyClose():
			return
		}
	}
}

func newRateLimiter(c *Config) (*httprl.RateLimiter, error) {
	var backend httprl.Backend
	switch c.RateLimitBackend {
	case "map":
		m := httprl.NewMap(1)
		m.Start()
		backend = m
	case "redis":
		addrs := strings.Split(c.RedisAddr, ",")
		rc, err := redis.NewClient(addrs...)
		if err != nil {
			return nil, err
		}
		rc.SetTimeout(c.RedisTimeout)
		backend = redisrl.New(rc)
	case "memcache":
		addrs := strings.Split(c.MemcacheAddr, ",")
		mc := memcache.New(addrs...)
		mc.Timeout = c.MemcacheTimeout
		backend = memcacherl.New(mc)
	default:
		return nil, fmt.Errorf("unsupported backend: %q" + c.RateLimitBackend)
	}
	rl := &httprl.RateLimiter{
		Backend:  backend,
		Limit:    c.RateLimitLimit,
		Interval: int32(c.RateLimitInterval.Seconds()),
		ErrorLog: c.errorLogger(),
		//Policy:   httprl.AllowPolicy,
	}
	return rl, nil
}
