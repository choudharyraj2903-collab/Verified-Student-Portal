package middleware

import (
	"net/http"
	"encoding/json"
	"context"
	"student_portal/backend/config"
	"student_portal/backend/internals/utils"
)

// Instead of differnent var we are using a struct for better readability
type DomainGuardConfig struct {
	// Allowed domains here only iitk.ac.in is allowed
	AllowedDomain string
	// Pointer to the audit logger
	AuditLogger   *AuditLogger
}

func NewDomainGuard(cfg *config.AppConfig, al *AuditLogger) *DomainGuardConfig {
	// Basically makes a new domain guard struct with the config and pointer of logger 
	return &DomainGuardConfig{cfg.Campus.EMAIL_DOMAIN, al}
}

func (dg *DomainGuardConfig) Guard(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Email string `json:"email"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            utils.SendValidationError(w, "Invalid request", dg.AuditLogger.cfg)
            return
        }

        email := utils.NormaliseMail(req.Email)
        if !utils.ValidateMailFormat(email) {
            utils.SendValidationError(w, "Invalid email", dg.AuditLogger.cfg)
            return
        }

        if !utils.ValidateMailDomain(email, dg.AllowedDomain) {
            utils.SendUnauthorized(w)
            dg.AuditLogger.Log(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
            return
        }

        // Add email to context
        ctx := context.WithValue(r.Context(), contextKeyEmail, email)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func EmailFromContext(ctx context.Context) (string, bool) {
    email, ok := ctx.Value(contextKeyEmail).(string)
    return email, ok
}


type contextKey string

const (
    contextKeyEmail       contextKey = "email"
    contextKeyUser        contextKey = "user"
    contextKeyRequestData contextKey = "requestData"
)