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
	"fmt"
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

const (
	// API limits
	maxrequests = 10000
	expire      = 3600

	// Server settings
	debug      = true
	addr       = ":8080"            // ip:port or /path/to/unix.sock
	redisaddr  = "127.0.0.1:6379"   // redis server
	staticpath = "./static"         // public website
	ipdb       = "./db/ipdb.sqlite" // geoip database
)

func main() {
	http.Handle("/", http.FileServer(http.Dir(staticpath)))
	h := GeoipHandler()
	http.HandleFunc("/csv/", h)
	http.HandleFunc("/xml/", h)
	http.HandleFunc("/json/", h)
	server := http.Server{
		Addr:         addr,
		Handler:      httpxtra.Handler{Logger: logger},
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	log.Println("FreeGeoIP server starting")
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
	db, err := sql.Open("sqlite3", ipdb)
	if err != nil {
		panic(err)
	}
	rc := redis.New(redisaddr)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case "OPTIONS":
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers",
				"X-Requested-With")
			return
		default:
			w.Header().Set("Allow", "GET, OPTIONS")
			http.Error(w, http.StatusText(405), 405)
			return
		}
		// GET
		// Check quota
		var ipkey string
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err != nil {
			ipkey = r.RemoteAddr // support for XHeaders
		} else {
			ipkey = ip
		}
		if qcs, err := rc.Get(ipkey); err != nil {
			http.Error(w, http.StatusText(503), 503) // redis down
			return
		} else if qcs == "" {
			if err := rc.Set(r.RemoteAddr, "1"); err == nil {
				rc.Expire(r.RemoteAddr, expire)
			}
		} else if qc, _ := strconv.Atoi(qcs); qc < maxrequests {
			rc.Incr(r.RemoteAddr)
		} else {
			// Out of quota, soz :(
			http.Error(w, http.StatusText(403), 403)
			return
		}
		// Parse URL and build the query.
		var ip string
		a := strings.SplitN(r.URL.Path, "/", 3)
		if len(a) == 3 && a[2] != "" {
			// e.g. /csv/google.com
			addrs, err := net.LookupHost(a[2])
			if err != nil {
				http.Error(w, http.StatusText(404), 404)
				return
			}
			ip = addrs[0]
		} else {
			ip = ipkey
		}
		geoip, err := GeoipLookup(db, ip)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch a[1] {
		case "csv":
			w.Header().Set("Content-Type", "application/csv")
			fmt.Fprintf(w, `"%s","%s","%s","%s","%s","%s",`+
				`"%s","%0.4f","%0.4f","%s","%s"`+"\r\n",
				geoip.Ip,
				geoip.CountryCode, geoip.CountryName,
				geoip.RegionCode, geoip.RegionName,
				geoip.CityName, geoip.ZipCode,
				geoip.Latitude, geoip.Longitude,
				geoip.MetroCode, geoip.AreaCode)
		case "json":
			resp, err := json.Marshal(geoip)
			if err != nil {
				if debug {
					log.Println("JSON error:", err.Error())
				}
				http.NotFound(w, r)
				return
			}
			callback := r.FormValue("callback")
			if callback != "" {
				w.Header().Set("Content-Type", "text/javascript")
				fmt.Fprintf(w, "%s(%s);\n", callback, resp)
			} else {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, "%s", resp)
			}
		case "xml":
			w.Header().Set("Content-Type", "application/xml")
			resp, err := xml.MarshalIndent(geoip, "", " ")
			if err != nil {
				if debug {
					log.Println("XML error:", err.Error())
				}
				http.Error(w, http.StatusText(500), 500)
				return
			}
			fmt.Fprintf(w, xml.Header+"%s\n", resp)
		}
	}
}

func GeoipLookup(db *sql.DB, ip string) (*GeoIP, error) {
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
		q := `SELECT
		  city_location.country_code, country_blocks.country_name,
		  city_location.region_code, region_names.region_name,
		  city_location.city_name, city_location.postal_code,
		  city_location.latitude, city_location.longitude,
		  city_location.metro_code, city_location.area_code
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
		stmt, err := db.Prepare(q)
		if err != nil {
			if debug {
				log.Println("[debug] SQLite", err.Error())
			}
			return nil, err
		}
		defer stmt.Close()
		var uintIP uint32
		b := bytes.NewBuffer(IP.To4())
		binary.Read(b, binary.BigEndian, &uintIP)
		err = stmt.QueryRow(uintIP).Scan(
			&geoip.CountryCode,
			&geoip.CountryName,
			&geoip.RegionCode,
			&geoip.RegionName,
			&geoip.CityName,
			&geoip.ZipCode,
			&geoip.Latitude,
			&geoip.Longitude,
			&geoip.MetroCode,
			&geoip.AreaCode)
		if err != nil {
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
