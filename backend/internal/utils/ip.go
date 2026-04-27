package utils

import (
	"net/http"
	"strings"
)

// GetClientIP explicitly checks standard X-Forwarded-For headers first,
// and correctly strips any provided port assignments safely.
func GetClientIP(r *http.Request) string {
	// 1. Check X-Forwarded-For (populated by proxies like Nginx/Cloudflare)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For could be a comma-separated list of IP addresses
		// Get the first one (original client IP)
		ips := strings.Split(xForwardedFor, ",")
		ip := strings.TrimSpace(ips[0])
		if ip != "" {
			return stripPort(ip)
		}
	}

	// 2. Fallback to r.RemoteAddr
	return stripPort(r.RemoteAddr)
}

func stripPort(ipWithPort string) string {
	// If it's an IPv6 address with a port, it'll look like "[2001:db8::1]:8080"
	if strings.HasPrefix(ipWithPort, "[") {
		endIdx := strings.LastIndex(ipWithPort, "]")
		if endIdx != -1 {
			return ipWithPort[1:endIdx]
		}
	}
	// Normal IPv4 like "192.168.1.1:8080"
	portIdx := strings.LastIndex(ipWithPort, ":")
	if portIdx != -1 {
		return ipWithPort[:portIdx]
	}
	return ipWithPort
}
