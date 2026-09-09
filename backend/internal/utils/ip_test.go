package utils

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func request(t *testing.T, remoteAddr, forwardedFor string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		r.Header.Set("X-Forwarded-For", forwardedFor)
	}
	return r
}

// resetProxyCache clears the memoized allowlist so each test can configure its
// own TRUSTED_PROXY_CIDRS.
func resetProxyCache(t *testing.T, cidrs string) {
	t.Helper()
	t.Setenv("TRUSTED_PROXY_CIDRS", cidrs)
	trustedProxies = nil
	trustedProxiesOnce = onceReset()
}

func TestGetClientIPIgnoresForwardedHeaderFromUntrustedPeer(t *testing.T) {
	resetProxyCache(t, "") // no trusted proxies configured

	// This is the rate-limiter bypass: a client that can set X-Forwarded-For
	// freely could otherwise present a new identity on every request.
	r := request(t, "203.0.113.9:51000", "1.2.3.4")
	if got := GetClientIP(r); got != "203.0.113.9" {
		t.Errorf("GetClientIP = %q, want the real peer 203.0.113.9", got)
	}
}

func TestGetClientIPHonoursForwardedHeaderFromTrustedProxy(t *testing.T) {
	resetProxyCache(t, "10.0.0.0/8")

	r := request(t, "10.0.0.5:51000", "198.51.100.7")
	if got := GetClientIP(r); got != "198.51.100.7" {
		t.Errorf("GetClientIP = %q, want the forwarded client 198.51.100.7", got)
	}
}

// With a chain, the rightmost address our own proxies did not add is the client.
func TestGetClientIPWalksProxyChain(t *testing.T) {
	resetProxyCache(t, "10.0.0.0/8")

	r := request(t, "10.0.0.5:51000", "198.51.100.7, 10.0.0.9")
	if got := GetClientIP(r); got != "198.51.100.7" {
		t.Errorf("GetClientIP = %q, want 198.51.100.7", got)
	}
}

func TestStripPort(t *testing.T) {
	tests := []struct{ in, want string }{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"192.168.1.1", "192.168.1.1"},
		{"[2001:db8::1]:8080", "2001:db8::1"},
		// A bare IPv6 address has no port. Splitting on the last colon used to
		// truncate it to "2001:db8:".
		{"2001:db8::1", "2001:db8::1"},
		{"[2001:db8::1]", "2001:db8::1"},
	}
	for _, tc := range tests {
		if got := stripPort(tc.in); got != tc.want {
			t.Errorf("stripPort(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// onceReset returns a fresh sync.Once so tests can re-parse TRUSTED_PROXY_CIDRS.
func onceReset() sync.Once { return sync.Once{} }
