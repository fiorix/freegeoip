package httprl

import (
	"sync"
	"time"
)

// Map is a rate limiter implementation using a map and goroutine
// to expire keys.
type Map struct {
	m    sync.Mutex
	s    map[string]*rldata
	p    time.Duration
	stop chan struct{}
}

type rldata struct {
	Count  uint64
	Expire time.Time
}

// NewMap creates and initializes a new Map. The precision determines
// how often the map is scanned for expired keys, in seconds.
func NewMap(precision int32) *Map {
	return &Map{
		s: make(map[string]*rldata),
		p: time.Duration(precision) * time.Second,
	}
}

// Hit implements the httprl.Backend interface.
func (m *Map) Hit(key string, ttlsec int32) (count uint64, remttl int32, err error) {
	m.m.Lock()
	defer m.m.Unlock()
	v, ok := m.s[key]
	if !ok {
		m.s[key] = &rldata{
			Count:  1,
			Expire: time.Now().Add(time.Duration(ttlsec) * time.Second),
		}
		return 1, ttlsec, nil
	}
	v.Count++
	rttl := v.Expire.Sub(time.Now()).Seconds()
	if rttl < 1 {
		return v.Count, 0, nil
	}
	return v.Count, int32(rttl), nil
}

// Start starts the internal goroutine that scans the map for
// expired keys and remove them.
func (m *Map) Start() {
	m.m.Lock()
	defer m.m.Unlock()
	if m.stop != nil {
		return
	}
	m.stop = make(chan struct{})
	ready := make(chan struct{})
	go m.run(ready)
	<-ready
}

// Stop stops the internal goroutine started by Start.
func (m *Map) Stop() {
	m.m.Lock()
	defer m.m.Unlock()
	if m.stop != nil {
		close(m.stop)
	}
}

func (m *Map) run(ready chan struct{}) {
	tick := time.NewTicker(m.p)
	close(ready)
	for {
		select {
		case <-m.stop:
			tick.Stop()
			m.m.Lock()
			m.stop = nil
			m.m.Unlock()
		case <-tick.C:
			m.clear()
		}
	}
}

func (m *Map) clear() {
	now := time.Now()
	m.m.Lock()
	for k, v := range m.s {
		if v.Expire.Sub(now) <= 0 {
			delete(m.s, k)
		}
	}
	m.m.Unlock()
}
