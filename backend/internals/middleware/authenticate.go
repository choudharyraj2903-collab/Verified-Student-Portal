package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"student_portal/backend/config"
	"student_portal/backend/db"
	"student_portal/backend/internals/tokens"
	"student_portal/backend/internals/utils"
)

// =========================================================================
// Structs
// =========================================================================

type Authenticator struct {
	db          *db.DB
	cfg         *config.AppConfig
	auditLogger *AuditLogger
}

// AuthenticatedUser is injected into request context after successful auth.
// Every protected handler reads from this — never re-parses the JWT.
type AuthenticatedUser struct {
	ID           string
	Email        string
	Role         string
	CouncilCodes []string
	IsActive     bool
}

// =========================================================================
// Constructor
// =========================================================================

// NewAuthenticator takes the full AppConfig — not just JWTConfig —
// because SendInternalError needs cfg and audit logging needs app context.
func NewAuthenticator(database *db.DB, cfg *config.AppConfig, al *AuditLogger) *Authenticator {
	return &Authenticator{
		db:          database,
		cfg:         cfg,
		auditLogger: al,
	}
}

// =========================================================================
// Middleware
// =========================================================================

func (a *Authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Step 1 — Extract Bearer token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.SendUnauthorized(w)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Step 2 — Verify JWT signature, algorithm, expiry, issuer
		claims, err := tokens.VerifyAccessToken(tokenString, &a.cfg.JWT)
		if err != nil {
			if errors.Is(err, tokens.ErrAccessTokenExpired) {
				// Return specific code so frontend knows to attempt refresh
				// rather than redirect to login
				utils.SendError(w, http.StatusUnauthorized, "token expired", "TOKEN_EXPIRED")
				return
			}
			utils.SendUnauthorized(w)
			return
		}

		// Step 3 — Re-validate user is still active in DB
		// JWT is stateless — it cannot know if user was deactivated after issuance
		user, err := a.getUserFromDB(r.Context(), claims.Subject)
		if err != nil {
			utils.SendUnauthorized(w)
			return
		}
		if !user.IsActive {
			utils.SendUnauthorized(w)
			return
		}

		// Step 4 — Re-validate role from DB
		// If role changed since token was issued, reject
		// Log as WARN — could indicate token tampering attempt
		if user.Role != claims.Role {
			_ = a.auditLogger.DirectLog(r.Context(), &AuditEvent{
				EventType: "UNAUTHORIZED_SCOPE_ACCESS",
				Severity:  "WARN",
				UserID:    user.ID,
				Metadata: map[string]any{
					"reason":      "role_mismatch",
					"token_role":  claims.Role,
					"db_role":     user.Role,
				},
			})
			utils.SendUnauthorized(w)
			return
		}

		// Step 5 — Re-validate council scopes from DB
		// A revoked scope must take effect immediately — not wait for token expiry
		dbScopes, err := a.getCouncilScopesFromDB(r.Context(), user.ID)
		if err != nil {
			utils.SendInternalError(w, err, a.cfg)
			return
		}

		// If token claims scopes that DB no longer has — flag and reject
		if !scopesMatch(claims.CouncilCodes, dbScopes) {
			_ = a.auditLogger.DirectLog(r.Context(), &AuditEvent{
				EventType: "UNAUTHORIZED_SCOPE_ACCESS",
				Severity:  "CRITICAL",
				UserID:    user.ID,
				Metadata: map[string]any{
					"reason":       "scope_mismatch",
					"token_scopes": claims.CouncilCodes,
					"db_scopes":    dbScopes,
				},
			})
			utils.SendUnauthorized(w)
			return
		}

		// Step 6 — Verify device fingerprint against stored refresh token
		// Fetch the fingerprint hash from the most recent active refresh token
		// for this user and compare against current request headers
		storedHash, err := a.getActiveFingerprintFromDB(r.Context(), user.ID)
		if err == nil && storedHash != "" {
			// Only check if a stored hash exists
			// If no refresh token found (e.g. fresh session edge case) — skip check
			if !utils.FingerprintMatch(r, storedHash) {
				utils.SendUnauthorized(w)
				return
			}
		}

		// Step 7 — Inject authenticated user into context
		// All downstream handlers read from context — never re-parse the JWT
		authedUser := &AuthenticatedUser{
			ID:           user.ID,
			Email:        user.Email,
			Role:         user.Role,
			CouncilCodes: dbScopes,
			IsActive:     user.IsActive,
		}
		ctx := context.WithValue(r.Context(), contextKeyUser, authedUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// =========================================================================
// Context helper — exported so every handler can call it
// =========================================================================

func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	u, ok := ctx.Value(contextKeyUser).(*AuthenticatedUser)
	return u, ok
}

// =========================================================================
// Private DB helpers
// =========================================================================

// getUserFromDB fetches the minimal user record needed for auth validation.
// Single indexed query on users.id — fast enough to run on every request.
func (a *Authenticator) getUserFromDB(ctx context.Context, userID string) (*AuthenticatedUser, error) {
	var u AuthenticatedUser
	err := a.db.Pool().QueryRow(ctx,
		`SELECT id, email, role, is_active
		 FROM users
		 WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.Role, &u.IsActive)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// getCouncilScopesFromDB returns all active council codes for the user.
// Used to verify JWT claims match current DB state on every request.
func (a *Authenticator) getCouncilScopesFromDB(ctx context.Context, userID string) ([]string, error) {
	rows, err := a.db.Pool().Query(ctx,
		`SELECT c.code
		 FROM councils c
		 JOIN user_council_scopes ucs ON c.id = ucs.council_id
		 WHERE ucs.user_id = $1
		   AND ucs.is_active = TRUE
		   AND (ucs.expires_at IS NULL OR ucs.expires_at > NOW())`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		scopes = append(scopes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return scopes, nil
}

// getActiveFingerprintFromDB fetches the fingerprint hash from the most recent
// non-revoked refresh token for this user.
// This is how we know what device fingerprint the session was created on.
func (a *Authenticator) getActiveFingerprintFromDB(ctx context.Context, userID string) (string, error) {
	var fingerprintHash string
	err := a.db.Pool().QueryRow(ctx,
		`SELECT fingerprint_hash
		 FROM refresh_tokens
		 WHERE user_id = $1
		   AND is_revoked = FALSE
		   AND expires_at > NOW()
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&fingerprintHash)
	if err != nil {
		// No active refresh token found — not a hard failure
		// Session may have just been created or token just rotated
		return "", nil
	}
	return fingerprintHash, nil
}

// =========================================================================
// Private helpers
// =========================================================================

// scopesMatch checks that every code in token claims exists in DB scopes
// and vice versa. Order does not matter — both directions checked.
func scopesMatch(tokenScopes, dbScopes []string) bool {
	if len(tokenScopes) != len(dbScopes) {
		return false
	}
	dbSet := make(map[string]bool, len(dbScopes))
	for _, s := range dbScopes {
		dbSet[s] = true
	}
	for _, s := range tokenScopes {
		if !dbSet[s] {
			return false
		}
	}
	return true
}