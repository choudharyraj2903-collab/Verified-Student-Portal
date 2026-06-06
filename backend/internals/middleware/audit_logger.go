package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"student_portal/backend/config"
	"student_portal/backend/db"
	"student_portal/backend/internals/utils"
)

type AuditLogger struct {
	db  *db.DB
	cfg *config.AppConfig
}

type AuditEvent struct {
	EventType string         `json:"event_type"`
	Severity  string         `json:"severity"`
	UserID    string         `json:"user_id,omitempty"`
	Metadata  map[string]any `json:"metadata"`
}

const contextKeyAuditEvent contextKey = "auditEvent"

// Constructor
func NewAuditLogger(database *db.DB, cfg *config.AppConfig) *AuditLogger {
	return &AuditLogger{db: database, cfg: cfg}
}

// =========================================================================
// ResponseRecorder — wraps ResponseWriter to capture status code
// =========================================================================

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// =========================================================================
// Log — middleware wrapper
// =========================================================================

// Log runs after every handler completes. Reads the AuditEvent injected
// into context by the handler via SetAuditEvent, enriches it with request
// metadata, and writes it to audit_logs asynchronously.
func (al *AuditLogger) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := newResponseRecorder(w)

		// Run handler first — audit happens after response is written
		next.ServeHTTP(wrapped, r)

		// Read audit event signaled by the handler
		event, ok := AuditEventFromContext(r.Context())
		if !ok {
			// No event signaled — public or non-audited route, skip silently
			return
		}

		// Enrich with request metadata
		// Never log raw UA string, raw IP, raw tokens, or PII here
		if event.Metadata == nil {
			event.Metadata = make(map[string]any)
		}
		event.Metadata["http_status"] = wrapped.statusCode
		event.Metadata["method"]      = r.Method
		event.Metadata["path"]        = r.URL.Path
		event.Metadata["ua_hash"]     = utils.Hash256String(r.Header.Get("User-Agent"))

		// Write asynchronously — audit log must never slow down the response
		go func(ev *AuditEvent) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := al.write(ctx, ev); err != nil {
				// Log to stderr only — do not panic or affect the request
				fmt.Printf("audit log write failed: %v\n", err)
			}
		}(event)
	})
}

// =========================================================================
// write — private DB insert
// =========================================================================

// write inserts one audit event into the audit_logs table.
// Uses al.db.Pool().Exec — Pool() is the correct way to access pgx
// through the db.DB wrapper. al.db.Exec does not exist.
func (al *AuditLogger) write(ctx context.Context, event *AuditEvent) error {
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("audit log metadata marshal failed: %w", err)
	}

	// Use NULL for empty UserID — audit_logs.user_id is nullable
	var userID *string
	if event.UserID != "" {
		userID = &event.UserID
	}

	_, err = al.db.Pool().Exec(ctx,
		`INSERT INTO audit_logs (user_id, event_type, severity, metadata, created_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		userID,
		event.EventType,
		event.Severity,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("audit log insert failed: %w", err)
	}
	return nil
}

// =========================================================================
// Context helpers
// =========================================================================

// SetAuditEvent injects an AuditEvent into the request context.
// Called by handlers to signal what happened — audit_logger reads it after
// the handler returns.
func SetAuditEvent(ctx context.Context, event *AuditEvent) context.Context {
	return context.WithValue(ctx, contextKeyAuditEvent, event)
}

// AuditEventFromContext retrieves the AuditEvent from context.
// Called by the Log middleware after the handler completes.
func AuditEventFromContext(ctx context.Context) (*AuditEvent, bool) {
	ev, ok := ctx.Value(contextKeyAuditEvent).(*AuditEvent)
	return ev, ok
}

// =========================================================================
// DirectLog — synchronous write for critical security events
// =========================================================================

// DirectLog writes an audit event synchronously — used when the middleware
// context pattern is not available, for example inside token theft detection
// in tokens/refresh_token.go or repeated invalidation detection.
// Critical security events must not be lost to async goroutine failures.
func (al *AuditLogger) DirectLog(ctx context.Context, event *AuditEvent) error {
	return al.write(ctx, event)
}