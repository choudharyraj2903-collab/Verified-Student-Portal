package middleware

import (
	"fmt"
	"net"
	"net/http"

	"student_portal/backend/config"
	"student_portal/backend/internals/utils"
)

// =========================================================================
// Structs
// =========================================================================

type CampusIPGuard struct {
	allowedRanges []*net.IPNet
	auditLogger   *AuditLogger
}

// =========================================================================
// Constructor
// =========================================================================

// NewCampusIPGuard parses the CIDR ranges from config at startup.
// Returns error if no ranges are configured — guard cannot function without
// at least one range to check against.
func NewCampusIPGuard(cfg *config.AppConfig, al *AuditLogger) (*CampusIPGuard, error) {
	if len(cfg.Campus.IP_RANGES) == 0 {
		return nil, fmt.Errorf("campus IP guard requires at least one IP range in CAMPUS_IP_RANGES")
	}

	ranges := cfg.Campus.IP_RANGES

	// In development, also allow loopback so local testing works
	if cfg.Server.APP_ENV == "development" {
		for _, cidr := range []string{"127.0.0.1/8", "::1/128"} {
			_, ipnet, err := net.ParseCIDR(cidr)
			if err == nil {
				ranges = append(ranges, ipnet)
			}
		}
	}

	return &CampusIPGuard{
		allowedRanges: ranges,
		auditLogger:   al,
	}, nil
}

// =========================================================================
// Middleware
// =========================================================================

// Guard restricts access to requests originating from within the campus
// network. Applied only to /admin/* routes after Authenticate has already
// confirmed the user is an admin.
// An authenticated admin hitting this from outside campus is a CRITICAL
// signal — could be a stolen token used remotely.
func (g *CampusIPGuard) Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Step 1 — Extract real client IP
		ipStr := utils.ExtractRealIP(r)

		// Step 2 — Parse the IP string
		parsedIP := net.ParseIP(ipStr)
		if parsedIP == nil {
			// Unparseable IP — reject immediately
			_ = g.auditLogger.DirectLog(r.Context(), &AuditEvent{
				EventType: "UNAUTHORIZED_SCOPE_ACCESS",
				Severity:  "CRITICAL",
				Metadata: map[string]any{
					"reason":   "unparseable_ip",
					"raw_ip":   ipStr,
					"endpoint": r.URL.Path,
					"method":   r.Method,
				},
			})
			utils.SendForbidden(w)
			return
		}

		// Step 3 — Check against all allowed campus IP ranges
		inCampus := false
		for _, network := range g.allowedRanges {
			if network.Contains(parsedIP) {
				inCampus = true
				break
			}
		}

		// Step 4 — Reject if not in campus network
		if !inCampus {
			// Get the authenticated user if available for richer log
			userID := ""
			userRole := ""
			if user, ok := UserFromContext(r.Context()); ok {
				userID = user.ID
				userRole = user.Role
			}

			_ = g.auditLogger.DirectLog(r.Context(), &AuditEvent{
				EventType: "UNAUTHORIZED_SCOPE_ACCESS",
				Severity:  "CRITICAL",
				UserID:    userID,
				Metadata: map[string]any{
					"reason":   "outside_campus_network",
					"endpoint": r.URL.Path,
					"method":   r.Method,
					"role":     userRole,
				},
			})
			utils.SendForbidden(w)
			return
		}

		// Step 5 — IP is within campus network — pass through
		next.ServeHTTP(w, r)
	})
}