package tokens 

import (
	"context"
	"time"
)


type DeviceToken struct {
	ID                string    `json:"id" db:"id"`
	UserID            string    `json:"user_id" db:"user_id"`
	DeviceTokenHash   string    `json:"device_token_hash" db:"device_token_hash"`
	FingerprintHash   string    `json:"fingerprint_hash" db:"fingerprint_hash"`
	IsRevoked         bool      `json:"is_revoked" db:"is_revoked"`
	ExpiresAt         time.Time `json:"expires_at" db:"expires_at"`
	LastUsedAt        time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

func Issue(ctx context.Context, pool *pgxpool.Pool, userID string, r *http.Request, cfg *config.AuthConfig) (string, error) {
	// Generate new device token
	deviceTokenHash := utils.Hash256String(utils.GenerateRandomString(32))
	fp := utils.ExtractFingerprint(r)
	fingerprintHash := utils.Hash256String(fp.Fingerprint)

	// INSERT INTO device_trust
// (user_id, device_token_hash, fingerprint_hash, is_revoked, expires_at, last_used_at)
// VALUES ($1, $2, $3, FALSE, $4, NOW())
// ON CONFLICT (user_id, fingerprint_hash) WHERE is_revoked = FALSE
// DO UPDATE SET
//     device_token_hash = EXCLUDED.device_token_hash,
//     expires_at = EXCLUDED.expires_at,
//     last_used_at = NOW()

	// Query single row
	row := pool.QueryRow(ctx,
		`INSERT INTO device_trust (user_id, device_token_hash, fingerprint_hash, is_revoked, expires_at, last_used_at)
		 VALUES ($1, $2, $3, FALSE, $4, NOW())
		 ON CONFLICT (user_id, fingerprint_hash) WHERE is_revoked = FALSE
		 DO UPDATE SET
		     device_token_hash = EXCLUDED.device_token_hash,
		     expires_at = EXCLUDED.expires_at,
		     last_used_at = NOW()
		 RETURNING device_token_hash`,
		userID, deviceTokenHash, fingerprintHash, expiresAt,
	)

	var deviceTokenHash string
	if err := row.Scan(&deviceTokenHash); err != nil {
		return "", fmt.Errorf("device token issue failed: %w", err)
	}

	return deviceTokenHash, nil
}

func SetCookie(w http.ResponseWriter, rawToken string, cfg *config.AuthConfig) {
	http.SetCookie(w, &http.Cookie{
    Name:     "device_token",
    Value:    rawToken,
    Path:     "/",
    HttpOnly: true,                              // JS cannot read this
    Secure:   true,                              // HTTPS only
    SameSite: http.SameSiteStrictMode,           // no cross-site sending
    MaxAge:   cfg.DeviceTokenExpiryDays * 86400, // seconds
	})

}

func Validate(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (*DeviceToken, error) {
	cookie, err := r.Cookie("device_token")
	if err != nil {
		return nil, ErrDeviceTokenMissing
	}

	deviceTokenHash := utils.Hash256String(cookie.Value)
	// SELECT * FROM device_trust
// WHERE device_token_hash=$1 AND is_revoked=FALSE AND expires_at > NOW()

	// Query single row
	row := pool.QueryRow(ctx,
		`SELECT id, user_id, device_token_hash, fingerprint_hash, is_revoked, expires_at, last_used_at, created_at
		 FROM device_trust WHERE device_token_hash = $1 AND is_revoked=FALSE AND expires_at > NOW()`,
		deviceTokenHash,
	)
	if !utils.FingerprintMatch(r, fingerprintHash) {
		return nil, ErrFingerprintChanged
	}
	// UPDATE device_trust SET last_used_at=NOW() WHERE id=$1

	// Query single row
	row := pool.QueryRow(ctx,
		`UPDATE device_trust SET last_used_at=NOW() WHERE id=$1
		 RETURNING id, user_id, device_token_hash, fingerprint_hash, is_revoked, expires_at, last_used_at, created_at`,
		token.ID,
	)
	return &DeviceToken{},nil
}

func Revoke(ctx context.Context, pool *pgxpool.Pool, deviceTokenHash string) error {
	_, err := pool.Exec(ctx,
		"UPDATE device_trust SET is_revoked=TRUE WHERE device_token_hash=$1 AND is_revoked=FALSE",
		deviceTokenHash,
	)
	return err
}

func RevokeAllUsers(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx,
		"UPDATE device_trust SET is_revoked=TRUE WHERE user_id=$1 AND is_revoked=FALSE",
		userID,
	)
	return err
}

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
    Name:     "device_token",
    MaxAge:   -1,     // instructs browser to delete immediately
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
})
}

var (
    ErrDeviceTokenMissing        = fmt.Errorf("device token cookie not present")
    ErrDeviceTokenInvalid        = fmt.Errorf("device token is invalid or expired")
    ErrDeviceFingerprintChanged  = fmt.Errorf("device fingerprint has changed")
)