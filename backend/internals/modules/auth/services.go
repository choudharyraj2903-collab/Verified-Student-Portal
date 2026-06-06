package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"student_portal/backend/config"
	"student_portal/backend/db"
	"student_portal/backend/internals/mail"
	"student_portal/backend/internals/tokens"
	"student_portal/backend/internals/utils"
)

// =========================================================================
// Types
// =========================================================================

// User is defined locally in the auth package — avoids import cycle with
// other modules that also define User. Auth only needs these fields.
type User struct {
	ID           string
	Email        string
	Role         string
	IsActive     bool
	CouncilCodes []string
}

type SessionResult struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	DeviceToken  string
}

// =========================================================================
// Service
// =========================================================================

type AuthService struct {
	db   *db.DB
	cfg  *config.AppConfig
	mail *mail.MailService // correct type — not a local Mailer alias
}

func NewAuthService(database *db.DB, cfg *config.AppConfig, mailSvc *mail.MailService) *AuthService {
	return &AuthService{
		db:   database,
		cfg:  cfg,
		mail: mailSvc,
	}
}

// =========================================================================
// RequestMagicLink
// =========================================================================

func (s *AuthService) RequestMagicLink(email string, r *http.Request) error {
	// Normalize before any processing
	email = utils.NormaliseMail(email)

	// Get existing user or create new STUDENT
	user, err := s.getOrCreateUser(context.Background(), email)
	if err != nil {
		return fmt.Errorf("get or create user failed: %w", err)
	}

	// Generate magic token — request is needed for UA binding
	result, err := tokens.Generate(r, s.cfg)
	if err != nil {
		return fmt.Errorf("magic token generation failed: %w", err)
	}

	// Store hash in DB — never store raw token
	if err := tokens.Store(context.Background(), s.db.Pool(), user.ID, result); err != nil {
		return fmt.Errorf("magic token store failed: %w", err)
	}

	// Build full URL and send email
	// APP_URL is the correct config field — no MAGIC_LINK_BASE_URL field exists
	magicURL := tokens.BuildMagicLinkURL(s.cfg.Server.APP_URL, result.RawToken)
	if err := s.mail.SendMagicLink(email, magicURL, result.ExpiresAt); err != nil {
		return fmt.Errorf("send magic link failed: %w", err)
	}

	return nil
}

// =========================================================================
// VerifyMagicLink
// =========================================================================

func (s *AuthService) VerifyMagicLink(rawToken string, r *http.Request) (*SessionResult, error) {
	// Verify token — validates hash, expiry, UA binding, atomic used flag
	// tokens.Verify returns *MagicToken not a result with UserID directly
	magicToken, err := tokens.Verify(context.Background(), s.db.Pool(), rawToken, r)
	if err != nil {
		return nil, fmt.Errorf("magic token verification failed: %w", err)
	}

	// Fetch user using UserID from the verified magic token struct
	user, err := s.getUserByID(context.Background(), magicToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}
	// Fetch active council scopes for JWT claims
	scopes, err := s.getActiveCouncilScopes(context.Background(), user.ID)
	if err != nil {
		return nil, fmt.Errorf("get council scopes failed: %w", err)
	}

	// Issue access token — stateless JWT
	accessToken, err := tokens.IssueAccessToken(user.ID, user.Role, scopes, &s.cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("issue access token failed: %w", err)
	}

	// Issue refresh token — new family ID for this session
	familyID := utils.GenerateRandomString(16)
	refreshToken, err := tokens.IssueRefreshToken(
		context.Background(),
		s.db.Pool(),
		user.ID,
		familyID,
		r,
		&s.cfg.Auth,
	)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token failed: %w", err)
	}

	// Issue device token — upserts trust entry for this browser fingerprint
	deviceToken, err := tokens.IssueDeviceToken(
		context.Background(),
		s.db.Pool(),
		user.ID,
		r,
		&s.cfg.Auth,
	)
	if err != nil {
		return nil, fmt.Errorf("issue device token failed: %w", err)
	}

	// Issue invalidation token for "Was This You?" email
	rawInvalidationToken, err := tokens.IssueInvalidationToken(
		context.Background(),
		s.db.Pool(),
		user.ID,
		familyID,
		&s.cfg.Auth,
	)
	if err != nil {
		// Non-fatal — session still valid without the invalidation email
		// Log but do not fail the login
		fmt.Printf("invalidation token issue failed: %v\n", err)
	} else {
		confirmURL := tokens.BuildConfirmationURL(s.cfg.Server.APP_URL, rawInvalidationToken)
		invalidateURL := tokens.BuildInvalidationURL(s.cfg.Server.APP_URL, rawInvalidationToken)
		// Non-fatal — do not fail login if email fails to send
		_ = s.mail.SendWasThisYou(user.Email, rawInvalidationToken, magicToken.ExpiresAt, r)
		_ = confirmURL
		_ = invalidateURL
	}

	return &SessionResult{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.RawToken,
		DeviceToken:  deviceToken,
	}, nil
}

func (s *AuthService) VerifyCaptcha(token string) bool {
	return token != ""
}

// =========================================================================
// RefreshSession
// =========================================================================

func (s *AuthService) RefreshSession(rawRefreshToken string, r *http.Request) (*SessionResult, error) {
	// Rotate refresh token — validates, revokes old, issues new in transaction
	newResult, err := tokens.RotateRefreshToken(
		context.Background(),
		s.db.Pool(),
		rawRefreshToken,
		r,
		&s.cfg.Auth,
	)
	if err != nil {
		return nil, fmt.Errorf("token rotation failed: %w", err)
	}

	// Fetch user with current DB state — not from JWT claims
	user, err := s.getUserByID(context.Background(), newResult.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	// Re-fetch scopes from DB — scope changes take effect on next refresh
	scopes, err := s.getActiveCouncilScopes(context.Background(), user.ID)
	if err != nil {
		return nil, fmt.Errorf("get council scopes failed: %w", err)
	}

	// Issue fresh access token with current role and scopes
	accessToken, err := tokens.IssueAccessToken(user.ID, user.Role, scopes, &s.cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("issue access token failed: %w", err)
	}

	return &SessionResult{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: newResult.RawToken,
		DeviceToken:  "", // device token unchanged on refresh
	}, nil
}

// =========================================================================
// Logout
// =========================================================================

func (s *AuthService) Logout(userID, rawRefreshToken string) error {
	// Hash to look up the token record
	tokenHash := utils.Hash256String(rawRefreshToken)

	// Fetch family ID so we can revoke the whole family
	var familyID string
	err := s.db.Pool().QueryRow(context.Background(),
		`SELECT family_id FROM refresh_tokens
		 WHERE token_hash = $1 AND is_revoked = FALSE`,
		tokenHash,
	).Scan(&familyID)
	if err != nil {
		// Token already revoked or not found — treat as successful logout
		return nil
	}

	// Revoke entire token family
	if _, err := s.db.Pool().Exec(context.Background(),
		`UPDATE refresh_tokens SET is_revoked = TRUE WHERE family_id = $1`,
		familyID,
	); err != nil {
		return fmt.Errorf("revoke token family failed: %w", err)
	}

	// Revoke device trust for this user
	if _, err := s.db.Pool().Exec(context.Background(),
		`UPDATE device_trust SET is_revoked = TRUE
		 WHERE user_id = $1 AND is_revoked = FALSE`,
		userID,
	); err != nil {
		return fmt.Errorf("revoke device token failed: %w", err)
	}

	return nil
}

// =========================================================================
// LogoutAllDevices
// =========================================================================

func (s *AuthService) LogoutAllDevices(userID string) error {
	if _, err := s.db.Pool().Exec(context.Background(),
		`UPDATE refresh_tokens SET is_revoked = TRUE
		 WHERE user_id = $1 AND is_revoked = FALSE`,
		userID,
	); err != nil {
		return fmt.Errorf("revoke all refresh tokens failed: %w", err)
	}

	if _, err := s.db.Pool().Exec(context.Background(),
		`UPDATE device_trust SET is_revoked = TRUE
		 WHERE user_id = $1 AND is_revoked = FALSE`,
		userID,
	); err != nil {
		return fmt.Errorf("revoke all device tokens failed: %w", err)
	}

	return nil
}

// =========================================================================
// Private DB helpers
// =========================================================================

func (s *AuthService) getOrCreateUser(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.Pool().QueryRow(ctx,
		`SELECT id, email, role, is_active FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Role, &u.IsActive)

	if err == pgx.ErrNoRows {
		// First login — create account with default STUDENT role
		err = s.db.Pool().QueryRow(ctx,
			`INSERT INTO users (email, role, is_active, created_at, updated_at)
			 VALUES ($1, 'STUDENT', TRUE, NOW(), NOW())
			 RETURNING id, email, role, is_active`,
			email,
		).Scan(&u.ID, &u.Email, &u.Role, &u.IsActive)
		if err != nil {
			return nil, fmt.Errorf("create user failed: %w", err)
		}
		return &u, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email failed: %w", err)
	}
	return &u, nil
}

func (s *AuthService) getUserByID(ctx context.Context, userID string) (*User, error) {
	var u User
	err := s.db.Pool().QueryRow(ctx,
		`SELECT id, email, role, is_active FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.Role, &u.IsActive)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id failed: %w", err)
	}
	return &u, nil
}

func (s *AuthService) getActiveCouncilScopes(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Pool().Query(ctx,
		`SELECT c.code
		 FROM councils c
		 JOIN user_council_scopes ucs ON c.id = ucs.council_id
		 WHERE ucs.user_id = $1
		   AND ucs.is_active = TRUE
		   AND (ucs.expires_at IS NULL OR ucs.expires_at > NOW())`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get council scopes failed: %w", err)
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan council scope failed: %w", err)
		}
		scopes = append(scopes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("council scopes rows error: %w", err)
	}

	return scopes, nil
}
