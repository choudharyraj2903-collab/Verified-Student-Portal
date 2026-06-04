package tokens 

import (
	"context"
	"time"
)


 type RefreshToken struct {
	 ID               string    `json:"id" db:"id"`
	 UserID           string    `json:"user_id" db:"user_id"`
	 TokenHash        string    `json:"token_hash" db:"token_hash"`
	 FamilyID         string    `json:"family_id" db:"family_id"`
	 FingerprintHash  string    `json:"fingerprint_hash" db:"fingerprint_hash"`
	 IsRevoked        bool      `json:"is_revoked" db:"is_revoked"`
	 ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	 CreatedAt        time.Time `json:"created_at" db:"created_at"`
 }

 RefreshTokenResult
 ├── RawToken     string    — set in HttpOnly cookie, never stored
 ├── TokenHash    string    — stored in DB
 ├── FamilyID     string    — stored in DB, passed forward on rotation
 └── ExpiresAt    time.Time

 type RefreshTokenResult struct {
	 RawToken     string    `json:"raw_token"`
	 TokenHash    string    `json:"token_hash"`
	 FamilyID     string    `json:"family_id"`
	 ExpiresAt    time.Time `json:"expires_at"`
 }

func Issue(ctx context.Context, pool *pgxpool.Pool, userID string, familyID string, r *http.Request, cfg *config.AuthConfig) (*RefreshTokenResult, error) {
	 // Generate new refresh token
	tokenHash := utils.Hash256String(utils.GenerateRandomString(32))
     fp := utils.ExtractFingerprint(r)
	 fingerprintHash := utils.Hash256String(fp.Fingerprint)

	 expiresAt := time.Now().UTC().Add(time.Duration(cfg.RefreshTokenExpiryMinutes) * time.Minute)

	 // Query single row
	 row := pool.QueryRow(ctx,
		 `INSERT INTO refresh_tokens (user_id, token_hash, family_id, fingerprint_hash,is_revoked, expires_at)
		  VALUES ($1, $2, $3, $4,False,$5)
		  RETURNING id, user_id, token_hash, family_id, fingerprint_hash, expires_at, created_at`,
		 userID, tokenHash, familyID, fingerprintHash, expiresAt,
	 )

	 var token RefreshToken
	 if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.FamilyID, &token.FingerprintHash, &token.ExpiresAt, &token.CreatedAt); err != nil {
		 return nil, fmt.Errorf("refresh token issue failed: %w", err)
	 }

	 return &RefreshTokenResult{},nil
 }

func Rotate(ctx context.Context, pool *pgxpool.Pool, rawToken string, r *http.Request, cfg *config.AuthConfig) (*RefreshTokenResult, error) {
	incomingHash := utils.Hash256String(rawToken)
	// Query single row
	row := pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, family_id, fingerprint_hash, expires_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, incomingHash)

	var token RefreshToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.FamilyID, &token.FingerprintHash, &token.ExpiresAt, &token.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("refresh token rotation failed: %w", err)
	}
	if time.Now().UTC().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	if utils.FingerprintMatch(r, token.FingerprintHash) {
		return nil, ErrFingerprintChanged
	}
	err := db.WithTransaction(ctx, func(tx pgx.Tx) error {
    // Revoke the old token
    _, err := tx.Exec(ctx,
        "UPDATE refresh_tokens SET is_revoked=TRUE WHERE id=$1 AND is_revoked=FALSE",
        token.ID,
    )
    if err != nil { return err }

    // Issue new token with same family ID
    newResult, err = Issue(ctx, pool, token.UserID, token.FamilyID, r, cfg)
    return err
	})
	if err != nil {
		return nil, fmt.Errorf("refresh token rotation failed: %w", err)
	}
	return newResult, nil
}

func RevokeFamily(ctx context.Context, pool *pgxpool.Pool, familyID string) error {
	_, err := pool.Exec(ctx,
		"UPDATE refresh_tokens SET is_revoked=TRUE WHERE family_id=$1 AND is_revoked=FALSE",
		familyID,
	)
	return err
}

var (
    ErrRefreshTokenNotFound  = fmt.Errorf("refresh token not found")
    ErrRefreshTokenExpired   = fmt.Errorf("refresh token has expired")
    ErrRefreshTokenReuse     = fmt.Errorf("refresh token reuse detected — session terminated")
    ErrFingerprintMismatch   = fmt.Errorf("device fingerprint does not match token binding")
)

