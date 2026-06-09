package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"student_portal/backend/internals/middleware"
	"student_portal/backend/internals/utils"
)

type AdminHandler struct {
	adminService *AdminService
}

func NewAdminHandler(service *AdminService) *AdminHandler {
	return &AdminHandler{adminService: service}
}

func (h *AdminHandler) CreateCouncilAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		CouncilCode string `json:"council_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendValidationError(w, "invalid request body", nil)
		return
	}

	admin, err := h.adminService.CreateCouncilAdmin(req.Email, req.CouncilCode)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}

	ctx := middleware.SetAuditEvent(r.Context(), &middleware.AuditEvent{
		EventType: "ROLE_CHANGED",
		Severity:  "CRITICAL",
		UserID:    admin.ID,
		Metadata:  map[string]any{"new_role": "COUNCIL_ADMIN", "council": admin.CouncilCode},
	})
	r = r.WithContext(ctx)

	utils.SendSuccess(w, http.StatusCreated, "council admin created", admin)
}

func (h *AdminHandler) ListCouncilAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.adminService.ListCouncilAdmins()
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "council admins retrieved", admins)
}

func (h *AdminHandler) RemoveCouncilAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.adminService.RemoveCouncilAdmin(id); err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "council admin removed", nil)
}

func (h *AdminHandler) ListAllStudents(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return
	}

	filters := map[string]string{
		"search":  r.URL.Query().Get("search"),
		"year":    r.URL.Query().Get("year"),
		"council": r.URL.Query().Get("council"),
		"status":  r.URL.Query().Get("status"),
		"page":    r.URL.Query().Get("page"),
	}

	result, err := h.adminService.ListStudents(user.Role, user.CouncilCodes, filters)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "students retrieved", result)
}

func (h *AdminHandler) GetStudentDetail(w http.ResponseWriter, r *http.Request) {
	studentID := r.PathValue("id")
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return
	}

	result, err := h.adminService.GetStudentDetail(studentID, user.Role, user.CouncilCodes)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "student detail retrieved", result)
}

func (h *AdminHandler) DeactivateStudent(w http.ResponseWriter, r *http.Request) {
	studentID := r.PathValue("id")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendValidationError(w, "invalid request body", nil)
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		utils.SendValidationError(w, "reason required", nil)
		return
	}

	if err := h.adminService.DeactivateStudent(studentID, req.Reason); err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "student deactivated", nil)
}

func (h *AdminHandler) ListAllRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := h.adminService.ListAllRequests(r.URL.Query().Get("status"))
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "verification requests retrieved", requests)
}

func (h *AdminHandler) AdminApprove(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return
	}
	id := r.PathValue("id")
	var req struct {
		Remarks string `json:"remarks"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	request, err := h.adminService.AdminApprove(id, user.ID, req.Remarks)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "request approved", request)
}

func (h *AdminHandler) AdminReject(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return
	}
	id := r.PathValue("id")
	var req struct {
		Remarks string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendValidationError(w, "invalid request body", nil)
		return
	}
	if strings.TrimSpace(req.Remarks) == "" {
		utils.SendValidationError(w, "remarks required", nil)
		return
	}

	request, err := h.adminService.AdminReject(id, user.ID, req.Remarks)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "request rejected", request)
}

func (h *AdminHandler) GenerateStudentReport(w http.ResponseWriter, r *http.Request) {
	studentID := r.PathValue("id")
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.SendUnauthorized(w)
		return
	}

	reportData, err := h.adminService.GenerateStudentReport(studentID, user.Role, user.CouncilCodes)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}

	pdfBytes, rollNumber, err := utils.RenderReportToPDF(reportData)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report_%s.pdf\"", rollNumber))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func (h *AdminHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	filters := map[string]string{
		"event":    r.URL.Query().Get("event"),
		"severity": r.URL.Query().Get("severity"),
		"user_id":  r.URL.Query().Get("user_id"),
		"from":     r.URL.Query().Get("from"),
		"to":       r.URL.Query().Get("to"),
		"page":     r.URL.Query().Get("page"),
	}

	result, err := h.adminService.GetAuditLogs(filters)
	if err != nil {
		utils.SendInternalError(w, err, nil)
		return
	}
	utils.SendSuccess(w, http.StatusOK, "audit logs retrieved", result)
}
