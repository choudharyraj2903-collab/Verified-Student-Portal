package councils

import (
    "github.com/gin-gonic/gin"
)

func RegisterCouncilsRoutes(r *gin.Engine, handler *CouncilsHandler) {
    r.GET("/councils", handler.ListCouncils)
}
