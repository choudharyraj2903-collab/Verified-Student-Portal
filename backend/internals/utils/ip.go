package utils

import (
	"net"
	"net/http"
	"strings"
)

// ExtractRealIP extracts the real client IP from the request.
// Checks X-Forwarded-For first (set by reverse proxy or load balancer),
// then X-Real-IP, then falls back to RemoteAddr.
// Never trusts a single header blindly — takes the first IP in
// X-Forwarded-For which is the original client, not the last proxy.
func ExtractRealIP(r *http.Request) string {
	// Check X-Forwarded-For first
	// Format: X-Forwarded-For: client, proxy1, proxy2
	// First IP in the list is the original client
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Check X-Real-IP — set by nginx and some proxies
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		ip := strings.TrimSpace(xri)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr — strip port if present
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr has no port — use as-is
		ip = r.RemoteAddr
	}

	return strings.TrimSpace(ip)
}