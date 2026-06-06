package tokens

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"student_portal/backend/config"
	"student_portal/backend/internals/utils"
)

// DeviceToken represents a trusted device entry in the database
type DeviceToken struct {
	ID              string    `json:"id"                db:"id"`
	UserID          string    `json:"user_id"           db:"user_id"`
	DeviceTokenHash string    `json:"device_token_hash" db:"device_token_hash"`
	FingerprintHash string    `json:"fingerprint_hash"  db:"fingerprint_hash"`
	IsRevoked       bool      `json:"is_revoked"        db:"is_revoked"`
	ExpiresAt       time.Time `json:"expires_at"        db:"expires_at"`
	LastUsedAt      time.Time `json:"last_used_at"      db:"last_used_at"`
	CreatedAt       time.Time `json:"created_at"        db:"created_at"`
}

var (
	ErrDeviceTokenMissing       = fmt.Errorf("device token cookie not present")
	ErrDeviceTokenInvalid       = fmt.Errorf("device token is invalid or expired")
	ErrDeviceFingerprintChanged = fmt.Errorf("device fingerprint has changed")
)

// IssueDeviceToken generates a new device trust token, upserts its hash into
// the database, and returns the raw token to be set as an HttpOnly cookie.
// If the same browser fingerprint already has an active trust entry it is
// refreshed rather than creating a duplicate row.
func IssueDeviceToken(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID string,
	r *http.Request,
	cfg *config.AuthConfig,
) (string, error) {
	// Step 1 — generate 32 random bytes and encode to hex
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", fmt.Errorf("device token generation failed: %w", err)
	}
	rawToken := hex.EncodeToString(rawBytes)

	// Step 2 — hash raw token — only the hash is stored in DB
	deviceTokenHash := utils.Hash256String(rawToken)

	// Step 3 — build fingerprint hash from request headers
	fipr := utils.ExtractFingerprints(r)
	fipr_string := utils.BuildFingerprintString(fipr)
	fingerprintHash := utils.HashFingerprint(fipr_string)

	// Step 4 — calculate expiry — was missing entirely causing undefined: expiresAt
	expiresAt := time.Now().UTC().Add(
		time.Duration(cfg.DEVICE_TOKEN_EXPIRY_DAYS) * 24 * time.Hour,
	)

	// Step 5 — upsert into device_trust
	// ON CONFLICT targets the partial unique index on (user_id, fingerprint_hash)
	// WHERE is_revoked = FALSE so a previously revoked device can be re-trusted
	_, err := pool.Exec(ctx,
		`INSERT INTO device_trust
			(user_id, device_token_hash, fingerprint_hash, is_revoked, expires_at, last_used_at)
		 VALUES ($1, $2, $3, FALSE, $4, NOW())
		 ON CONFLICT (user_id, fingerprint_hash) WHERE is_revoked = FALSE
		 DO UPDATE SET
		 	device_token_hash = EXCLUDED.device_token_hash,
		 	expires_at        = EXCLUDED.expires_at,
		 	last_used_at      = NOW()`,
		userID, deviceTokenHash, fingerprintHash, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("device token upsert failed: %w", err)
	}

	// Return raw token — caller writes this into the HttpOnly cookie
	// DB stores only the hash — never the raw value
	return rawToken, nil
}

// SetCookie writes the device trust token into a secure HttpOnly cookie.
// All cookie security attributes are enforced here — nothing else in the
// codebase sets this cookie.
func SetCookie(w http.ResponseWriter, rawToken string, cfg *config.AuthConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     "device_token",
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   cfg.DEVICE_TOKEN_EXPIRY_DAYS * 86400,
	})
}

// Validate reads the device token cookie, verifies it against the database,
// checks the fingerprint still matches the stored one, updates last_used_at,
// and returns the full DeviceToken struct on success.
func Validate(
	ctx context.Context,
	pool *pgxpool.Pool,
	r *http.Request,
) (*DeviceToken, error) {
	// Step 1 — read cookie from request
	cookie, err := r.Cookie("device_token")
	if err != nil {
		return nil, ErrDeviceTokenMissing
	}

	// Step 2 — hash the cookie value for DB lookup
	// Never query by raw token — always by hash
	deviceTokenHash := utils.Hash256String(cookie.Value)

	// Step 3 — fetch token record from DB
	// was: row declared but never scanned into a struct — undefined: token later
	var token DeviceToken
	err = pool.QueryRow(ctx,
		`SELECT id, user_id, device_token_hash, fingerprint_hash,
		        is_revoked, expires_at, last_used_at, created_at
		 FROM device_trust
		 WHERE device_token_hash = $1
		   AND is_revoked = FALSE
		   AND expires_at > NOW()`,
		deviceTokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.DeviceTokenHash,
		&token.FingerprintHash,
		&token.IsRevoked,
		&token.ExpiresAt,
		&token.LastUsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, ErrDeviceTokenInvalid
	}

	// Step 4 — verify fingerprint matches what was stored at trust time
	// Uses the stored hash from the DB row — not a freshly computed one
	// A browser update or different browser on the same laptop fails here
	// That is expected — they just need to log in again
	if !utils.FingerprintMatch(r, token.FingerprintHash) {
		return nil, ErrDeviceFingerprintChanged
	}

	// Step 5 — update last_used_at passively
	// Non-fatal if this fails — do not block the request over a timestamp update
	_, _ = pool.Exec(ctx,
		`UPDATE device_trust SET last_used_at = NOW() WHERE id = $1`,
		token.ID,
	)

	return &token, nil
}

// Revoke marks a single device trust entry as revoked by its token hash.
// Called on logout and session invalidation.
// Always paired with ClearCookie on the response — server side and client
// side must both be cleared together.
func Revoke(
	ctx context.Context,
	pool *pgxpool.Pool,
	deviceTokenHash string,
) error {
	_, err := pool.Exec(ctx,
		`UPDATE device_trust
		 SET is_revoked = TRUE
		 WHERE device_token_hash = $1
		   AND is_revoked = FALSE`,
		deviceTokenHash,
	)
	if err != nil {
		return fmt.Errorf("device token revoke failed: %w", err)
	}
	return nil
}

// RevokeAllForUser marks every active device trust entry for a user as revoked.
// Called on logout-all-devices and account deactivation.
func RevokeAllForUser(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID string,
) error {
	_, err := pool.Exec(ctx,
		`UPDATE device_trust
		 SET is_revoked = TRUE
		 WHERE user_id = $1
		   AND is_revoked = FALSE`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all device tokens failed for user %s: %w", userID, err)
	}
	return nil
}

// ClearCookie instructs the browser to immediately delete the device token
// cookie. Must always be paired with a server-side Revoke call —
// clearing the cookie alone does not invalidate the server-side DB record.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "device_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}