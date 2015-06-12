// Copyright 2009-2014 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	// otto is used for testing the JSONP encoder. It's imported here
	// to make `go get` download it before `go test` fails.
	_ "github.com/robertkrimen/otto"
)

// A Query object is used to query the IP database.
//
// Currently the only database supported is MaxMind, and the query is a
// data structure with tags that are used by the maxminddb.Lookup function.
type Query interface{}

// An Encoder that can provide a query specification to be used for
// querying the IP database, and later encode the results of that
// query in a specific format.
type Encoder interface {
	// NewQuery returns a query specification that is used to query
	// the IP database. It should be a data structure with tags
	// associated to its fields describing what fields to query in
	// the IP database, such as country and city.
	//
	// See the maxminddb package documentation for details on
	// fields available for the MaxMind database.
	NewQuery() Query

	// Encode writes data to the response of an http request
	// using the results of a query to the IP database.
	//
	// It encodes the query object into a specific format such
	// as XML or JSON and writes to the response.
	//
	// The IP passed to the encoder may be the result of a DNS
	// lookup, and if there are multiple IPs associated to the
	// hostname this will be a random one from the list.
	Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error
}

// JSONEncoder encodes the results of an IP lookup as JSON.
type JSONEncoder struct {
	Indent bool
}

// NewQuery implements the Encoder interface.
func (f *JSONEncoder) NewQuery() Query {
	return &maxmindQuery{}
}

// Encode implements the Encoder interface.
func (f *JSONEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := newResponse(q.(*maxmindQuery), ip, requestLang(r))
	callback := r.FormValue("callback")
	if len(callback) > 0 {
		return f.P(w, r, record, callback)
	}
	w.Header().Set("Content-Type", "application/json")
	if f.Indent {
	}
	return json.NewEncoder(w).Encode(record)
}

// P writes JSONP to an http response.
func (f *JSONEncoder) P(w http.ResponseWriter, r *http.Request, record *responseRecord, callback string) error {
	w.Header().Set("Content-Type", "application/javascript")
	_, err := io.WriteString(w, callback+"(")
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(record)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, ");")
	return err
}

// XMLEncoder encodes the results of an IP lookup as XML.
type XMLEncoder struct {
	Indent bool
}

// NewQuery implements the Encoder interface.
func (f *XMLEncoder) NewQuery() Query {
	return &maxmindQuery{}
}

// Encode implements the Encoder interface.
func (f *XMLEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := newResponse(q.(*maxmindQuery), ip, requestLang(r))
	w.Header().Set("Content-Type", "application/xml")
	_, err := io.WriteString(w, xml.Header)
	if err != nil {
		return err
	}
	if f.Indent {
		enc := xml.NewEncoder(w)
		enc.Indent("", "\t")
		err := enc.Encode(record)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte("\n"))
		return err
	}
	return xml.NewEncoder(w).Encode(record)
}

// CSVEncoder encodes the results of an IP lookup as CSV.
type CSVEncoder struct {
	UseCRLF bool
}

// NewQuery implements the Encoder interface.
func (f *CSVEncoder) NewQuery() Query {
	return &maxmindQuery{}
}

// Encode implements the Encoder interface.
func (f *CSVEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := newResponse(q.(*maxmindQuery), ip, requestLang(r))
	w.Header().Set("Content-Type", "text/csv")
	cw := csv.NewWriter(w)
	cw.UseCRLF = f.UseCRLF
	err := cw.Write([]string{
		ip.String(),
		record.CountryCode,
		record.CountryName,
		record.RegionCode,
		record.RegionName,
		record.City,
		record.ZipCode,
		record.TimeZone,
		strconv.FormatFloat(record.Latitude, 'f', 2, 64),
		strconv.FormatFloat(record.Longitude, 'f', 2, 64),
		strconv.Itoa(int(record.MetroCode)),
	})
	if err != nil {
		return err
	}
	cw.Flush()
	return nil
}

// maxmindQuery is the object used to query the maxmind database.
//
// See the maxminddb package documentation for details.
type maxmindQuery struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Region []struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		MetroCode uint    `maxminddb:"metro_code"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`
}

// responseRecord is the object that gets encoded as the response of an
// IP lookup request. It is encoded to formats such as xml and json.
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

// newResponse translates a maxmindQuery into a responseRecord, setting
// the country, region and city names to their localized name according
// to the given lang.
//
// See the maxminddb documentation for supported languages.
func newResponse(query *maxmindQuery, ip net.IP, lang []string) *responseRecord {

	record := &responseRecord{
		IP:          ip.String(),
		CountryCode: query.Country.ISOCode,
		CountryName: localizedName(query.Country.Names, lang),
		City:        localizedName(query.City.Names, lang),
		ZipCode:     query.Postal.Code,
		TimeZone:    query.Location.TimeZone,
		Latitude:    roundFloat(query.Location.Latitude, .5, 4),
		Longitude:   roundFloat(query.Location.Longitude, .5, 4),
		MetroCode:   query.Location.MetroCode,
	}
	if len(query.Region) > 0 {
		record.RegionCode = query.Region[0].ISOCode
		record.RegionName = localizedName(query.Region[0].Names, lang)
	}
	return record
}

func requestLang(r *http.Request) (list []string) {
	// TODO: Check Accept-Charset, sort languages by qvalue.
	l := r.Header.Get("Accept-Language")
	if len(l) == 0 {
		return nil
	}
	accpt := strings.Split(l, ",")
	if len(accpt) == 0 {
		return nil
	}
	for n, name := range accpt {
		accpt[n] = strings.Trim(strings.SplitN(name, ";", 2)[0], " ")
	}
	return accpt
}

func localizedName(field map[string]string, accept []string) (name string) {
	if accept != nil {
		var f string
		var ok bool
		for _, l := range accept {
			f, ok = field[l]
			if ok {
				return f
			}
		}
	}
	return field["en"]
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
