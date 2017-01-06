package httplog

import (
	"net"
	"net/http"
	"strings"
)

// UseXForwardedFor parses the first value from the X-Forwarded-For
// header and updates the request RemoteAddr field with it, then
// call the next handler and reverts the value back at the end.
func UseXForwardedFor(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := parseXFF(r.Header.Get("X-Forwarded-For"))
		if ip != "" {
			_, port, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				addr := r.RemoteAddr
				r.RemoteAddr = net.JoinHostPort(ip, port)
				defer func() { r.RemoteAddr = addr }()
			}
		}
		next(w, r)
	}
}

func parseXFF(iplist string) string {
	for _, ip := range strings.Split(iplist, ",") {
		ip = strings.TrimSpace(ip)
		if addr := net.ParseIP(ip); addr != nil {
			return ip
		}
	}
	return ""
}
