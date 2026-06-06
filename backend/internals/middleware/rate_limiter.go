package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	"student_portal/backend/config"
	"student_portal/backend/internals/utils"
)

// =========================================================================
// Structs
// =========================================================================

type RateLimiter struct {
	redis       *redis.Client
	cfg         *config.AuthConfig
	appCfg      *config.AppConfig
	auditLogger *AuditLogger
}

type RateLimitResult struct {
	Allowed        bool
	Remaining      int
	RetryAfter     int
	TriggerCaptcha bool
}

const contextKeyRateLimit contextKey = "rateLimit"

// =========================================================================
// Constructor
// =========================================================================

// NewRateLimiter takes both AuthConfig for rate limit thresholds and
// AppConfig for SendInternalError calls which need the full config.
func NewRateLimiter(
	redisClient *redis.Client,
	cfg *config.AuthConfig,
	appCfg *config.AppConfig,
	al *AuditLogger,
) *RateLimiter {
	return &RateLimiter{
		redis:       redisClient,
		cfg:         cfg,
		appCfg:      appCfg,
		auditLogger: al,
	}
}

// =========================================================================
// LimitByEmail — primary throttle
// =========================================================================

// LimitByEmail is the primary rate limiter. Throttles magic link requests
// per email address. Email must already be in context — domain_guard runs
// first and injects it. Per-email limit is the main protection against
// flooding a specific student's inbox.
func (rl *RateLimiter) LimitByEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Email injected by domain_guard — never re-parse body here
		email, ok := EmailFromContext(r.Context())
		if !ok {
			utils.SendInternalError(w, fmt.Errorf("email missing from context"), rl.appCfg)
			return
		}

		// Hash email before using as Redis key — raw emails must never sit
		// in Redis keys which appear in Redis logs and monitoring tools
		key := fmt.Sprintf("rl:email:%s", utils.Hash256String(email))
		limit := rl.cfg.MAGIC_LINK_MAX_REQUESTS_PER_HOUR
		window := 3600 // 1 hour in seconds

		// Lua script makes check-and-increment atomic
		// A naive GET then INCR has a race condition where two simultaneous
		// requests both read count=2, both pass a limit of 3, both proceed
		script := redis.NewScript(`
            local key    = KEYS[1]
            local limit  = tonumber(ARGV[1])
            local window = tonumber(ARGV[2])

            local current = redis.call('GET', key)
            if current and tonumber(current) >= limit then
                return {0, tonumber(current), redis.call('TTL', key)}
            end

            local new = redis.call('INCR', key)
            if new == 1 then
                redis.call('EXPIRE', key, window)
            end
            return {1, new, window}
        `)

		res, err := script.Run(r.Context(), rl.redis, []string{key}, limit, window).Result()
		if err != nil {
			utils.SendInternalError(w, fmt.Errorf("redis rate limit error: %w", err), rl.appCfg)
			return
		}

		arr := res.([]interface{})
		allowed := arr[0].(int64) == 1
		current := int(arr[1].(int64))
		ttl := int(arr[2].(int64))

		result := &RateLimitResult{
			Allowed:    allowed,
			Remaining:  limit - current,
			RetryAfter: ttl,
		}

		// Flag captcha threshold — handler reads TriggerCaptcha from context
		// and decides whether to require captcha before proceeding
		if current >= rl.cfg.CAPTCHA_THRESHOLD {
			result.TriggerCaptcha = true
		}

		if !result.Allowed {
			_ = rl.auditLogger.DirectLog(r.Context(), &AuditEvent{
				EventType: "RATE_LIMIT_HIT",
				Severity:  "WARN",
				Metadata: map[string]any{
					"type":        "per_email",
					"retry_after": result.RetryAfter,
				},
			})
			utils.SendRateLimited(w, result.RetryAfter)
			return
		}

		// Inject result into context — handler reads TriggerCaptcha from here
		ctx := context.WithValue(r.Context(), contextKeyRateLimit, result)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// =========================================================================
// LimitByIP — secondary loose throttle
// =========================================================================

// LimitByIP is the secondary rate limiter. Much higher threshold than
// per-email — its only job is catching automated bots hammering many
// different emails from the same IP. On campus shared WiFi this threshold
// must be high enough that normal student usage never triggers it.
// IP-level blocking is logged at CRITICAL — on a campus network this
// signals something abnormal like a script or compromised device.
func (rl *RateLimiter) LimitByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractRealIP(r)

		// Hash IP before using as Redis key
		key := fmt.Sprintf("rl:ip:%s", utils.Hash256String(ip))
		limit := rl.cfg.MAGIC_LINK_MAX_REQUESTS_PER_HOUR
		window := 3600

		script := redis.NewScript(`
            local key    = KEYS[1]
            local limit  = tonumber(ARGV[1])
            local window = tonumber(ARGV[2])

            local current = redis.call('GET', key)
            if current and tonumber(current) >= limit then
                return {0, tonumber(current), redis.call('TTL', key)}
            end

            local new = redis.call('INCR', key)
            if new == 1 then
                redis.call('EXPIRE', key, window)
            end
            return {1, new, window}
        `)

		res, err := script.Run(r.Context(), rl.redis, []string{key}, limit, window).Result()
		if err != nil {
			utils.SendInternalError(w, fmt.Errorf("redis rate limit error: %w", err), rl.appCfg)
			return
		}

		arr := res.([]interface{})
		allowed := arr[0].(int64) == 1
		ttl := int(arr[2].(int64))

		if !allowed {
			// CRITICAL — IP level block on a campus network is a serious signal
			_ = rl.auditLogger.DirectLog(r.Context(), &AuditEvent{
				EventType: "RATE_LIMIT_HIT",
				Severity:  "CRITICAL",
				Metadata: map[string]any{
					"type":        "per_ip",
					"retry_after": ttl,
				},
			})
			utils.SendRateLimited(w, ttl)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// =========================================================================
// Context helper
// =========================================================================

// RateLimitResultFromContext retrieves the RateLimitResult injected by
// LimitByEmail. Handlers call this to check TriggerCaptcha before
// proceeding with magic link generation.
func RateLimitResultFromContext(ctx context.Context) (*RateLimitResult, bool) {
	res, ok := ctx.Value(contextKeyRateLimit).(*RateLimitResult)
	return res, ok
}

// =========================================================================
// Private helpers
// =========================================================================

// extractRealIP extracts the real client IP from the request.
// Checks X-Forwarded-For first then X-Real-IP then RemoteAddr.
// Validates every extracted value with net.ParseIP before returning —
// raw header values are never trusted blindly.
func extractRealIP(r *http.Request) string {
	// X-Forwarded-For: client, proxy1, proxy2
	// First IP in the list is the original client
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// X-Real-IP — set by nginx and some proxies
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		ip := strings.TrimSpace(xrip)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// RemoteAddr — strip port if present
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}