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
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/fiorix/go-web/http"
	"github.com/fiorix/go-web/mux"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// API limits
	maxRequestsPerIP = 10000
	expirySeconds    = 3600

	// Server settings
	debug          = true
	listenOn       = ":8080"
	memcacheServer = "127.0.0.1:11211"
	staticPath     = "./static"
	databaseFile   = "./db/ipdb.sqlite"
)

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
	ASNID       string   `json:"asn_id,omitempty" xml:"ASNID,omitempty"`
	ASNName     string   `json:"asn_name,omitempty" xml:"ASNName,omitempty"`
}

// http://en.wikipedia.org/wiki/Reserved_IP_addresses
var reservedIPs = []net.IPNet{
	{net.IPv4(0, 0, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
	{net.IPv4(0, 0, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
	{net.IPv4(10, 0, 0, 0), net.IPv4Mask(255, 192, 0, 0)},
	{net.IPv4(100, 64, 0, 0), net.IPv4Mask(255, 0, 0, 0)},
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

func Lookup(w http.ResponseWriter, req *http.Request, db *sql.DB) {
	format, addr := req.Vars[0], req.Vars[1]
	if addr == "" {
		addr = req.RemoteAddr // port number previously removed
	} else {
		addrs, err := net.LookupHost(addr)
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		addr = addrs[0]
	}
	IP := net.ParseIP(addr)
	reserved := false
	for _, net := range reservedIPs {
		if net.Contains(IP) {
			reserved = true
			break
		}
	}
	asn := req.FormValue("asn")
	geoip := GeoIP{Ip: addr}
	if reserved {
		geoip.CountryCode = "RD"
		geoip.CountryName = "Reserved"
	} else {
		q := "SELECT " +
			"  city_location.country_code, country_blocks.country_name, " +
			"  city_location.region_code, region_names.region_name, " +
			"  city_location.city_name, city_location.postal_code, " +
			"  city_location.latitude, city_location.longitude, " +
			"  city_location.metro_code, city_location.area_code "
		if asn != "" {
			q = q + ",  asn_blocks.asn_id, asn_blocks.asn_name "
		}
		q = q + "FROM city_blocks " +
			"  NATURAL JOIN city_location " +
			"  INNER JOIN country_blocks ON " +
			"    city_location.country_code = country_blocks.country_code " +
			"  INNER JOIN region_names ON " +
			"    city_location.country_code = region_names.country_code " +
			"    AND " +
			"    city_location.region_code = region_names.region_code "
		if asn != "" {
			q = q + "  INNER JOIN asn_blocks ON " +
					"    asn_blocks.ip_end >= ? "
		}
		q = q + "WHERE city_blocks.ip_start <= ? " +
			"ORDER BY city_blocks.ip_start DESC LIMIT 1"
		stmt, err := db.Prepare(q)
		if err != nil {
			if debug {
				log.Println("[debug] SQLite", err.Error())
			}
			http.Error(w, http.StatusText(500), 500)
			return
		}
		defer stmt.Close()
		var uintIP uint32
		b := bytes.NewBuffer(IP.To4())
		binary.Read(b, binary.BigEndian, &uintIP)
		if asn != "" {
			err = stmt.QueryRow(uintIP,uintIP).Scan(
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
				&geoip.ASNID,
				&geoip.ASNName)
		} else {
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
		}
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}
	}
	switch format[0] {
	case 'c':
		w.Header().Set("Content-Type", "application/csv")
		if asn != "" {
			fmt.Fprintf(w, `"%s","%s","%s","%s","%s","%s",`+
				`"%s","%0.4f","%0.4f","%s","%s","%s","%s"`+"\r\n",
				geoip.Ip,
				geoip.CountryCode, geoip.CountryName,
				geoip.RegionCode, geoip.RegionName,
				geoip.CityName, geoip.ZipCode,
				geoip.Latitude, geoip.Longitude,
				geoip.MetroCode, geoip.AreaCode,
				geoip.ASNID, geoip.ASNName)
		} else {
			fmt.Fprintf(w, `"%s","%s","%s","%s","%s","%s",`+
				`"%s","%0.4f","%0.4f","%s","%s"`+"\r\n",
				geoip.Ip,
				geoip.CountryCode, geoip.CountryName,
				geoip.RegionCode, geoip.RegionName,
				geoip.CityName, geoip.ZipCode,
				geoip.Latitude, geoip.Longitude,
				geoip.MetroCode, geoip.AreaCode)
		}
	case 'j':
		resp, err := json.Marshal(geoip)
		if err != nil {
			if debug {
				log.Println("[debug] JSON", err.Error())
			}
			http.Error(w, http.StatusText(404), 404)
			return
		}
		callback := req.FormValue("callback")
		if callback != "" {
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, "%s(%s);\n", callback, resp)
		} else {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "%s\n", resp)
		}
	case 'x':
		w.Header().Set("Content-Type", "application/xml")
		resp, err := xml.MarshalIndent(geoip, "", " ")
		if err != nil {
			if debug {
				log.Println("[debug] XML", err.Error())
			}
			http.Error(w, http.StatusText(500), 500)
			return
		}
		fmt.Fprintf(w, xml.Header+"%s\n", resp)
	}
}

func makeHandler() http.HandlerFunc {
	db, err := sql.Open("sqlite3", databaseFile)
	if err != nil {
		panic(err)
	}
	mc := memcache.New(memcacheServer)
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// IPv4 RemoteAddr without the port number. Breaks IPv6.
		req.RemoteAddr = strings.Split(req.RemoteAddr, ":")[0]
		// Check quota
		el, err := mc.Get(req.RemoteAddr)
		if err == memcache.ErrCacheMiss {
			err = mc.Set(&memcache.Item{
				Key: req.RemoteAddr, Value: []byte("1"),
				Expiration: expirySeconds})
		}
		if err != nil {
			// Service Unavailable
			if debug {
				log.Println("[debug] memcache", err.Error())
			}
			http.Error(w, http.StatusText(503), 503)
			return
		}
		if el != nil {
			count, _ := strconv.Atoi(string(el.Value))
			if count < maxRequestsPerIP {
				mc.Increment(req.RemoteAddr, 1)
			} else {
				// Out of quota
				http.Error(w, http.StatusText(403), 403)
				return
			}
		}
		Lookup(w, req, db)
	}
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, filepath.Join(staticPath, "index.html"))
}

func StaticHandler(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, filepath.Join(staticPath, req.Vars[0]))
}

func logger(w http.ResponseWriter, req *http.Request) {
	log.Printf("HTTP %d %s %s (%s) :: %s",
		w.Status(),
		req.Method,
		req.URL.Path,
		req.RemoteAddr,
		time.Since(req.Created))
}

func main() {
	mux.HandleFunc("^/$", IndexHandler)
	mux.HandleFunc("^/static/(.*)$", StaticHandler)
	mux.HandleFunc("^/(crossdomain.xml)$", StaticHandler)
	mux.HandleFunc("^/(csv|json|xml)/(.*)$", makeHandler())
	server := http.Server{
		Addr:         listenOn,
		Handler:      mux.DefaultServeMux,
		Logger:       logger,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	log.Println("FreeGeoIP server starting")
	if e := server.ListenAndServe(); e != nil {
		log.Println(e.Error())
	}
}
