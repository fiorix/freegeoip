// Package httprl provides a rate limiter for http servers.
package httprl

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
)

// Errors.
var (
	ErrLimitExceeded      = errors.New("Rate limit exceeded")
	ErrServiceUnavailable = errors.New("Service unavailable, try again later")
)

// A KeyMaker makes keys from the http.Request object to the RateLimiter.
type KeyMaker func(r *http.Request) string

// DefaultKeyMaker is a KeyMaker that returns the client IP
// address from the request, without the port.
var DefaultKeyMaker = func(r *http.Request) string {
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return addr
}

// Backend defines an interface for rate limiters. It can be implemented
// by in-memory caches such as redis and memcache.
type Backend interface {
	// Hit tells the backend that a given key has been hit, and
	// if it does not exist in the backend, should be set to 1
	// with the given time-to-live, in seconds. Returns the
	// hit count and remaining ttl for the given key.
	Hit(key string, ttlsec int32) (count uint64, remttl int32, err error)
}

// A RateLimiter is an http.Handler that wraps another handler,
// and calls it up to a certain limit, per time interval.
type RateLimiter struct {
	Backend                Backend          // Backend for the rate limiter
	Limit                  uint64           // Maximum number of requests per interval
	Interval               int32            // Interval in seconds
	KeyMaker               KeyMaker         // Function to generate a key from the request (DefaultKeyMaker)
	Policy                 Policy           // Policy when backend fails (default BlockPolicy)
	ErrorLog               *log.Logger      // Optional logger for backend errors (optional)
	LimitExceededFunc      http.HandlerFunc // Function called when limit exceeded (optional)
	ServiceUnavailableFunc http.HandlerFunc // Function called when backend is unavailable (optional)
}

// Policy defines the rate limiter policy to apply when the backend fails.
type Policy int

const (
	// BlockPolicy blocks requests when backend is unavailable.
	BlockPolicy Policy = iota
	// AllowPolicy allows requests when backend is unavailable.
	AllowPolicy
)

// Handle handles incoming requests by applying rate limit, and calls
// f.ServeHTTP if the client is under the limit.
func (rl *RateLimiter) Handle(next http.Handler) http.Handler {
	return rl.HandleFunc(next.ServeHTTP)
}

// HandleFunc handles incoming requests by applying rate limit, and calls
// f if the client is under the limit.
func (rl *RateLimiter) HandleFunc(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := rl.limit(w, r); err != nil {
			return
		}
		f(w, r)
	}
}

func (rl *RateLimiter) limit(w http.ResponseWriter, r *http.Request) error {
	km := rl.KeyMaker
	if km == nil {
		km = DefaultKeyMaker
	}
	k := km(r)
	nreq, remttl, err := rl.Backend.Hit(k, rl.Interval)
	if err != nil {
		if rl.ErrorLog != nil {
			rl.ErrorLog.Printf("ratelimiter: %v", err)
		}
		if rl.Policy == BlockPolicy {
			if f := rl.ServiceUnavailableFunc; f != nil {
				f(w, r)
				return ErrServiceUnavailable
			}
			return errServiceUnavailable(w)
		}
		return nil // Allow.
	}
	w.Header().Set("X-RateLimit-Limit", strconv.FormatUint(rl.Limit, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(remttl)))
	switch {
	case nreq == rl.Limit:
		w.Header().Set("X-RateLimit-Remaining", "0")
		return nil
	case nreq > rl.Limit:
		w.Header().Set("X-RateLimit-Remaining", "0")
		if f := rl.LimitExceededFunc; f != nil {
			f(w, r)
			return ErrLimitExceeded
		}
		return errLimitExceeded(w)
	}
	rem := rl.Limit - nreq
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatUint(rem, 10))
	return nil
}

func errServiceUnavailable(w http.ResponseWriter) error {
	http.Error(w, ErrServiceUnavailable.Error(), http.StatusServiceUnavailable)
	return ErrServiceUnavailable
}

func errLimitExceeded(w http.ResponseWriter) error {
	http.Error(w, ErrLimitExceeded.Error(), http.StatusForbidden)
	return ErrLimitExceeded
}
