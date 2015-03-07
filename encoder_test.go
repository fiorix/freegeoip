// Copyright 2009-2014 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/robertkrimen/otto"
)

func TestCSVEncoder(t *testing.T) {
	_, srv, err := runServer("/csv/", &CSVEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/csv/8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	row, err := csv.NewReader(resp.Body).Read()
	if err != nil {
		t.Fatal(err)
	}
	if row[1] != "US" {
		t.Fatalf("Unexpected country code in record: %#v", row)
	}
}

func TestXMLEncoder(t *testing.T) {
	_, srv, err := runServer("/xml/", &XMLEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/xml/8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	var record responseRecord
	err = xml.NewDecoder(resp.Body).Decode(&record)
	if err != nil {
		t.Fatal(err)
	}
	if record.CountryCode != "US" {
		t.Fatalf("Unexpected country code in record: %#v", record.CountryCode)
	}
}

func TestXMLEncoderIndent(t *testing.T) {
	// TODO: validate indentation?
	_, srv, err := runServer("/xml/", &XMLEncoder{Indent: true})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/xml/8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	var record responseRecord
	err = xml.NewDecoder(resp.Body).Decode(&record)
	if err != nil {
		t.Fatal(err)
	}
	if record.CountryCode != "US" {
		t.Fatalf("Unexpected country code in record: %#v", record.CountryCode)
	}
}

func TestJSONEncoder(t *testing.T) {
	_, srv, err := runServer("/json/", &JSONEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/json/8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	var record responseRecord
	err = json.NewDecoder(resp.Body).Decode(&record)
	if err != nil {
		t.Fatal(err)
	}
	if record.CountryCode != "US" {
		t.Fatalf("Unexpected country code in record: %#v", record.CountryCode)
	}
}

func TestJSONEncoderIndent(t *testing.T) {
	// TODO: validate indentation?
	_, srv, err := runServer("/json/", &JSONEncoder{Indent: true})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/json/8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	var record responseRecord
	err = json.NewDecoder(resp.Body).Decode(&record)
	if err != nil {
		t.Fatal(err)
	}
	if record.CountryCode != "US" {
		t.Fatalf("Unexpected country code in record: %#v", record.CountryCode)
	}
}

func TestJSONPEncoder(t *testing.T) {
	_, srv, err := runServer("/json/", &JSONEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/json/8.8.8.8?callback=f")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	code := bytes.NewBuffer([]byte(`
		function f(record) {
			set(record.country_code);
		};
	`))
	code.Write(b)
	vm := otto.New()
	var countryCode string
	vm.Set("set", func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) > 0 {
			countryCode = call.Argument(0).String()
		}
		return otto.UndefinedValue()
	})
	_, err = vm.Run(code.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if countryCode != "US" {
		t.Fatalf("Unexpected country code in record: %#v", countryCode)
	}
}

func TestRequestLang(t *testing.T) {
	r := http.Request{}
	list := requestLang(&r)
	if list != nil {
		t.Fatal("Unexpected list is not nil")
	}
	r.Header = map[string][]string{
		"Accept-Language": {"en-us,en;q=0.5"},
	}
	want := []string{"en-us", "en"}
	list = requestLang(&r)
	if len(list) != 2 {
		t.Fatal("Unexpected list length:", len(list))
	}
	for i, lang := range want {
		if list[i] != lang {
			t.Fatal("Unexpected item in list:", list[i])
		}
	}
}

func TestLocalizedName(t *testing.T) {
	names := map[string]string{
		"de":    "USA",
		"en":    "United States",
		"es":    "Estados Unidos",
		"fr":    "États-Unis",
		"ja":    "アメリカ合衆国",
		"pt-BR": "Estados Unidos",
		"ru":    "Сша",
		"zh-CN": "美国",
	}
	mkReq := func(lang string) *http.Request {
		return &http.Request{
			Header: map[string][]string{
				"Accept-Language": {lang},
			},
		}
	}
	test := map[string]string{
		"pt-BR,en":                  "Estados Unidos",
		"pt-br":                     "United States",
		"es-ES,ru;q=0.8,q=0.2":      "Сша",
		"da, en-gb;q=0.8, en;q=0.7": "United States",
		"da, fr;q=0.8, en;q=0.7":    "États-Unis",
		"da, de;q=0.5, zh-CN;q=0.8": "USA", // TODO: Use qvalue.
		"da, es":                    "Estados Unidos",
		"es-ES, ja":                 "アメリカ合衆国",
	}
	for k, v := range test {
		name := localizedName(names, requestLang(mkReq(k)))
		if name != v {
			t.Fatalf("Unexpected name: want %q, have %q", v, name)
		}
	}
}
