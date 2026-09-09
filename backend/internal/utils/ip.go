package utils

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// trustedProxies holds the CIDR blocks whose X-Forwarded-For headers we honour.
// It is parsed once from TRUSTED_PROXY_CIDRS (comma-separated). When the list is
// empty every request is treated as direct, which is the safe default: the header
// is attacker-controlled, so trusting it unconditionally lets anyone defeat a
// per-IP rate limiter simply by rotating the value.
var (
	trustedProxiesOnce sync.Once
	trustedProxies     []*net.IPNet
)

func loadTrustedProxies() {
	raw := os.Getenv("TRUSTED_PROXY_CIDRS")
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// Accept both a bare address and a CIDR block.
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				trustedProxies = append(trustedProxies, &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(bits, bits),
				})
			}
			continue
		}
		if _, block, err := net.ParseCIDR(entry); err == nil {
			trustedProxies = append(trustedProxies, block)
		}
	}
}

func isTrustedProxy(ip net.IP) bool {
	trustedProxiesOnce.Do(loadTrustedProxies)
	if ip == nil {
		return false
	}
	for _, block := range trustedProxies {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// GetClientIP returns the address to attribute a request to. X-Forwarded-For is
// consulted only when the immediate peer is a configured trusted proxy.
func GetClientIP(r *http.Request) string {
	peer := stripPort(r.RemoteAddr)

	if !isTrustedProxy(net.ParseIP(peer)) {
		return peer
	}

	// Walk right-to-left and take the last address the chain did not vouch for:
	// entries appended by our own proxies are trustworthy, anything before them
	// was supplied by the client.
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return peer
	}
	hops := strings.Split(forwarded, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		ip := net.ParseIP(hop)
		if ip == nil {
			continue
		}
		if !isTrustedProxy(ip) {
			return ip.String()
		}
	}
	return peer
}

// stripPort removes a trailing port from an address. net.SplitHostPort handles
// bracketed IPv6 correctly; a bare IPv6 address has no port to strip and is
// returned unchanged rather than truncated at its last colon.
func stripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return strings.Trim(addr, "[]")
}
