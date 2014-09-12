// Copyright 2009-2014 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Web server of freegeoip.net

package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fiorix/go-redis/redis"
	"github.com/fiorix/go-web/httpxtra"
	"github.com/gorilla/context"

	// SQLite driver.
	// _ "github.com/mattn/go-sqlite3"
	_ "code.google.com/p/gosqlite/sqlite3"
)

var (
	collectStats  bool
	outputCount   = expvar.NewMap("Output")   // json, xml or csv
	statusCount   = expvar.NewMap("Status")   // 200, 403, 404, etc
	protocolCount = expvar.NewMap("Protocol") // HTTP or HTTPS

	dns *dnsPool
)

func main() {
	flLog := flag.String("l", "", "log to file instead of stderr")
	flConfig := flag.String("c", "freegeoip.conf", "set config file")
	flProfile := flag.Bool("profile", false, "run cpu and mem profiler")
	flag.Parse()

	if *flProfile {
		runProfile()
	}

	cf, err := loadConfig(*flConfig)
	if err != nil {
		log.Fatal(err)
	}

	collectStats = cf.Debug

	if cf.DNS.Enabled {
		t, err := time.ParseDuration(cf.DNS.Timeout)
		if err != nil {
			log.Fatal("Invalid DNS timeout:", err)
		}
		dns = new(dnsPool)
		dns.init(cf.DNS.MaxConcurrent, t)
	}

	if len(*flLog) > 0 {
		setLog(*flLog)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("FreeGeoIP server starting. debug=%t", cf.Debug)

	if cf.Debug && len(cf.DebugSrv) > 0 {
		go func() {
			// server for expvar's /debug/vars only
			log.Printf("Starting DEBUG server on tcp/%s", cf.DebugSrv)
			log.Fatal(http.ListenAndServe(cf.DebugSrv, nil))
		}()
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(cf.DocumentRoot)))

	st := time.Now()
	db, err := openDB(cf)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("IPDB cached in", time.Since(st))

	lh := lookupHandler(cf, db)
	mux.HandleFunc("/csv/", lh)
	mux.HandleFunc("/xml/", lh)
	mux.HandleFunc("/json/", lh)

	for _, c := range cf.Listen {
		go runServer(mux, c)
	}

	select {}
}

func lookupHandler(cf *configFile, db *DB) http.HandlerFunc {
	var rl rateLimiter
	if cf.Limit.MaxRequests > 0 {
		if len(cf.Redis) > 0 {
			rl = new(redisQuota)
			log.Printf("Using redis to manage quota: %s", cf.Redis)
		} else {
			rl = new(mapQuota)
			log.Printf("Using internal map to manage quota.")
		}
		rl.init(cf)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		switch r.Method {
		case "GET":
			w.Header().Set("Access-Control-Allow-Origin", "*")
			handleRequest(cf, db, rl, w, r)
		case "OPTIONS":
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "X-Requested-With")
			w.WriteHeader(200)
		default:
			w.Header().Set("Allow", "GET, OPTIONS")
			http.Error(w, http.StatusText(405), 405)
		}
	}
}

func handleRequest(
	cf *configFile,
	db *DB,
	rl rateLimiter,
	w http.ResponseWriter,
	r *http.Request,
) {
	// If xheaders is enabled, RemoteAddr might be a copy of
	// the X-Real-IP or X-Forwarded-For HTTP headers, which
	// can be a comma separated list of IPs. In this case,
	// only the first IP in the list is used.
	if strings.Index(r.RemoteAddr, ",") > 0 {
		r.RemoteAddr = strings.SplitN(r.RemoteAddr, ",", 2)[0]
	}

	// Parse remote address.
	var ip net.IP
	if sIP, _, err := net.SplitHostPort(r.RemoteAddr); err != nil {
		ip = net.ParseIP(r.RemoteAddr) // Use X-Real-IP, etc
	} else {
		ip = net.ParseIP(sIP)
	}

	if ip == nil {
		// This could be a misconfigured unix socket server.
		context.Set(r, "log", "Invalid source IP: "+r.RemoteAddr)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Convert remote IP to integer to check quota.
	// IPv6 is not supported yet. See issue #21 for details.
	nIP, err := ip2int(ip)
	if err != nil {
		context.Set(r, "log", err.Error())
		http.Error(w, "IPv6 is not supported.", 501)
		return
	}

	// Check quota.
	if rl != nil {
		var ok bool
		if ok, err = rl.Ok(nIP); err != nil {
			context.Set(r, "log", err.Error()) // redis err
			http.Error(w, http.StatusText(503), 503)
			return
		} else if !ok {
			// Over quota, soz :(
			http.Error(w, http.StatusText(403), 403)
			return
		}
	}

	// Figure out the query: /fmt/{query} or /fmt/{nil}
	// In case of {nil} we query the remote IP.
	path := strings.SplitN(r.URL.Path, "/", 3)
	if len(path) != 3 {
		// This handler is for /fmt/ where fmt is json, xml or csv.
		log.Fatal("Unexpected URL:", r.URL.Path)
	}

	// Process the query, if there's one.
	if len(path[2]) > 0 {
		// Allow to query by IP or hostname.
		if ip = net.ParseIP(path[2]); ip == nil {
			if dns == nil || strings.Contains(path[2], " ") {
				// DNS lookups not allowed or invalid hostname.
				http.Error(w, http.StatusText(404), 404)
				return
			} else if ip = dns.LookupHost(path[2]); ip == nil {
				// DNS lookup failed, assume host not found.
				http.Error(w, http.StatusText(404), 404)
				return
			}
		}

		if nIP, err = ip2int(ip); err != nil { // IPv6 fails here.
			context.Set(r, "log", err.Error())
			http.Error(w, http.StatusText(404), 404)
			return
		}
	}

	// Query the db.
	var record *geoipRecord
	if record, err = db.Lookup(ip, nIP); err != nil {
		http.NotFound(w, r)
		return
	}

	// Write response.
	switch path[1][0] {
	case 'j':
		if cb := r.FormValue("callback"); len(cb) > 0 {
			w.Header().Set("Content-Type", "text/javascript")
			err = record.JSONP(w, cb)
		} else {
			w.Header().Set("Content-Type", "application/json")
			err = record.JSON(w)
		}
	case 'x':
		w.Header().Set("Content-Type", "application/xml")
		err = record.XML(w)
	case 'c':
		w.Header().Set("Content-Type", "application/csv")
		err = record.CSV(w)
	}
	if err != nil {
		context.Set(r, "log", err.Error())
	}
}

func runServer(mux *http.ServeMux, c *serverConfig) {
	h := httpxtra.Handler{
		Handler:  mux,
		XHeaders: c.XHeaders,
	}
	if c.Log {
		h.Logger = httpLogger
	}
	s := http.Server{
		Addr:         c.Addr,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	if len(c.KeyFile) > 0 && len(c.CertFile) > 0 {
		log.Printf("Starting HTTPS server on tcp/%s "+
			"log=%t xheaders=%t cert=%s key=%s",
			c.Addr,
			c.Log,
			c.XHeaders,
			c.CertFile,
			c.KeyFile,
		)
		log.Fatal(s.ListenAndServeTLS(c.CertFile, c.KeyFile))
	} else {
		log.Printf("Starting HTTP server on tcp/%s "+
			"log=%t xheaders=%t",
			c.Addr,
			c.Log,
			c.XHeaders,
		)
		log.Fatal(httpxtra.ListenAndServe(s))
	}
}

type dnsPool struct {
	queryChan    chan *dnsQuery
	queryTimeout time.Duration
}

type dnsQuery struct {
	hostname string
	respChan chan net.IP
}

func (p *dnsPool) init(size int, queryTimeout time.Duration) {
	p.queryChan = make(chan *dnsQuery)
	p.queryTimeout = queryTimeout
	for n := 0; n < size; n++ {
		go p.doWork()
	}
}

func (p *dnsPool) doWork() {
	for q := range p.queryChan {
		q.respChan <- p.resolve(q.hostname)
	}
}

func (p *dnsPool) resolve(hostname string) net.IP {
	if a, err := net.LookupHost(hostname); err != nil {
		return nil
	} else if len(a) == 1 {
		return net.ParseIP(a[0])
	} else {
		return net.ParseIP(a[rand.Intn(len(a)-1)])
	}
}

func (p *dnsPool) LookupHost(hostname string) net.IP {
	q := &dnsQuery{hostname, make(chan net.IP, 1)}
	select {
	case p.queryChan <- q:
		select {
		case ip := <-q.respChan:
			close(q.respChan)
			return ip
		case <-time.After(p.queryTimeout):
			return nil
		}
	default:
		return nil
	}
}

type DB struct {
	db   *sql.DB
	stmt *sql.Stmt

	// cache
	country map[string]string
	region  map[regionKey]string
	city    map[int]locationData
}

type regionKey struct {
	CountryCode,
	RegionCode string
}

type locationData struct {
	CountryCode,
	RegionCode,
	CityName,
	ZipCode string
	Latitude,
	Longitude float32
	MetroCode,
	AreaCode string
}

func openDB(cf *configFile) (*DB, error) {
	var (
		db  DB
		err error
	)
	if db.db, err = sql.Open("sqlite3", cf.IPDB.File); err != nil {
		return nil, err
	}
	if _, err = db.db.Exec("PRAGMA cache_size=" + cf.IPDB.CacheSize); err != nil {
		return nil, err
	}
	if db.stmt, err = db.db.Prepare(`
		SELECT
			loc_id
		FROM
			city_blocks
		WHERE
			ip_start <= ?
		ORDER BY
			ip_start DESC
		LIMIT 1
	`); err != nil {
		return nil, err
	}
	if err = db.loadCache(); err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *DB) loadCache() error {
	var (
		wg   sync.WaitGroup
		errc = make(chan error, 3)
	)
	wg.Add(3)
	go db.loadCountries(&wg, errc)
	go db.loadRegions(&wg, errc)
	go db.loadCities(&wg, errc)
	wg.Wait()
	select {
	case err := <-errc:
		return err
	default:
		return nil
	}
}

func (db *DB) loadCountries(wg *sync.WaitGroup, errc chan error) {
	defer wg.Done()
	db.country = make(map[string]string)
	row, err := db.db.Query(`
		SELECT
			country_code,
			country_name
		FROM
			country_blocks
	`)
	if err != nil {
		errc <- err
		return
	}
	defer row.Close()
	var country_code, name string
	for row.Next() {
		if err = row.Scan(
			&country_code,
			&name,
		); err != nil {
			errc <- err
			return
		}
		db.country[country_code] = name
	}
}

func (db *DB) loadRegions(wg *sync.WaitGroup, errc chan error) {
	defer wg.Done()
	db.region = make(map[regionKey]string)
	row, err := db.db.Query(`
		SELECT
			country_code,
			region_code,
			region_name
		FROM
			region_names
	`)
	if err != nil {
		errc <- err
		return
	}
	defer row.Close()
	var country_code, region_code, name string
	for row.Next() {
		if err = row.Scan(
			&country_code,
			&region_code,
			&name,
		); err != nil {
			errc <- err
			return
		}
		db.region[regionKey{country_code, region_code}] = name
	}
}

func (db *DB) loadCities(wg *sync.WaitGroup, errc chan error) {
	defer wg.Done()
	db.city = make(map[int]locationData)
	row, err := db.db.Query("SELECT * FROM city_location")
	if err != nil {
		errc <- err
		return
	}
	defer row.Close()
	var (
		locId int
		loc   locationData
	)
	for row.Next() {
		if err = row.Scan(
			&locId,
			&loc.CountryCode,
			&loc.RegionCode,
			&loc.CityName,
			&loc.ZipCode,
			&loc.Latitude,
			&loc.Longitude,
			&loc.MetroCode,
			&loc.AreaCode,
		); err != nil {
			errc <- err
			return
		}
		db.city[locId] = loc
	}
}

func (db *DB) Lookup(ip net.IP, nIP uint32) (*geoipRecord, error) {
	for _, net := range reservedIPs {
		if net.Contains(ip) {
			return &geoipRecord{
				Ip:          ip.String(),
				CountryCode: "RD",
				CountryName: "Reserved",
			}, nil
		}
	}
	var locId int
	if err := db.stmt.QueryRow(nIP).Scan(&locId); err != nil {
		return nil, err
	}
	return db.newRecord(&ip, locId), nil
}

func (db *DB) newRecord(ip *net.IP, locId int) *geoipRecord {
	city, ok := db.city[locId]
	if !ok {
		return &geoipRecord{Ip: ip.String()}
	}
	return &geoipRecord{
		Ip:          ip.String(),
		CountryCode: city.CountryCode,
		CountryName: db.country[city.CountryCode],
		RegionCode:  city.RegionCode,
		RegionName: db.region[regionKey{
			city.CountryCode,
			city.RegionCode,
		}],
		CityName:  city.CityName,
		ZipCode:   city.ZipCode,
		Latitude:  city.Latitude,
		Longitude: city.Longitude,
		MetroCode: city.MetroCode,
		AreaCode:  city.AreaCode,
	}
}

type geoipRecord struct {
	XMLName     xml.Name `json:"-" xml:"Response"`
	Ip          string   `json:"ip"`
	CountryCode string   `json:"country_code"`
	CountryName string   `json:"country_name"`
	RegionCode  string   `json:"region_code"`
	RegionName  string   `json:"region_name"`
	CityName    string   `json:"city" xml:"City"`
	ZipCode     string   `json:"zipcode"`
	Latitude    float32  `json:"latitude"`
	Longitude   float32  `json:"longitude"`
	MetroCode   string   `json:"metro_code"`
	AreaCode    string   `json:"area_code"`
}

func (r *geoipRecord) JSON(w io.Writer) error {
	if collectStats {
		outputCount.Add("json", 1)
	}
	return json.NewEncoder(w).Encode(r)
}

func (r *geoipRecord) JSONP(w io.Writer, callback string) error {
	if collectStats {
		outputCount.Add("json", 1)
	}
	var err error
	if _, err = w.Write([]byte(callback)); err != nil {
		return err
	}
	if _, err = w.Write([]byte("(")); err != nil {
		return err
	}
	if err = json.NewEncoder(w).Encode(r); err != nil {
		return err
	}
	if _, err = w.Write([]byte(");")); err != nil {
		return err
	}
	return nil
}

func (r *geoipRecord) XML(w io.Writer) error {
	if collectStats {
		outputCount.Add("xml", 1)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")
	var err error
	if _, err = w.Write([]byte(xml.Header)); err != nil {
		return err
	}
	if err = enc.Encode(r); err != nil {
		return err
	}
	if _, err = w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func (r *geoipRecord) CSV(w io.Writer) error {
	if collectStats {
		outputCount.Add("csv", 1)
	}
	if _, err := fmt.Fprintf(w,
		`"%s","%s","%s","%s","%s","%s","%s","%0.4f","%0.4f","%s","%s"`+"\r\n",
		r.Ip,
		r.CountryCode,
		r.CountryName,
		r.RegionCode,
		r.RegionName,
		r.CityName,
		r.ZipCode,
		r.Latitude,
		r.Longitude,
		r.MetroCode,
		r.AreaCode,
	); err != nil {
		return err
	}
	return nil
}

type rateLimiter interface {
	init(cf *configFile)           // Initializes the limiter
	Ok(ipkey uint32) (bool, error) // Returns true if under quota
}

// mapQuota implements the rateLimiter interface using a map as the backend.
type mapQuota struct {
	cf *configFile
	mu sync.Mutex
	ip map[uint32]int
}

func (q *mapQuota) init(cf *configFile) {
	q.cf = cf
	q.ip = make(map[uint32]int)
}

func (q *mapQuota) Ok(ipkey uint32) (bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if n, ok := q.ip[ipkey]; ok {
		if n < q.cf.Limit.MaxRequests {
			q.ip[ipkey]++
			return true, nil
		}
		return false, nil
	}
	q.ip[ipkey] = 1
	go func() {
		time.Sleep(time.Duration(q.cf.Limit.Expire) * time.Second)
		q.mu.Lock()
		delete(q.ip, ipkey)
		q.mu.Unlock()
	}()
	return true, nil
}

// redisQuota implements the rateLimiter interface using Redis as the backend.
type redisQuota struct {
	cf *configFile
	rc *redis.Client
}

func (q *redisQuota) init(cf *configFile) {
	redis.MaxIdleConnsPerAddr = 5000
	q.cf = cf
	q.rc = redis.New(cf.Redis...)
	q.rc.Timeout = time.Duration(1500) * time.Millisecond
}

func (q *redisQuota) Ok(ipkey uint32) (bool, error) {
	k := strconv.Itoa(int(ipkey)) // "numeric" key
	if ns, err := q.rc.Get(k); err != nil {
		return false, fmt.Errorf("redis get: %s", err)
	} else if len(ns) == 0 {
		if err = q.rc.SetEx(k, q.cf.Limit.Expire, "1"); err != nil {
			return false, fmt.Errorf("redis setex: %s", err)
		}
	} else if n, _ := strconv.Atoi(ns); n < q.cf.Limit.MaxRequests {
		if n, err = q.rc.Incr(k); err != nil {
			return false, fmt.Errorf("redis incr: %s", err)
		} else if n == 1 {
			q.rc.Expire(k, q.cf.Limit.Expire)
		}
	} else {
		return false, nil
	}
	return true, nil
}

func ip2int(ip net.IP) (uint32, error) {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0, fmt.Errorf("IP %s is not IPv4", ip.String())
	}
	return binary.BigEndian.Uint32(ipv4), nil
}

func runProfile() {
	f, err := os.Create("freegeoip.cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	go func() {
		<-sig
		pprof.StopCPUProfile()
		f.Close()
		f, err = os.Create("freegeoip.mem.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		os.Exit(0)
	}()
}

func setLog(filename string) {
	f := openLog(filename)
	log.SetOutput(f)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP)
	go func() {
		// Recycle log file on SIGHUP.
		var fb *os.File
		for {
			<-sigc
			fb = f
			f = openLog(filename)
			log.SetOutput(f)
			fb.Close()
		}
	}()
}

func openLog(filename string) *os.File {
	f, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
	return f
}

func httpLogger(r *http.Request, created time.Time, status, bytes int) {
	//fmt.Println(httpxtra.ApacheCommonLog(r, created, status, bytes))
	var (
		s, ip, msg string
		err        error
	)
	if r.TLS == nil {
		s = "HTTP"
	} else {
		s = "HTTPS"
	}
	if ip, _, err = net.SplitHostPort(r.RemoteAddr); err != nil {
		ip = r.RemoteAddr
	}
	if tmp := context.Get(r, "log"); tmp != nil {
		msg = fmt.Sprintf(" (%s)", tmp)
		context.Clear(r)
	}
	log.Printf("%s %d %s %q (%s) :: %d bytes in %s%s",
		s,
		status,
		r.Method,
		r.URL.Path,
		ip,
		bytes,
		time.Since(created),
		msg,
	)
	if collectStats {
		protocolCount.Add(s, 1)
		statusCount.Add(strconv.Itoa(status), 1)
	}
}

type serverConfig struct {
	Log      bool   `xml:"log,attr"`
	XHeaders bool   `xml:"xheaders,attr"`
	Addr     string `xml:"addr,attr"`
	CertFile string
	KeyFile  string
}

type configFile struct {
	XMLName      xml.Name `xml:"Server"`
	Debug        bool     `xml:"debug,attr"`
	DebugSrv     string   `xml:"debugsrv,attr"`
	DocumentRoot string

	Listen []*serverConfig

	DNS struct {
		Enabled       bool   `xml:",attr"`
		Timeout       string `xml:",attr"`
		MaxConcurrent int    `xml:",attr"`
	}

	IPDB struct {
		File      string `xml:",attr"`
		CacheSize string `xml:",attr"`
	}

	Limit struct {
		MaxRequests int `xml:",attr"`
		Expire      int `xml:",attr"`
	}

	Redis []string `xml:"Redis>Addr"`
}

func loadConfig(filename string) (*configFile, error) {
	var cf configFile
	if fd, err := os.Open(filename); err != nil {
		return nil, err
	} else {
		if err = xml.NewDecoder(fd).Decode(&cf); err != nil {
			return nil, err
		}
		fd.Close()
	}
	// Make files' path relative to the config file's directory.
	basedir := filepath.Dir(filename)
	relativePath(basedir, &cf.IPDB.File)
	relativePath(basedir, &cf.DocumentRoot)
	for _, l := range cf.Listen {
		relativePath(basedir, &l.CertFile)
		relativePath(basedir, &l.KeyFile)
	}
	return &cf, nil
}

func relativePath(basedir string, filename *string) {
	if len(*filename) > 0 && (*filename)[0] != '/' {
		*filename = filepath.Join(basedir, *filename)
	}
}

// http://en.wikipedia.org/wiki/Reserved_IP_addresses
var reservedIPs = []net.IPNet{
	{net.IPv4(0, 0, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
	{net.IPv4(10, 0, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
	{net.IPv4(100, 64, 0, 0), net.IPv4Mask(255, 192, 0, 0)},
	{net.IPv4(127, 0, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
	{net.IPv4(169, 254, 0, 0), net.IPv4Mask(255, 255, 0, 0)},
	{net.IPv4(172, 16, 0, 0), net.IPv4Mask(255, 240, 0, 0)},
	{net.IPv4(192, 0, 0, 0), net.IPv4Mask(255, 255, 255, 248)},
	{net.IPv4(192, 0, 2, 0), net.IPv4Mask(255, 255, 255, 0)},
	{net.IPv4(192, 88, 99, 0), net.IPv4Mask(255, 255, 255, 0)},
	{net.IPv4(192, 168, 0, 0), net.IPv4Mask(255, 255, 0, 0)},
	{net.IPv4(198, 18, 0, 0), net.IPv4Mask(255, 254, 0, 0)},
	{net.IPv4(198, 51, 100, 0), net.IPv4Mask(255, 255, 255, 0)},
	{net.IPv4(203, 0, 113, 0), net.IPv4Mask(255, 255, 255, 0)},
	{net.IPv4(224, 0, 0, 0), net.IPv4Mask(240, 0, 0, 0)},
	{net.IPv4(240, 0, 0, 0), net.IPv4Mask(240, 0, 0, 0)},
	{net.IPv4(255, 255, 255, 255), net.IPv4Mask(255, 255, 255, 255)},
}
