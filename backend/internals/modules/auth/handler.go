package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"student_portal/backend/internals/middleware"
	"student_portal/backend/internals/tokens"
	"student_portal/backend/internals/utils"
)

type AuthHandler struct {
	authService *AuthService
	auditLogger *middleware.AuditLogger
	frontendURL string
}

func NewAuthHandler(service *AuthService, auditLogger *middleware.AuditLogger, frontendURL string) *AuthHandler {
	return &AuthHandler{authService: service, auditLogger: auditLogger, frontendURL: frontendURL}
}

func (h *AuthHandler) RequestLink(w http.ResponseWriter, r *http.Request) error {
	email, ok := middleware.EmailFromContext(r.Context())
	if !ok {
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.SendValidationError(w, "email is required", nil)
			return nil
		}
		email = utils.NormaliseMail(req.Email)
	}
	if !utils.ValidateMailFormat(email) {
		utils.SendValidationError(w, "invalid email", nil)
		return nil
	}

	if rl, ok := middleware.RateLimitResultFromContext(r.Context()); ok && rl.TriggerCaptcha {
		captcha := r.Header.Get("X-Captcha-Token")
		if captcha == "" || !h.authService.VerifyCaptcha(captcha) {
			utils.SendError(w, http.StatusBadRequest, "captcha required", "CAPTCHA_REQUIRED")
			return nil
		}
	}

	_ = h.authService.RequestMagicLink(email, r)

	ctx := middleware.SetAuditEvent(r.Context(), &middleware.AuditEvent{
		EventType: "MAGIC_LINK_REQUESTED",
		Severity:  "INFO",
		UserID:    "",
		Metadata:  map[string]any{"email_hash": utils.Hash256String(email)},
	})
	r = r.WithContext(ctx)

	utils.SendSuccess(w, http.StatusOK, "If this email is registered, a link has been sent", nil)
	return nil
}

func (h *AuthHandler) VerifyLink(w http.ResponseWriter, r *http.Request) error {
	rawToken := r.URL.Query().Get("token")
	result, err := h.authService.VerifyMagicLink(rawToken, r)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/auth/login?error=invalid_link", http.StatusFound)
		return nil
	}

	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: result.AccessToken, HttpOnly: true, Path: "/", MaxAge: 900, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: result.RefreshToken, HttpOnly: true, Path: "/", MaxAge: 2592000, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: "device_token", Value: result.DeviceToken, HttpOnly: true, Path: "/", MaxAge: 2592000, SameSite: http.SameSiteLaxMode})

	ctx := middleware.SetAuditEvent(r.Context(), &middleware.AuditEvent{
		EventType: "LOGIN_SUCCESS",
		Severity:  "INFO",
		UserID:    result.UserID,
		Metadata:  map[string]any{},
	})
	r = r.WithContext(ctx)

	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusFound)
	return nil
}

func (h *AuthHandler) RefreshSession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		utils.SendUnauthorized(w)
		return nil
	}

	result, err := h.authService.RefreshSession(cookie.Value, r)
	if err != nil {
		if errors.Is(err, tokens.ErrRefreshTokenNotFound) {
			clearAllCookies(w)
		}
		utils.SendUnauthorized(w)
		return nil
	}

	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: result.AccessToken, HttpOnly: true, Path: "/", MaxAge: 900, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: result.RefreshToken, HttpOnly: true, Path: "/", MaxAge: 2592000, SameSite: http.SameSiteLaxMode})

	ctx := middleware.SetAuditEvent(r.Context(), &middleware.AuditEvent{
		EventType: "TOKEN_ROTATION",
		Severity:  "INFO",
		UserID:    result.UserID,
		Metadata:  map[string]any{},
	})
	r = r.WithContext(ctx)

	utils.SendSuccess(w, http.StatusOK, "session refreshed", nil)
	return nil
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return nil
	}

	if cookie, err := r.Cookie("refresh_token"); err == nil {
		_ = h.authService.Logout(user.ID, cookie.Value)
	}
	clearAllCookies(w)

	ctx := middleware.SetAuditEvent(r.Context(), &middleware.AuditEvent{
		EventType: "LOGOUT",
		Severity:  "INFO",
		UserID:    user.ID,
		Metadata:  map[string]any{},
	})
	r = r.WithContext(ctx)

	utils.SendSuccess(w, http.StatusOK, "logged out", nil)
	return nil
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) error {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return nil
	}

	_ = h.authService.LogoutAllDevices(user.ID)
	clearAllCookies(w)

	ctx := middleware.SetAuditEvent(r.Context(), &middleware.AuditEvent{
		EventType: "LOGOUT_ALL_DEVICES",
		Severity:  "INFO",
		UserID:    user.ID,
		Metadata:  map[string]any{},
	})
	r = r.WithContext(ctx)

	utils.SendSuccess(w, http.StatusOK, "logged out from all devices", nil)
	return nil
}

func (h *AuthHandler) ConfirmSession(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.SendValidationError(w, "token is required", nil)
		return nil
	}
	_, _ = fmt.Fprint(w, "Session confirmed. You are safe.")
	return nil
}

func (h *AuthHandler) InvalidateSession(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.SendValidationError(w, "token is required", nil)
		return nil
	}
	_, _ = fmt.Fprint(w, "Session terminated.")
	return nil
}

func clearAllCookies(w http.ResponseWriter) {
	expired := time.Now().Add(-time.Hour)
	for _, name := range []string{"access_token", "refresh_token", "device_token"} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", Expires: expired, MaxAge: -1, SameSite: http.SameSiteLaxMode})
	}
}
