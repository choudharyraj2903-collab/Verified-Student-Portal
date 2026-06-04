

package tokens

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
    "student_portal/backend/config"
)

// Claims embeds standard JWT claims and adds system-specific fields
type Claims struct {
    jwt.RegisteredClaims
    Role         string   `json:"role"`
    CouncilCodes []string `json:"council_codes"`
}

// IssueAccessToken generates and signs an RS256 JWT access token
func IssueAccessToken(userID string, role string, councilCodes []string, cfg *config.JWTConfig) (string, error) {
    // Step 1 — Build claims
    claims := Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID,
            Issuer:    cfg.Issuer,
            IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
            ExpiresAt: jwt.NewNumericDate(
                time.Now().UTC().Add(time.Duration(cfg.AccessTokenExpiryMinutes) * time.Minute),
            ),
        },
        Role:         role,
        CouncilCodes: councilCodes,
    }

    // Step 2 — Sign with RS256 using private key
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    signed, err := token.SignedString(cfg.PrivateKey) // *rsa.PrivateKey
    if err != nil {
        return "", err
    }

    // Step 3 — Return signed token string
    return signed, nil
}

func VerifyAccessToken(tokenString string, cfg *config.JWTConfig) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
    tokenString,
    &Claims{},
    func(token *jwt.Token) (interface{}, error) {
        // CRITICAL — reject any token claiming a non-RS256 algorithm
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return cfg.PublicKey, nil
    },
)
	if err != nil {
    if errors.Is(err, jwt.ErrTokenExpired) {
        return nil, ErrAccessTokenExpired
    }
    return nil, ErrAccessTokenInvalid
	}
	if !token.Claims.(*Claims).VerifyIssuer(cfg.Issuer, true) {
    return nil, ErrAccessTokenInvalid
	}
	return token.Claims.(*Claims), nil
}

var (
    ErrAccessTokenExpired = fmt.Errorf("access token has expired")
    ErrAccessTokenInvalid = fmt.Errorf("access token is invalid")
)
