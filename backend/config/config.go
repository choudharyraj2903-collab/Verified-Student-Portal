package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type AppConfig struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Mail     MailConfig
	Auth     AuthConfig
	Campus   CampusConfig
}

type ServerConfig struct {
	APP_ENV         string
	PORT            int
	APP_URL         string
	FRONTEND_URL    string
	ALLOWED_ORIGINS []string
}

type DatabaseConfig struct {
	DB_HOST                      string
	DB_PORT                      int
	DB_USER                      string
	DB_PASSWORD                  string
	DB_NAME                      string
	DB_SSL_MODE                  string
	DB_MAX_OPEN_CONNS            int
	DB_MAX_IDLE_CONNS            int
	DB_CONN_MAX_LIFETIME_MINUTES int
}

type JWTConfig struct {
	PrivateKey               *rsa.PrivateKey
	PublicKey                *rsa.PublicKey
	AccessTokenExpiryMinutes int
	Issuer                   string
}

type MailConfig struct {
	MAIL_HOST         string
	MAIL_PORT         int
	MAIL_USERNAME     string
	MAIL_PASSWORD     string
	MAIL_FROM_ADDRESS string
	MAIL_FROM_NAME    string
	MAIL_ENCRYPTION   string
}

type AuthConfig struct {
	MAGIC_LINK_EXPIRY_MINUTES         int
	MAGIC_LINK_MAX_REQUESTS_PER_HOUR  int
	REFRESH_TOKEN_EXPIRY_DAYS         int
	DEVICE_TOKEN_EXPIRY_DAYS          int
	INVALIDATION_TOKEN_EXPIRY_MINUTES int
	CAPTCHA_THRESHOLD                 int
	RATE_LIMIT_IP_MAX_PER_HOUR        int
}

type CampusConfig struct {
	EMAIL_DOMAIN string
	IP_RANGES    []*net.IPNet
}

func Env(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func EnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

func CutAtComma(s string) []string {
	// Split string by comma and trim spaces
	parts := strings.Split(s, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}
func parsePrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read private key: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM in private key")
	}

	// Try PKCS#1 first
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// Fallback to PKCS#8
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA private key: %v", err)
	}
	rsaKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaKey, nil
}

func parsePublicKey(path string) (*rsa.PublicKey, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY_PATH (%s) not readable: %v", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY_PATH (%s) invalid PEM", path)
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY_PATH (%s) invalid RSA public key: %v", path, err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY_PATH (%s) is not RSA", path)
	}
	return rsaPub, nil
}
func LoadServerConfig() (ServerConfig, error) {
	// Load server configuration from environment variables or config file
	AppEnv := Env("APP_ENV", "development")
	Port := EnvInt("PORT", 8080)
	AppURL := Env("APP_URL", "http://localhost:8080")
	FrontendURL := Env("FRONTEND_URL", "http://localhost:3000")
	AllowedOrigins := CutAtComma(Env("ALLOWED_ORIGINS", "http://localhost:3000"))
	return ServerConfig{AppEnv, Port, AppURL, FrontendURL, AllowedOrigins}, nil
}

func LoadDatabaseConfig() (DatabaseConfig, error) {
	// Load database configuration from environment variables or config file
	DBHost := Env("DB_HOST", "localhost")
	DBPort := EnvInt("DB_PORT", 5432)
	DBUser := Env("DB_USER", "postgres")
	DBPassword := Env("DB_PASSWORD", "")
	DBName := Env("DB_NAME", "student_portal")
	DBSSLMode := Env("DB_SSL_MODE", "disable")
	DBMaxOpenConns := EnvInt("DB_MAX_OPEN_CONNS", 60)
	DBMaxIdleConns := EnvInt("DB_MAX_IDLE_CONNS", 10)
	DBConnMaxLifetimeMinutes := EnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)
	return DatabaseConfig{DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode, DBMaxOpenConns, DBMaxIdleConns, DBConnMaxLifetimeMinutes}, nil
}

func LoadJWTConfig() (JWTConfig, error) {
	// Load JWT configuration from environment variables or config file
	JWTPrivateKeyPath, err := parsePrivateKey(Env("JWT_PRIVATE_KEY_PATH", "keys/private.pem"))
	if err != nil {
		privateKey, genErr := rsa.GenerateKey(rand.Reader, 2048)
		if genErr != nil {
			return JWTConfig{}, fmt.Errorf("failed to parse JWT private key: %v", err)
		}
		JWTAccessTokenExpiryMinutes := EnvInt("JWT_ACCESS_TOKEN_EXPIRY_MINUTES", 15)
		JWTIssuer := Env("JWT_ISSUER", "student_portal")
		return JWTConfig{privateKey, &privateKey.PublicKey, JWTAccessTokenExpiryMinutes, JWTIssuer}, nil
	}
	JWTPublicKeyPath, err := parsePublicKey(Env("JWT_PUBLIC_KEY_PATH", "keys/public.pem"))
	if err != nil {
		return JWTConfig{}, fmt.Errorf("failed to parse JWT public key: %v", err)
	}
	JWTAccessTokenExpiryMinutes := EnvInt("JWT_ACCESS_TOKEN_EXPIRY_MINUTES", 15)
	JWTIssuer := Env("JWT_ISSUER", "student_portal")
	return JWTConfig{JWTPrivateKeyPath, JWTPublicKeyPath, JWTAccessTokenExpiryMinutes, JWTIssuer}, nil
}

func LoadMailConfig() (MailConfig, error) {
	// Load mail configuration from environment variables or config file
	MailHost := Env("MAIL_HOST", "smtp.example.com")
	MailPort := EnvInt("MAIL_PORT", 587)
	MailUsername := Env("MAIL_USERNAME", "")
	MailPassword := Env("MAIL_PASSWORD", "")
	MailFromAddress := Env("MAIL_FROM_ADDRESS", "nBb0K@example.com")
	MailFromName := Env("MAIL_FROM_NAME", "Student Portal")
	MailEncryption := Env("MAIL_ENCRYPTION", "tls")
	return MailConfig{MailHost, MailPort, MailUsername, MailPassword, MailFromAddress, MailFromName, MailEncryption}, nil
}

func LoadAuthConfig() (AuthConfig, error) {
	// Load authentication configuration from environment variables or config file
	MagicLinkExpiryMinutes := EnvInt("MAGIC_LINK_EXPIRY_MINUTES", 15)
	MagicLinkMaxRequestsPerHour := EnvInt("MAGIC_LINK_MAX_REQUESTS_PER_HOUR", 5)
	RefreshTokenExpiryDays := EnvInt("REFRESH_TOKEN_EXPIRY_DAYS", 30)
	DeviceTokenExpiryDays := EnvInt("DEVICE_TOKEN_EXPIRY_DAYS", 30)
	InvalidationTokenExpiryMinutes := EnvInt("INVALIDATION_TOKEN_EXPIRY_MINUTES", 15)
	CaptchaThreshold := EnvInt("CAPTCHA_THRESHOLD", 0)
	RateLimitIPMaxPerHour := EnvInt("RATE_LIMIT_IP_MAX_PER_HOUR", 100)
	return AuthConfig{MagicLinkExpiryMinutes, MagicLinkMaxRequestsPerHour, RefreshTokenExpiryDays, DeviceTokenExpiryDays, InvalidationTokenExpiryMinutes, CaptchaThreshold, RateLimitIPMaxPerHour}, nil
}

func LoadCampusConfig() (CampusConfig, error) {
	domain := Env("EMAIL_DOMAIN", "iitk.ac.in")
	ranges := Env("IP_RANGES", "10.0.0.0/8,172.16.0.0/12")
	parts := CutAtComma(ranges)
	var nets []*net.IPNet
	for _, r := range parts {
		_, ipnet, err := net.ParseCIDR(r)
		if err != nil {
			return CampusConfig{}, fmt.Errorf("invalid CIDR %s: %v", r, err)
		}
		nets = append(nets, ipnet)
	}
	return CampusConfig{domain, nets}, nil
}

func LoadAppConfig() (AppConfig, error) {
	// Load all configurations and return as AppConfig struct
	// Walk up from CWD until we find a .env file (handles any invocation path)
	dir, _ := os.Getwd()
	for {
		if err := godotenv.Load(filepath.Join(dir, ".env")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	serverConfig, err := LoadServerConfig()
	if err != nil {
		return AppConfig{}, err
	}
	databaseConfig, err := LoadDatabaseConfig()
	if err != nil {
		return AppConfig{}, err
	}
	jwtConfig, err := LoadJWTConfig()
	if err != nil {
		return AppConfig{}, err
	}
	mailConfig, err := LoadMailConfig()
	if err != nil {
		return AppConfig{}, err
	}
	authConfig, err := LoadAuthConfig()
	if err != nil {
		return AppConfig{}, err
	}
	campusConfig, err := LoadCampusConfig()
	if err != nil {
		return AppConfig{}, err
	}
	return AppConfig{
		Server:   serverConfig,
		Database: databaseConfig,
		JWT:      jwtConfig,
		Mail:     mailConfig,
		Auth:     authConfig,
		Campus:   campusConfig,
	}, nil
}
