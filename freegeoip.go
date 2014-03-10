// Copyright 2013-2014 Alexandre Fiori
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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/fiorix/go-web/httpxtra"

	_ "github.com/mattn/go-sqlite3"
	//_ "code.google.com/p/gosqlite/sqlite3"
)

var (
	conf        *ConfigFile
	protoCount  = expvar.NewMap("Protocol") // HTTP or HTTPS
	outputCount = expvar.NewMap("Output")   // json, xml, csv or other
	statusCount = expvar.NewMap("Status")   // 200, 403, 404, etc
)

func main() {
	cf := flag.String("config", "freegeoip.conf", "set config file")
	prof := flag.Bool("profile", false, "run cpu and mem profiling")
	flag.Parse()

	if buf, err := ioutil.ReadFile(*cf); err != nil {
		log.Fatal(err)
	} else {
		conf = &ConfigFile{}
		if err := xml.Unmarshal(buf, conf); err != nil {
			log.Fatal(err)
		}
	}

	if *prof {
		profile()
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("FreeGeoIP server starting. debug=%t", conf.Debug)

	if conf.Debug && len(conf.DebugSrv) > 0 {
		go func() {
			// server for expvar's /debug/vars only
			log.Printf("Starting DEBUG HTTP server on %s", conf.DebugSrv)
			log.Fatal(http.ListenAndServe(conf.DebugSrv, nil))
		}()
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(conf.DocumentRoot)))

	h := LookupHandler()
	mux.HandleFunc("/csv/", h)
	mux.HandleFunc("/xml/", h)
	mux.HandleFunc("/json/", h)

	wg := new(sync.WaitGroup)
	for _, l := range conf.Listen {
		if l.Addr == "" {
			continue
		}
		wg.Add(1)
		h := httpxtra.Handler{Handler: mux, XHeaders: l.XHeaders}
		if l.Log {
			h.Logger = logger
		}

		s := http.Server{
			Addr:         l.Addr,
			Handler:      h,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		}

		if l.KeyFile == "" && l.CertFile == "" {
			log.Printf("Starting HTTP server on %s "+
				"log=%t xheaders=%t",
				l.Addr, l.Log, l.XHeaders)
			go func() {
				log.Fatal(httpxtra.ListenAndServe(s))
			}()
		} else {
			log.Printf("Starting HTTPS server on %s "+
				"log=%t xheaders=%t cert=%s key=%s",
				l.Addr, l.Log, l.XHeaders,
				l.CertFile, l.KeyFile)
			go func() {
				log.Fatal(s.ListenAndServeTLS(
					l.CertFile,
					l.KeyFile,
				))
			}()
		}
	}

	wg.Wait()
}

// LookupHandler handles GET on /csv, /xml and /json.
func LookupHandler() http.HandlerFunc {
	db, err := sql.Open("sqlite3", conf.IPDB.File)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("PRAGMA cache_size=" + conf.IPDB.CacheSize)
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := db.Prepare(ipdb_query)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Caching database, please wait...")
	cache := NewCache(db)

	//defer stmt.Close()

	var quota Quota
	if len(conf.Redis) == 0 {
		quota = new(MapQuota)
		quota.Setup()
		log.Printf("Using internal map to manage quota.")
	} else {
		quota = new(RedisQuota)
		quota.Setup(conf.Redis...)
		log.Printf("Using redis to manage quota: %s", conf.Redis)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case "OPTIONS":
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "X-Requested-With")
			w.WriteHeader(200)
			return
		default:
			w.Header().Set("Allow", "GET, OPTIONS")
			http.Error(w, http.StatusText(405), 405)
			return
		}

		// GET continues...
		var srcIP net.IP

		// If xheaders is enabled, RemoteAddr might be a copy of
		// the X-Real-IP or X-Forwarded-For HTTP headers, which
		// can be a comma separated list of IPs. In this case,
		// only the first IP in the list is used.
		r.RemoteAddr = strings.SplitN(r.RemoteAddr, ",", 2)[0]

		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err != nil {
			srcIP = net.ParseIP(r.RemoteAddr) // Use X-Real-IP
		} else {
			srcIP = net.ParseIP(ip)
		}

		if srcIP == nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}

		nsrcIP, err := ip2int(srcIP)
		if err != nil {
			if conf.Debug {
				log.Println(err)
			}
			http.Error(w, http.StatusText(400), 400)
			return
		}

		// Check quota.
		if conf.Limit.MaxRequests > 0 {
			var ok bool
			if ok, err = quota.Ok(nsrcIP); err != nil {
				if conf.Debug {
					log.Println(err) // redis error
				}
				http.Error(w, http.StatusText(503), 503)
				return
			} else if !ok {
				// Over quota, soz :(
				http.Error(w, http.StatusText(403), 403)
				return
			}
		}

		var (
			queryIP  net.IP
			nqueryIP uint32
		)

		// Parse URL (e.g. /csv/ip, /xml/)
		a := strings.SplitN(r.URL.Path, "/", 3)
		if len(a) == 3 && a[2] != "" {
			addrs, err := net.LookupHost(a[2])
			if err != nil {
				// DNS lookup failed, assume host not found.
				http.Error(w, http.StatusText(404), 404)
				return
			}

			if queryIP = net.ParseIP(addrs[0]); queryIP == nil {
				http.Error(w, http.StatusText(400), 400)
				return
			}

			nqueryIP, err = ip2int(net.ParseIP(addrs[0]))
			if err != nil {
				if conf.Debug {
					log.Println(err)
				}
				http.Error(w, http.StatusText(400), 400)
				return
			}

		} else {
			queryIP = srcIP
			nqueryIP = nsrcIP
		}

		// Query the db.
		geoip, err := ipdb_lookup(stmt, cache, queryIP, nqueryIP)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		switch a[1][0] {
		case 'j': // json
			enc := json.NewEncoder(w)
			if cb := r.FormValue("callback"); cb != "" {
				w.Header().Set("Content-Type", "text/javascript")
				fmt.Fprintf(w, "%s(", cb)
				enc.Encode(geoip)
				fmt.Fprintf(w, ");")
			} else {
				w.Header().Set("Content-Type", "application/json")
				enc.Encode(geoip)
			}
		case 'x': // xml
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, xml.Header)
			enc := xml.NewEncoder(w)
			enc.Indent("", " ")
			enc.Encode(geoip)
			fmt.Fprintf(w, "\n")
		case 'c': // csv
			w.Header().Set("Content-Type", "application/csv")
			fmt.Fprintf(w, `"%s","%s","%s","%s","%s","%s",`+
				`"%s","%0.4f","%0.4f","%s","%s"`+"\r\n",
				geoip.Ip,
				geoip.CountryCode, geoip.CountryName,
				geoip.RegionCode, geoip.RegionName,
				geoip.CityName, geoip.ZipCode,
				geoip.Latitude, geoip.Longitude,
				geoip.MetroCode, geoip.AreaCode)
		}
	}
}

func ip2int(ip net.IP) (uint32, error) {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0, fmt.Errorf("IP %s is not IPv4", ip.String())
	}

	return binary.BigEndian.Uint32(ipv4), nil
}

func profile() {
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

func logger(r *http.Request, created time.Time, status, bytes int) {
	//fmt.Println(httpxtra.ApacheCommonLog(r, created, status, bytes))
	var (
		s, ip string
		err   error
	)
	if r.TLS == nil {
		s = "HTTP"
	} else {
		s = "HTTPS"
	}
	if ip, _, err = net.SplitHostPort(r.RemoteAddr); err != nil {
		ip = r.RemoteAddr
	}
	log.Printf("%s %d %s %q (%s) :: %s",
		s,
		status,
		r.Method,
		r.URL.Path,
		ip,
		time.Since(created),
	)
	if conf.Debug {
		protoCount.Add(s, 1)
		statusCount.Add(fmt.Sprintf("%d", status), 1)
		switch strings.SplitN(r.URL.Path, "/", 2)[1] {
		case "json/":
			outputCount.Add("json", 1)
		case "xml/":
			outputCount.Add("xml", 1)
		case "csv/":
			outputCount.Add("csv", 1)
		default:
			outputCount.Add("other", 1)
		}
	}
}

type ConfigFile struct {
	XMLName      xml.Name `xml:"Server"`
	Debug        bool     `xml:"debug,attr"`
	DebugSrv     string   `xml:"debugsrv,attr"`
	DocumentRoot string

	Listen []*struct {
		Log      bool   `xml:"log,attr"`
		XHeaders bool   `xml:"xheaders,attr"`
		Addr     string `xml:"addr,attr"`
		CertFile string
		KeyFile  string
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
