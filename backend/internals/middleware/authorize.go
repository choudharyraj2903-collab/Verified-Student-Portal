package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"student_portal/backend/config"
	"student_portal/backend/internals/tokens"
	"student_portal/backend/internals/utils"
)

// =========================================================================
// Structs
// =========================================================================

type Authorizer struct {
	pool        *pgxpool.Pool
	appCfg      *config.AppConfig
	auditLogger *AuditLogger
}

const contextKeyCouncil contextKey = "council"

// =========================================================================
// Constructor
// =========================================================================

// NewAuthorizer now takes pool and appCfg — both are needed by middleware
// functions that call tokens.Verify and utils.SendValidationError.
func NewAuthorizer(pool *pgxpool.Pool, appCfg *config.AppConfig, al *AuditLogger) *Authorizer {
	return &Authorizer{
		pool:        pool,
		appCfg:      appCfg,
		auditLogger: al,
	}
}

// =========================================================================
// RequireRole
// =========================================================================

func (az *Authorizer) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				utils.SendUnauthorized(w)
				return
			}

			allowed := false
			for _, role := range roles {
				if user.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				_ = az.auditLogger.DirectLog(r.Context(), &AuditEvent{
					EventType: "UNAUTHORIZED_SCOPE_ACCESS",
					Severity:  "WARN",
					UserID:    user.ID,
					Metadata: map[string]any{
						"reason":         "insufficient_role",
						"required_roles": roles,
						"actual_role":    user.Role,
						"endpoint":       r.URL.Path,
						"method":         r.Method,
					},
				})
				utils.SendForbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// =========================================================================
// RequireCouncilScope
// =========================================================================

func (az *Authorizer) RequireCouncilScope(councilParamKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				utils.SendUnauthorized(w)
				return
			}

			// Super Admin has access to all councils — skip scope check
			if user.Role == "SUPER_ADMIN" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract and normalize council code from URL path
			councilCode := strings.ToUpper(strings.TrimSpace(r.PathValue(councilParamKey)))

			// Validate against known council codes — never trust URL params blindly
			validCodes := map[string]bool{
				"MNC": true,
				"SNT": true,
				"GNS": true,
				"ANC": true,
			}
			if !validCodes[councilCode] {
				// SendValidationError needs appCfg — fixes "not enough arguments" error
				utils.SendValidationError(w, "invalid council code", az.appCfg)
				return
			}

			// Check user has scope for this specific council
			hasScope := false
			for _, code := range user.CouncilCodes {
				if code == councilCode {
					hasScope = true
					break
				}
			}

			if !hasScope {
				_ = az.auditLogger.DirectLog(r.Context(), &AuditEvent{
					EventType: "UNAUTHORIZED_SCOPE_ACCESS",
					Severity:  "CRITICAL",
					UserID:    user.ID,
					Metadata: map[string]any{
						"reason":      "missing_council_scope",
						"council":     councilCode,
						"user_scopes": user.CouncilCodes,
						"endpoint":    r.URL.Path,
						"method":      r.Method,
					},
				})
				utils.SendForbidden(w)
				return
			}

			// Inject validated council code into context
			ctx := context.WithValue(r.Context(), contextKeyCouncil, councilCode)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// =========================================================================
// InjectCouncilScope
// =========================================================================

func (az *Authorizer) InjectCouncilScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyCouncil, user.CouncilCodes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// =========================================================================
// RequireCompleteProfile
// =========================================================================

func (az *Authorizer) RequireCompleteProfile(profileChecker ProfileChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				utils.SendUnauthorized(w)
				return
			}

			// Admins do not need a complete profile to act
			if user.Role == "COUNCIL_ADMIN" || user.Role == "SUPER_ADMIN" {
				next.ServeHTTP(w, r)
				return
			}

			complete, err := profileChecker.IsProfileComplete(r.Context(), user.ID)
			if err != nil {
				utils.SendInternalError(w, err, az.appCfg)
				return
			}
			if !complete {
				utils.SendError(w, http.StatusForbidden,
					"complete your profile before submitting a verification request",
					"PROFILE_INCOMPLETE",
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// =========================================================================
// RequireReAuth
// =========================================================================

func (az *Authorizer) RequireReAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken := r.Header.Get("X-ReAuth-Token")
		if rawToken == "" {
			utils.SendError(w, http.StatusForbidden,
				"re-authentication required for this action",
				"REAUTH_REQUIRED",
			)
			return
		}

		// tokens.Verify needs pool as second argument — fixes "not enough arguments" error
		_, err := tokens.Verify(r.Context(), az.pool, rawToken, r)
		if err != nil {
			utils.SendError(w, http.StatusForbidden,
				"re-authentication required for this action",
				"REAUTH_REQUIRED",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// =========================================================================
// Context helpers
// =========================================================================

func CouncilFromContext(ctx context.Context) (string, bool) {
	code, ok := ctx.Value(contextKeyCouncil).(string)
	return code, ok
}

func CouncilCodesFromContext(ctx context.Context) ([]string, bool) {
	codes, ok := ctx.Value(contextKeyCouncil).([]string)
	return codes, ok
}

// =========================================================================
// ProfileChecker interface
// =========================================================================

type ProfileChecker interface {
	IsProfileComplete(ctx context.Context, userID string) (bool, error)
}