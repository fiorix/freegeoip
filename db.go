// Copyright 2009 The freegeoip authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package freegeoip

import (
	"compress/gzip"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
	"github.com/oschwald/maxminddb-golang"
)

var (
	// ErrUnavailable may be returned by DB.Lookup when the database
	// points to a URL and is not yet available because it's being
	// downloaded in background.
	ErrUnavailable = errors.New("no database available")

	// Local cached copy of a database downloaded from a URL.
	defaultDB = filepath.Join(os.TempDir(), "freegeoip", "db.gz")

	// MaxMindDB is the URL of the free MaxMind GeoLite2 database.
	MaxMindDB = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"
)

// DB is the IP geolocation database.
type DB struct {
	file        string            // Database file name.
	reader      *maxminddb.Reader // Actual db object.
	notifyQuit  chan struct{}     // Stop auto-update and watch goroutines.
	notifyOpen  chan string       // Notify when a db file is open.
	notifyError chan error        // Notify when an error occurs.
	notifyInfo  chan string       // Notify random actions for logging
	closed      bool              // Mark this db as closed.
	lastUpdated time.Time         // Last time the db was updated.
	mu          sync.RWMutex      // Protects all the above.

	updateInterval   time.Duration // Update interval.
	maxRetryInterval time.Duration // Max retry interval in case of failure.
}

// Open creates and initializes a DB from a local file.
//
// The database file is monitored by fsnotify and automatically
// reloads when the file is updated or overwritten.
func Open(dsn string) (db *DB, err error) {
	db = &DB{
		file:        dsn,
		notifyQuit:  make(chan struct{}),
		notifyOpen:  make(chan string, 1),
		notifyError: make(chan error, 1),
		notifyInfo:  make(chan string, 1),
	}
	err = db.openFile()
	if err != nil {
		db.Close()
		return nil, err
	}
	err = db.watchFile()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("fsnotify failed for %s: %s", dsn, err)
	}
	return db, nil
}

// Calculate geoipupdate URL
// The auto update URL for paid products has a fun scheme.
// Use this function to calculate that URL from various information
func GeoIPUpdateURL(hostName string, userID string, licenseKey string, productID string) (url string, err error) {
	// Get the file name from the product
	url = fmt.Sprintf("%s://%s/app/update_getfilename?product_id=%s", "https", hostName, productID)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	hexDigest := fmt.Sprintf("%x", md5.Sum(body))

	// Get our client IP address
	url = fmt.Sprintf("%s://%s/app/update_getipaddr", "https", hostName)
	resp, err = http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	challenge := []byte(fmt.Sprintf("%s%s", licenseKey, body))
	hexDigest2 := fmt.Sprintf("%x", md5.Sum(challenge))

	// Create the URL
	return fmt.Sprintf("%s://%s/app/update_secure?db_md5=%s&challenge_md5=%s&user_id=%s&edition_id=%s", "https", hostName, hexDigest, hexDigest2, userID, productID), nil
}

// OpenURL creates and initializes a DB from a URL.
// It automatically downloads and updates the file in background, and
// keeps a local copy on $TMPDIR.
func OpenURL(url string, updateInterval, maxRetryInterval time.Duration) (db *DB, err error) {
	db = &DB{
		file:             defaultDB,
		notifyQuit:       make(chan struct{}),
		notifyOpen:       make(chan string, 1),
		notifyError:      make(chan error, 1),
		notifyInfo:       make(chan string, 1),
		updateInterval:   updateInterval,
		maxRetryInterval: maxRetryInterval,
	}
	db.openFile() // Optional, might fail.
	go db.autoUpdate(url)
	err = db.watchFile()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("fsnotify failed for %s: %s", db.file, err)
	}
	return db, nil
}

func (db *DB) watchFile() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	dbdir, err := db.makeDir()
	if err != nil {
		return err
	}
	go db.watchEvents(watcher)
	return watcher.Watch(dbdir)
}

func (db *DB) watchEvents(watcher *fsnotify.Watcher) {
	for {
		select {
		case ev := <-watcher.Event:
			if ev.Name == db.file && (ev.IsCreate() || ev.IsModify()) {
				db.openFile()
			}
		case <-watcher.Error:
		case <-db.notifyQuit:
			watcher.Close()
			return
		}
		time.Sleep(time.Second) // Suppress high-rate events.
	}
}

func (db *DB) openFile() error {
	reader, err := db.newReader(db.file)
	if err != nil {
		return err
	}
	stat, err := os.Stat(db.file)
	if err != nil {
		return err
	}
	db.setReader(reader, stat.ModTime())
	return nil
}

func (db *DB) newReader(dbfile string) (*maxminddb.Reader, error) {
	f, err := os.Open(dbfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzf.Close()
	b, err := ioutil.ReadAll(gzf)
	if err != nil {
		return nil, err
	}
	return maxminddb.FromBytes(b)
}

func (db *DB) setReader(reader *maxminddb.Reader, modtime time.Time) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		reader.Close()
		return
	}
	if db.reader != nil {
		db.reader.Close()
	}
	db.reader = reader
	db.lastUpdated = modtime.UTC()
	select {
	case db.notifyOpen <- db.file:
	default:
	}
}

func (db *DB) autoUpdate(url string) {
	backoff := time.Second
	for {
		err := db.runUpdate(url)
		if err != nil {
			bs := backoff.Seconds()
			ms := db.maxRetryInterval.Seconds()
			backoff = time.Duration(math.Min(bs*math.E, ms)) * time.Second
			db.sendError(fmt.Errorf("download failed (will retry in %s): %s", backoff, err))
		} else {
			backoff = db.updateInterval
		}
		select {
		case <-db.notifyQuit:
			return
		case <-time.After(backoff):
			// Sleep till time for the next update attempt.
		}
	}
}

func (db *DB) runUpdate(url string) error {
	db.sendInfo("starting update")
	yes, err := db.needUpdate(url)
	if err != nil {
		return err
	}
	if !yes {
		return nil
	}
	tmpfile, err := db.download(url)
	if err != nil {
		return err
	}
	err = db.renameFile(tmpfile)
	if err != nil {
		// Cleanup the tempfile if renaming failed.
		os.RemoveAll(tmpfile)
	}
	db.sendInfo("finished update")
	return err
}

func (db *DB) needUpdate(url string) (bool, error) {
	stat, err := os.Stat(db.file)
	if err != nil {
		return true, nil // Local db is missing, must be downloaded.
	}
	resp, err := http.Head(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if stat.Size() != resp.ContentLength {
		return true, nil
	}
	return false, nil
}

func (db *DB) download(url string) (tmpfile string, err error) {
	db.sendInfo("starting download")
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	tmpfile = filepath.Join(os.TempDir(),
		fmt.Sprintf("_freegeoip.%d.db.gz", time.Now().UnixNano()))
	f, err := os.Create(tmpfile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}
	db.sendInfo("finished download")
	return tmpfile, nil
}

func (db *DB) makeDir() (dbdir string, err error) {
	dbdir = filepath.Dir(db.file)
	_, err = os.Stat(dbdir)
	if err != nil {
		err = os.MkdirAll(dbdir, 0755)
		if err != nil {
			return "", err
		}
	}
	return dbdir, nil
}

func (db *DB) renameFile(name string) error {
	os.Rename(db.file, db.file+".bak") // Optional, might fail.
	_, err := db.makeDir()
	if err != nil {
		return err
	}
	return os.Rename(name, db.file)
}

// Date returns the UTC date the database file was last modified.
// If no database file has been opened the behaviour of Date is undefined.
func (db *DB) Date() time.Time {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.lastUpdated
}

// NotifyClose returns a channel that is closed when the database is closed.
func (db *DB) NotifyClose() <-chan struct{} {
	return db.notifyQuit
}

// NotifyOpen returns a channel that notifies when a new database is
// loaded or reloaded. This can be used to monitor background updates
// when the DB points to a URL.
func (db *DB) NotifyOpen() (filename <-chan string) {
	return db.notifyOpen
}

// NotifyError returns a channel that notifies when an error occurs
// while downloading or reloading a DB that points to a URL.
func (db *DB) NotifyError() (errChan <-chan error) {
	return db.notifyError
}

// NotifyInfo returns a channel that notifies informational messages
// while downloading or reloading.
func (db *DB) NotifyInfo() <-chan string {
	return db.notifyInfo
}

func (db *DB) sendError(err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return
	}
	select {
	case db.notifyError <- err:
	default:
	}
}

func (db *DB) sendInfo(message string) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return
	}
	select {
	case db.notifyInfo <- message:
	default:
	}
}

// Lookup takes an IP address and a pointer to the result value to decode
// into. The result value pointed to must be a data value that corresponds
// to a record in the database. This may include a struct representation
// of the data, a map capable of holding the data or an empty interface{}
// value.
//
// If result is a pointer to a struct, the struct need not include a field
// for every value that may be in the database. If a field is not present
// in the structure, the decoder will not decode that field, reducing the
// time required to decode the record.
//
// See https://godoc.org/github.com/oschwald/maxminddb-golang#Reader.Lookup
// for details.
func (db *DB) Lookup(addr net.IP, result interface{}) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.reader != nil {
		return db.reader.Lookup(addr, result)
	}
	return ErrUnavailable
}

// DefaultQuery is the default query used for database lookups.
type DefaultQuery struct {
	Continent struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`
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

// Close the database.
func (db *DB) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()
	if !db.closed {
		db.closed = true
		close(db.notifyQuit)
		close(db.notifyOpen)
		close(db.notifyError)
		close(db.notifyInfo)
	}
	if db.reader != nil {
		db.reader.Close()
		db.reader = nil
	}
}
