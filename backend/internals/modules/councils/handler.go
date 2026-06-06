package councils

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "student_portal/backend/internals/utils"
)

type CouncilsHandler struct {
    service *CouncilsService
}

func NewCouncilsHandler(service *CouncilsService) *CouncilsHandler {
    return &CouncilsHandler{service: service}
}

func (h *CouncilsHandler) ListCouncils(c *gin.Context) {
    councils, err := h.service.ListCouncils(c.Request.Context())
    if err != nil {
        utils.SendError(c.Writer, http.StatusInternalServerError, "failed to fetch councils", "COUNCILS_FETCH_FAILED")
        return
    }

    utils.SendSuccess(c.Writer, http.StatusOK, "councils retrieved", councils)
}
