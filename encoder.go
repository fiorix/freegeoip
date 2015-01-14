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
	Indent       bool
	AddContinent bool
}

// NewQuery implements the Encoder interface.
func (f *JSONEncoder) NewQuery() Query {
	return &maxmindQuery{}
}

// Encode implements the Encoder interface.
func (f *JSONEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := newResponse(q.(*maxmindQuery), ip, requestLang(r), f.AddContinent)
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
	Indent       bool
	AddContinent bool
}

// NewQuery implements the Encoder interface.
func (f *XMLEncoder) NewQuery() Query {
	return &maxmindQuery{}
}

// Encode implements the Encoder interface.
func (f *XMLEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := newResponse(q.(*maxmindQuery), ip, requestLang(r), f.AddContinent)
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
	UseCRLF      bool
	AddContinent bool
}

// NewQuery implements the Encoder interface.
func (f *CSVEncoder) NewQuery() Query {
	return &maxmindQuery{}
}

// Encode implements the Encoder interface.
func (f *CSVEncoder) Encode(w http.ResponseWriter, r *http.Request, q Query, ip net.IP) error {
	record := newResponse(q.(*maxmindQuery), ip, requestLang(r), f.AddContinent)
	w.Header().Set("Content-Type", "text/csv")
	cw := csv.NewWriter(w)
	cw.UseCRLF = f.UseCRLF
	var csvRecord []string
	if record.Continent != "" {
		csvRecord = []string{
			ip.String(),
			record.Continent,
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
		}
	} else {
		csvRecord = []string{
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
		}
	}
	err := cw.Write(csvRecord)
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
	Continent   string   `json:"continent,omitempty" xml:",omitempty"`
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
func newResponse(query *maxmindQuery, ip net.IP, lang []string, addContinent bool) *responseRecord {
	record := &responseRecord{
		IP:          ip.String(),
		CountryCode: query.Country.ISOCode,
		CountryName: localizedName(query.Country.Names, lang),
		City:        localizedName(query.City.Names, lang),
		ZipCode:     query.Postal.Code,
		TimeZone:    query.Location.TimeZone,
		Latitude:    roundFloat(query.Location.Latitude, .5, 3),
		Longitude:   roundFloat(query.Location.Longitude, .5, 3),
		MetroCode:   query.Location.MetroCode,
	}
	if addContinent {
		countryToContinent := map[string]string{
			"A1": "--", "A2": "--", "AD": "EU", "AE": "AS", "AF": "AS", "AG": "NA",
			"AI": "NA", "AL": "EU", "AM": "AS", "AN": "NA", "AO": "AF", "AP": "AS",
			"AQ": "AN", "AR": "SA", "AS": "OC", "AT": "EU", "AU": "OC", "AW": "NA",
			"AX": "EU", "AZ": "AS", "BA": "EU", "BB": "NA", "BD": "AS", "BE": "EU",
			"BF": "AF", "BG": "EU", "BH": "AS", "BI": "AF", "BJ": "AF", "BL": "NA",
			"BM": "NA", "BN": "AS", "BO": "SA", "BR": "SA", "BS": "NA", "BT": "AS",
			"BV": "AN", "BW": "AF", "BY": "EU", "BZ": "NA", "CA": "NA", "CC": "AS",
			"CD": "AF", "CF": "AF", "CG": "AF", "CH": "EU", "CI": "AF", "CK": "OC",
			"CL": "SA", "CM": "AF", "CN": "AS", "CO": "SA", "CR": "NA", "CU": "NA",
			"CV": "AF", "CX": "AS", "CY": "AS", "CZ": "EU", "DE": "EU", "DJ": "AF",
			"DK": "EU", "DM": "NA", "DO": "NA", "DZ": "AF", "EC": "SA", "EE": "EU",
			"EG": "AF", "EH": "AF", "ER": "AF", "ES": "EU", "ET": "AF", "EU": "EU",
			"FI": "EU", "FJ": "OC", "FK": "SA", "FM": "OC", "FO": "EU", "FR": "EU",
			"FX": "EU", "GA": "AF", "GB": "EU", "GD": "NA", "GE": "AS", "GF": "SA",
			"GG": "EU", "GH": "AF", "GI": "EU", "GL": "NA", "GM": "AF", "GN": "AF",
			"GP": "NA", "GQ": "AF", "GR": "EU", "GS": "AN", "GT": "NA", "GU": "OC",
			"GW": "AF", "GY": "SA", "HK": "AS", "HM": "AN", "HN": "NA", "HR": "EU",
			"HT": "NA", "HU": "EU", "ID": "AS", "IE": "EU", "IL": "AS", "IM": "EU",
			"IN": "AS", "IO": "AS", "IQ": "AS", "IR": "AS", "IS": "EU", "IT": "EU",
			"JE": "EU", "JM": "NA", "JO": "AS", "JP": "AS", "KE": "AF", "KG": "AS",
			"KH": "AS", "KI": "OC", "KM": "AF", "KN": "NA", "KP": "AS", "KR": "AS",
			"KW": "AS", "KY": "NA", "KZ": "AS", "LA": "AS", "LB": "AS", "LC": "NA",
			"LI": "EU", "LK": "AS", "LR": "AF", "LS": "AF", "LT": "EU", "LU": "EU",
			"LV": "EU", "LY": "AF", "MA": "AF", "MC": "EU", "MD": "EU", "ME": "EU",
			"MF": "NA", "MG": "AF", "MH": "OC", "MK": "EU", "ML": "AF", "MM": "AS",
			"MN": "AS", "MO": "AS", "MP": "OC", "MQ": "NA", "MR": "AF", "MS": "NA",
			"MT": "EU", "MU": "AF", "MV": "AS", "MW": "AF", "MX": "NA", "MY": "AS",
			"MZ": "AF", "NA": "AF", "NC": "OC", "NE": "AF", "NF": "OC", "NG": "AF",
			"NI": "NA", "NL": "EU", "NO": "EU", "NP": "AS", "NR": "OC", "NU": "OC",
			"NZ": "OC", "O1": "--", "OM": "AS", "PA": "NA", "PE": "SA", "PF": "OC",
			"PG": "OC", "PH": "AS", "PK": "AS", "PL": "EU", "PM": "NA", "PN": "OC",
			"PR": "NA", "PS": "AS", "PT": "EU", "PW": "OC", "PY": "SA", "QA": "AS",
			"RE": "AF", "RO": "EU", "RS": "EU", "RU": "EU", "RW": "AF", "SA": "AS",
			"SB": "OC", "SC": "AF", "SD": "AF", "SE": "EU", "SG": "AS", "SH": "AF",
			"SI": "EU", "SJ": "EU", "SK": "EU", "SL": "AF", "SM": "EU", "SN": "AF",
			"SO": "AF", "SR": "SA", "ST": "AF", "SV": "NA", "SY": "AS", "SZ": "AF",
			"TC": "NA", "TD": "AF", "TF": "AN", "TG": "AF", "TH": "AS", "TJ": "AS",
			"TK": "OC", "TL": "AS", "TM": "AS", "TN": "AF", "TO": "OC", "TR": "EU",
			"TT": "NA", "TV": "OC", "TW": "AS", "TZ": "AF", "UA": "EU", "UG": "AF",
			"UM": "OC", "US": "NA", "UY": "SA", "UZ": "AS", "VA": "EU", "VC": "NA",
			"VE": "SA", "VG": "NA", "VI": "NA", "VN": "AS", "VU": "OC", "WF": "OC",
			"WS": "OC", "YE": "AS", "YT": "AF", "ZA": "AF", "ZM": "AF", "ZW": "AF",
		}
		record.Continent = countryToContinent[query.Country.ISOCode]
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
