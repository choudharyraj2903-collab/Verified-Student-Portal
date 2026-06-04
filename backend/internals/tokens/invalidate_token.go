package tokens 

import (
	"context"
	"time"
)


type InvalidationToken struct {
	ID                    string    `json:"id" db:"id"`
	UserID                string    `json:"user_id" db:"user_id"`
	RefreshTokenFamily    string    `json:"refresh_token_family" db:"refresh_token_family"`
	TokenHash             string    `json:"token_hash" db:"token_hash"`
	IsUsed                bool      `json:"is_used" db:"is_used"`
	ExpiresAt             time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

func Issue(ctx context.Context, pool *pgxpool.Pool, userID string, familyID string, cfg *config.AuthConfig) (string, error) {

	// Generate new invalidation token
	tokenHash := utils.Hash256String(utils.GenerateRandomString(32))
	expiresAT := time.Now().UTC().Add(time.Duration(cfg.InvalidationTokenExpiryMinutes) * time.Minute)

	// Query single row
	row := pool.QueryRow(ctx,
		`INSERT INTO invalidation_tokens (user_id, token_hash, refresh_token_family, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, token_hash, refresh_token_family, expires_at, created_at`,
		userID, tokenHash, familyID, expiresAT,
	)

	//— Return raw token Caller builds the "No, this was not me" URL:
	var token InvalidationToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.RefreshTokenFamily, &token.ExpiresAt, &token.CreatedAt); err != nil {
		return "", fmt.Errorf("invalidation token issue failed: %w", err)
	}

	return token.TokenHash
	
}

func BuildInvalidationURL(baseURL string, rawToken string) string {
    return fmt.Sprintf("%s/session/invalidate?token=%s", baseURL, rawToken)
}

// BuildConfirmationURL builds the "Yes, this was me" link for session confirmation
func BuildConfirmationURL(baseURL string, rawToken string) string {
    return fmt.Sprintf("%s/session/confirm?token=%s", baseURL, rawToken)
}

func Consume(ctx context.Context, pool *pgxpool.Pool, rawToken string) (*InvalidationToken, error) {
	hash := utils.Hash256String(rawToken)

	// Query single row
	row := pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, refresh_token_family, expires_at, created_at
		 FROM invalidation_tokens WHERE token_hash = $1`, hash)

	var token InvalidationToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.RefreshTokenFamily, &token.ExpiresAt, &token.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("invalidation token consumption failed: %w", err)
	}

	return &token, nil
}

func CheckRepeatedInvalidations(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {

	// Query single row
	row := pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, refresh_token_family, expires_at, created_at
		 FROM invalidation_tokens WHERE user_id = $1 AND is_used = FALSE`, userID)

	var token InvalidationToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.RefreshTokenFamily, &token.ExpiresAt, &token.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("invalidation token consumption failed: %w", err)
	}

	return true, nil
}

var (
    ErrInvalidationTokenNotFound = fmt.Errorf("invalidation token not found or expired")
    ErrInvalidationTokenUsed     = fmt.Errorf("invalidation token has already been used")
)