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

type RefreshToken struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	TokenHash       string    `json:"token_hash"`
	FamilyID        string    `json:"family_id"`
	FingerprintHash string    `json:"fingerprint_hash"`
	IsRevoked       bool      `json:"is_revoked"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type RefreshTokenResult struct {
	UserID    string    `json:"user_id"`
	RawToken  string    `json:"raw_token"`
	TokenHash string    `json:"token_hash"`
	FamilyID  string    `json:"family_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func IssueRefreshToken(ctx context.Context, pool *pgxpool.Pool, userID string, familyID string, r *http.Request, cfg *config.AuthConfig) (*RefreshTokenResult, error) {
	rawToken, err := generateSecureRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := utils.Hash256String(rawToken)
	expiresAt := time.Now().UTC().Add(time.Duration(cfg.REFRESH_TOKEN_EXPIRY_DAYS) * 24 * time.Hour)

	// Build fingerprint from the current request (the verify request)
	fp := utils.ExtractFingerprints(r)
	fingerprintHash := utils.HashFingerprint(utils.BuildFingerprintString(fp))

	_, err = pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, family_id, fingerprint_hash, is_revoked, expires_at)
		 VALUES ($1, $2, $3, $4, FALSE, $5)`,
		userID, tokenHash, familyID, fingerprintHash, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &RefreshTokenResult{
		UserID:    userID,
		RawToken:  rawToken,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		ExpiresAt: expiresAt,
	}, nil
}

func RotateRefreshToken(ctx context.Context, pool *pgxpool.Pool, rawToken string, r *http.Request, cfg *config.AuthConfig) (*RefreshTokenResult, error) {
	tokenHash := utils.Hash256String(rawToken)

	var rt RefreshToken
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, family_id, is_revoked, expires_at FROM refresh_tokens
		 WHERE token_hash = $1 AND is_revoked = FALSE AND expires_at > NOW()`,
		tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.FamilyID, &rt.IsRevoked, &rt.ExpiresAt)
	if err != nil {
		return nil, ErrRefreshTokenNotFound
	}

	// Revoke old token
	_, err = pool.Exec(ctx, `UPDATE refresh_tokens SET is_revoked = TRUE WHERE id = $1`, rt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Issue new token in same family
	newRaw, err := generateSecureRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	newHash := utils.Hash256String(newRaw)
	newExpiry := time.Now().UTC().Add(time.Duration(cfg.REFRESH_TOKEN_EXPIRY_DAYS) * 24 * time.Hour)
	fp := utils.ExtractFingerprints(r)
	fingerprintHash := utils.HashFingerprint(utils.BuildFingerprintString(fp))

	_, err = pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, family_id, fingerprint_hash, is_revoked, expires_at)
		 VALUES ($1, $2, $3, $4, FALSE, $5)`,
		rt.UserID, newHash, rt.FamilyID, fingerprintHash, newExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store rotated refresh token: %w", err)
	}

	return &RefreshTokenResult{
		UserID:    rt.UserID,
		RawToken:  newRaw,
		TokenHash: newHash,
		FamilyID:  rt.FamilyID,
		ExpiresAt: newExpiry,
	}, nil
}

func RevokeFamily(ctx context.Context, pool *pgxpool.Pool, familyID string) error {
	_, err := pool.Exec(ctx, `UPDATE refresh_tokens SET is_revoked = TRUE WHERE family_id = $1`, familyID)
	return err
}

func generateSecureRandomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

var (
	ErrRefreshTokenNotFound = fmt.Errorf("refresh token not found")
	ErrRefreshTokenExpired  = fmt.Errorf("refresh token has expired")
	ErrFingerprintChanged   = fmt.Errorf("device fingerprint does not match token binding")
)
