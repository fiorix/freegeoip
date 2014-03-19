// Copyright 2013-2014 Alexandre Fiori
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"database/sql"
	"log"
	"sync"
)

type Cache struct {
	Country map[string]string
	Region  map[RegionKey]string
	City    map[int]Location
}

type RegionKey struct {
	CountryCode,
	RegionCode string
}

type Location struct {
	CountryCode,
	RegionCode,
	CityName,
	ZipCode string
	Latitude,
	Longitude float32
	MetroCode,
	AreaCode string
}

func NewCache(db *sql.DB) *Cache {
	cache := &Cache{
		Country: make(map[string]string),
		Region:  make(map[RegionKey]string),
		City:    make(map[int]Location),
	}

	var wg sync.WaitGroup

	go func() {
		wg.Add(1)

		// Load list of countries.
		row, err := db.Query(`
		SELECT
			country_code,
			country_name
		FROM
			country_blocks
		`)
		if err != nil {
			log.Fatal("Failed to load countries from db:", err)
		}

		var country_code, name string
		for row.Next() {
			if err = row.Scan(
				&country_code,
				&name,
			); err != nil {
				log.Fatal("Failed to load country from db:", err)
			}

			cache.Country[country_code] = name
		}

		row.Close()
		wg.Done()
	}()

	go func() {
		wg.Add(1)
		// Load list of regions.
		row, err := db.Query(`
		SELECT
			country_code,
			region_code,
			region_name
		FROM
			region_names
		`)
		if err != nil {
			log.Fatal("Failed to load regions from db:", err)
		}

		var country_code, region_code, name string
		for row.Next() {
			if err = row.Scan(
				&country_code,
				&region_code,
				&name,
			); err != nil {
				log.Fatal("Failed to load region from db:", err)
			}

			cache.Region[RegionKey{country_code, region_code}] = name
		}

		row.Close()
		wg.Done()
	}()

	go func() {
		wg.Add(1)
		// Load list of city locations.
		row, err := db.Query("SELECT * FROM city_location")
		if err != nil {
			log.Fatal("Failed to load cities from db:", err)
		}

		var (
			locId int
			loc   Location
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
				log.Fatal("Failed to load city from db:", err)
			}

			cache.City[locId] = loc
		}

		row.Close()
		wg.Done()
	}()

	wg.Wait()
	return cache
}

func (cache *Cache) Update(geoip *GeoIP, locId int) {
	city, ok := cache.City[locId]
	if !ok {
		return
	}

	geoip.CountryCode = city.CountryCode
	geoip.CountryName = cache.Country[city.CountryCode]

	geoip.RegionCode = city.RegionCode
	geoip.RegionName = cache.Region[RegionKey{
		city.CountryCode,
		city.RegionCode,
	}]

	geoip.CityName = city.CityName
	geoip.ZipCode = city.ZipCode
	geoip.Latitude = city.Latitude
	geoip.Longitude = city.Longitude
	geoip.MetroCode = city.MetroCode
	geoip.AreaCode = city.AreaCode
}
