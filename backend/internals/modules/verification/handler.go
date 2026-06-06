package verification

import (
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "student_portal/backend/internals/middleware"
    "student_portal/backend/internals/utils"
)

type VerificationHandler struct {
    verificationService *VerificationService
}

func NewVerificationHandler(service *VerificationService) *VerificationHandler {
    return &VerificationHandler{verificationService: service}
}

func (h *VerificationHandler) SubmitRequest(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }
    var req struct {
        CouncilCode string    `json:"council_code"`
        Title       string    `json:"title"`
        Description string    `json:"description"`
        ProofLink   string    `json:"proof_link"`
        PorDate     time.Time `json:"por_date"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c.Writer, "invalid request body", nil)
        return
    }

    if strings.TrimSpace(req.CouncilCode) == "" {
        utils.SendValidationError(c.Writer, "invalid council code", nil)
        return
    }
    if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" || strings.TrimSpace(req.ProofLink) == "" {
        utils.SendValidationError(c.Writer, "missing required fields", nil)
        return
    }
    if _, err := url.ParseRequestURI(req.ProofLink); err != nil {
        utils.SendValidationError(c.Writer, "invalid proof link", nil)
        return
    }
    if req.PorDate.IsZero() || req.PorDate.After(time.Now()) {
        utils.SendValidationError(c.Writer, "invalid PoR date", nil)
        return
    }

    request, err := h.verificationService.SubmitRequest(user.ID, &SubmitRequestData{
        CouncilID:   strings.ToUpper(strings.TrimSpace(req.CouncilCode)),
        Title:       req.Title,
        Description: req.Description,
        ProofLink:   req.ProofLink,
        PorDate:     req.PorDate,
    })
    if err != nil {
        switch err {
        case ErrProfileIncomplete:
            utils.SendError(c.Writer, http.StatusForbidden, "profile incomplete", "PROFILE_REQUIRED")
        case ErrInvalidCouncil:
            utils.SendError(c.Writer, http.StatusBadRequest, "invalid council code", "INVALID_COUNCIL")
        case ErrDuplicateRequest:
            utils.SendError(c.Writer, http.StatusConflict, "duplicate pending request", "DUPLICATE_REQUEST")
        case ErrTooManyPending:
            utils.SendError(c.Writer, http.StatusTooManyRequests, "too many pending requests", "TOO_MANY_PENDING")
        case ErrInvalidPorDate:
            utils.SendError(c.Writer, http.StatusBadRequest, "invalid PoR date", "INVALID_POR_DATE")
        default:
            utils.SendInternalError(c.Writer, err, nil)
        }
        return
    }

    ctx := middleware.SetAuditEvent(c.Request.Context(), &middleware.AuditEvent{
        EventType: "VERIFICATION_REQUEST_SUBMITTED",
        Severity:  "INFO",
        UserID:    user.ID,
        Metadata:  map[string]any{"council": req.CouncilCode},
    })
    c.Request = c.Request.WithContext(ctx)
    utils.SendSuccess(c.Writer, http.StatusCreated, "request submitted", request)
}

func (h *VerificationHandler) GetMyRequests(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    status := c.Query("status")
    council := c.Query("council")
    requests, err := h.verificationService.GetMyRequests(user.ID, status, council)
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    utils.SendSuccess(c.Writer, http.StatusOK, "requests retrieved", requests)
}

func (h *VerificationHandler) GetRequestByID(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    id := c.Param("id")
    request, err := h.verificationService.GetRequestByID(id, user)
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    utils.SendSuccess(c.Writer, http.StatusOK, "request retrieved", request)
}

func (h *VerificationHandler) WithdrawRequest(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    id := c.Param("id")
    if err := h.verificationService.WithdrawRequest(id, user.ID); err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    utils.SendSuccess(c.Writer, http.StatusOK, "request withdrawn", nil)
}

func (h *VerificationHandler) GetCouncilRequests(c *gin.Context) {
    _, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    councilCode, ok := middleware.CouncilFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    status := c.Query("status")
    result, err := h.verificationService.GetCouncilRequests(councilCode, status)
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    utils.SendSuccess(c.Writer, http.StatusOK, "council requests retrieved", result)
}

func (h *VerificationHandler) ApproveRequest(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    id := c.Param("id")
    var req struct {
        Remarks string `json:"remarks"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c.Writer, "invalid request body", nil)
        return
    }

    request, err := h.verificationService.ApproveRequest(id, user.ID, req.Remarks, user)
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    ctx := middleware.SetAuditEvent(c.Request.Context(), &middleware.AuditEvent{
        EventType: "VERIFICATION_REQUEST_APPROVED",
        Severity:  "INFO",
        UserID:    user.ID,
        Metadata:  map[string]any{"request_id": id},
    })
    c.Request = c.Request.WithContext(ctx)
    utils.SendSuccess(c.Writer, http.StatusOK, "request approved", request)
}

func (h *VerificationHandler) RejectRequest(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    id := c.Param("id")
    var req struct {
        Remarks string `json:"remarks"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c.Writer, "invalid request body", nil)
        return
    }
    if strings.TrimSpace(req.Remarks) == "" {
        utils.SendValidationError(c.Writer, "remarks required", nil)
        return
    }

    request, err := h.verificationService.RejectRequest(id, user.ID, req.Remarks, user)
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    ctx := middleware.SetAuditEvent(c.Request.Context(), &middleware.AuditEvent{
        EventType: "VERIFICATION_REQUEST_REJECTED",
        Severity:  "WARN",
        UserID:    user.ID,
        Metadata:  map[string]any{"request_id": id},
    })
    c.Request = c.Request.WithContext(ctx)
    utils.SendSuccess(c.Writer, http.StatusOK, "request rejected", request)
}

func (h *VerificationHandler) GetVerifiedCard(c *gin.Context) {
    userID := c.Param("userID")
    card, err := h.verificationService.GetVerifiedCard(userID)
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }
    utils.SendSuccess(c.Writer, http.StatusOK, "verified card retrieved", card)
}
