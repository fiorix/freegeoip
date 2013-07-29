// Copyright 2013 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fiorix/go-redis/redis"
	"github.com/fiorix/go-web/httpxtra"
	_ "github.com/mattn/go-sqlite3"
)

type Settings struct {
	XMLName      xml.Name `xml:"Server"`
	Debug        bool     `xml:"debug,attr"`
	XHeaders     bool     `xml:"xheaders,attr"`
	Addr         string   `xml:"addr,attr"`
	DocumentRoot string
	IPDB         struct {
		File      string `xml:",attr"`
		CacheSize string `xml:",attr"`
	}
	Limit struct {
		MaxRequests int `xml:",attr"`
		Expire      int `xml:",attr"`
	}
	Redis []string `xml:"Redis>Addr"`
}

var conf *Settings


func main() {
	cf := flag.String("config", "freegeoip.conf", "set config file")
	flag.Parse()
	if buf, err := ioutil.ReadFile(*cf); err != nil {
		log.Fatal(err)
	} else {
		conf = &Settings{}
		if err := xml.Unmarshal(buf, conf); err != nil {
			log.Fatal(err)
		}
	}
	http.Handle("/", http.FileServer(http.Dir(conf.DocumentRoot)))
	h := GeoipHandler()
	http.HandleFunc("/csv/", h)
	http.HandleFunc("/xml/", h)
	http.HandleFunc("/json/", h)
	server := http.Server{
		Addr: conf.Addr,
		Handler: httpxtra.Handler{
			Logger:   logger,
			XHeaders: conf.XHeaders,
		},
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	log.Printf("FreeGeoIP server starting on %s (xheaders=%t)",
		conf.Addr, conf.XHeaders)
	if e := httpxtra.ListenAndServe(server); e != nil {
		log.Println(e.Error())
	}
}

func logger(r *http.Request, created time.Time, status, bytes int) {
	//fmt.Println(httpxtra.ApacheCommonLog(r, created, status, bytes))
	log.Printf("HTTP %d %s %s (%s) :: %s",
		status,
		r.Method,
		r.URL.Path,
		r.RemoteAddr,
		time.Since(created))
}

// GeoipHandler handles GET on /csv, /xml and /json.
func GeoipHandler() http.HandlerFunc {
	db, err := sql.Open("sqlite3", conf.IPDB.File)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("PRAGMA cache_size=" + conf.IPDB.CacheSize)
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	//defer stmt.Close()
	rc := redis.New(conf.Redis...)
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
		// GET continuing...
		var (
			ip, ipkey string
			err       error
			ok        bool
		)
		if ip, _, err = net.SplitHostPort(r.RemoteAddr); err != nil {
			ipkey = r.RemoteAddr // Support for XHeaders
		} else {
			ipkey = ip
		}
		// Check quota
		if ok, err = HasQuota(rc, &ipkey); err != nil {
			// Redis down?
			if conf.Debug {
				log.Println("Redis error:", err.Error())
			}
			http.Error(w, http.StatusText(503), 503)
			return
		} else if !ok {
			// Over quota, soz :(
			http.Error(w, http.StatusText(403), 403)
			return
		}
		// Parse URL (e.g. /csv/ip, /xml/)
		a := strings.SplitN(r.URL.Path, "/", 3)
		if len(a) == 3 && a[2] != "" {
			addrs, err := net.LookupHost(a[2])
			if err != nil {
				// DNS lookup failed, assume host not found.
				http.Error(w, http.StatusText(404), 404)
				return
			}
			ip = addrs[0]
		} else {
			ip = ipkey
		}
		// Query the db
		geoip, err := GeoipLookup(stmt, ip)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch a[1][0] {
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
		case 'j': // json
			resp, err := json.Marshal(geoip)
			if err != nil {
				if conf.Debug {
					log.Println("JSON error:", err.Error())
				}
				http.NotFound(w, r)
				return
			}
			callback := r.FormValue("callback")
			if callback != "" {
				w.Header().Set("Content-Type", "text/javascript")
				fmt.Fprintf(w, "%s(%s);", callback, resp)
			} else {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, "%s", resp)
			}
		case 'x': // xml
			w.Header().Set("Content-Type", "application/xml")
			resp, err := xml.MarshalIndent(geoip, "", " ")
			if err != nil {
				if conf.Debug {
					log.Println("XML error:", err.Error())
				}
				http.Error(w, http.StatusText(500), 500)
				return
			}
			fmt.Fprintf(w, xml.Header+"%s\n", resp)
		}
	}
}

func HasQuota(rc *redis.Client, ipkey *string) (bool, error) {
	if ns, err := rc.Get(*ipkey); err != nil {
		return false, err
	} else if ns == "" {
		if err := rc.Set(*ipkey, "1"); err != nil {
			return false, err
		}
		rc.Expire(*ipkey, conf.Limit.Expire)
	} else if n, _ := strconv.Atoi(ns); n < conf.Limit.MaxRequests {
		rc.Incr(*ipkey)
	} else {
		return false, nil
	}
	return true, nil
}

const query = `SELECT
  city_location.country_code,
  country_blocks.country_name,
  city_location.region_code,
  region_names.region_name,
  city_location.city_name,
  city_location.postal_code,
  city_location.latitude,
  city_location.longitude,
  city_location.metro_code,
  city_location.area_code
FROM city_blocks
  NATURAL JOIN city_location
  INNER JOIN country_blocks ON
    city_location.country_code = country_blocks.country_code
  LEFT OUTER JOIN region_names ON
    city_location.country_code = region_names.country_code
    AND
    city_location.region_code = region_names.region_code
WHERE city_blocks.ip_start <= ?
ORDER BY city_blocks.ip_start DESC LIMIT 1`

func GeoipLookup(stmt *sql.Stmt, ip string) (*GeoIP, error) {
	IP := net.ParseIP(ip)
	reserved := false
	for _, net := range reservedIPs {
		if net.Contains(IP) {
			reserved = true
			break
		}
	}
	geoip := GeoIP{Ip: ip}
	if reserved {
		geoip.CountryCode = "RD"
		geoip.CountryName = "Reserved"
	} else {
		var uintIP uint32
		b := bytes.NewBuffer(IP.To4())
		binary.Read(b, binary.BigEndian, &uintIP)
		if err := stmt.QueryRow(uintIP).Scan(
			&geoip.CountryCode,
			&geoip.CountryName,
			&geoip.RegionCode,
			&geoip.RegionName,
			&geoip.CityName,
			&geoip.ZipCode,
			&geoip.Latitude,
			&geoip.Longitude,
			&geoip.MetroCode,
			&geoip.AreaCode,
		); err != nil {
			return nil, err
		}
	}
	return &geoip, nil
}

type GeoIP struct {
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
	AreaCode    string   `json:"areacode"`
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
