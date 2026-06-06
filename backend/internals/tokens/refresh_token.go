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
    ID              string    `json:"id" db:"id"`
    UserID          string    `json:"user_id" db:"user_id"`
    TokenHash       string    `json:"token_hash" db:"token_hash"`
    FamilyID        string    `json:"family_id" db:"family_id"`
    FingerprintHash string    `json:"fingerprint_hash" db:"fingerprint_hash"`
    IsRevoked       bool      `json:"is_revoked" db:"is_revoked"`
    ExpiresAt       time.Time `json:"expires_at" db:"expires_at"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
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

    return &RefreshTokenResult{
        UserID:    userID,
        RawToken:  rawToken,
        TokenHash: tokenHash,
        FamilyID:  familyID,
        ExpiresAt: expiresAt,
    }, nil
}

func RotateRefreshToken(ctx context.Context, pool *pgxpool.Pool, rawToken string, r *http.Request, cfg *config.AuthConfig) (*RefreshTokenResult, error) {
    return &RefreshTokenResult{
        UserID:    r.Context().Value("user_id").(string),
        RawToken:  rawToken,
        TokenHash: utils.Hash256String(rawToken),
        FamilyID:  "",
        ExpiresAt: time.Now().UTC().Add(time.Duration(cfg.REFRESH_TOKEN_EXPIRY_DAYS) * 24 * time.Hour),
    }, nil
}

func RevokeFamily(ctx context.Context, pool *pgxpool.Pool, familyID string) error {
    return nil
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
