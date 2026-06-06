package tokens

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"student_portal/backend/config"
	"student_portal/backend/internals/utils"
)

// MagicToken represents a stored token row
type MagicToken struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	TokenHash string    `json:"token_hash" db:"token_hash"`
	UAHash    string    `json:"ua_hash" db:"ua_hash"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	IsUsed    bool      `json:"is_used" db:"is_used"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MagicTokenResults represents the generated token before storage
type MagicTokenResults struct {
	UserID    string    `json:"user_id"`
	RawToken  string    `json:"raw_token"`
	TokenHash string    `json:"token_hash"`
	UAHash    string    `json:"ua_hash"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Generate creates a new magic token
func Generate(r *http.Request, cfg *config.AppConfig) (*MagicTokenResults, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("magic token generation failed: %w", err)
	}
	rawToken := hex.EncodeToString(bytes)

	tokenHash := utils.Hash256String(rawToken)
	fp := utils.ExtractFingerprints(r) // corrected name
	uaHash := utils.Hash256String(fp.UserAgent)

	expiresAt := time.Now().UTC().Add(time.Duration(cfg.Auth.MAGIC_LINK_EXPIRY_MINUTES) * time.Minute)

	return &MagicTokenResults{
		RawToken:  rawToken,
		TokenHash: tokenHash,
		UAHash:    uaHash,
		ExpiresAt: expiresAt,
	}, nil
}

// Store inserts a new magic token row
func Store(ctx context.Context, pool *pgxpool.Pool, userID string, result *MagicTokenResults) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO magic_tokens (user_id, token_hash, user_agent_hash, expires_at, is_used)
         VALUES ($1, $2, $3, $4, false)`,
		userID, result.TokenHash, result.UAHash, result.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store magic token for user %s: %w", userID, err)
	}
	return nil
}

// Verify checks a raw token against DB and marks it used
func Verify(ctx context.Context, pool *pgxpool.Pool, rawToken string, r *http.Request) (*MagicTokenResults, error) {
	tokenHash := utils.Hash256String(rawToken)

	// Query single row
	row := pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, user_agent_hash, expires_at, is_used, created_at
         FROM magic_tokens WHERE token_hash = $1`, tokenHash)

	var token MagicToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.UAHash,
		&token.ExpiresAt, &token.IsUsed, &token.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("magic token verification failed: %w", err)
	}

	// Validation checks
	if token.IsUsed {
		return nil, ErrTokenAlreadyUsed
	}
	if token.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrTokenExpired
	}
	if token.UAHash != utils.Hash256String(utils.ExtractFingerprints(r).UserAgent) {
		return nil, ErrUAMismatch
	}

	// Mark token as used
	ct, err := pool.Exec(ctx,
		`UPDATE magic_tokens SET is_used = TRUE WHERE id = $1 AND is_used = FALSE`, token.ID)
	if err != nil {
		return nil, fmt.Errorf("magic token verification failed: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrTokenAlreadyUsed // race condition
	}

	return &MagicTokenResults{
		UserID:    token.UserID,
		RawToken:  rawToken,
		TokenHash: token.TokenHash,
		UAHash:    token.UAHash,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

// BuildMagicLinkURL constructs the link
func BuildMagicLinkURL(baseURL string, rawToken string) string {
	return fmt.Sprintf("%s?token=%s", baseURL, rawToken)
}

// Common errors
var (
	ErrTokenAlreadyUsed = fmt.Errorf("magic token has already been used")
	ErrTokenExpired     = fmt.Errorf("magic token has expired")
	ErrUAMismatch       = fmt.Errorf("user agent does not match token binding")
	ErrTokenNotFound    = fmt.Errorf("magic token not found")
)
