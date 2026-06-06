package verification

import (
	"github.com/gin-gonic/gin"
	"student_portal/backend/internals/middleware"
)

func RegisterVerificationRoutes(r *gin.Engine, handler *VerificationHandler, auth *middleware.Authenticator, az *middleware.Authorizer) {
	verGroup := r.Group("/verification")
	verGroup.Use(auth.GinAuthenticate())
	{
		// Student routes
		verGroup.POST("", az.GinRequireRole("STUDENT"), handler.SubmitRequest)
		verGroup.GET("", az.GinRequireRole("STUDENT"), handler.GetMyRequests)
		verGroup.GET("/:id", handler.GetRequestByID)
		verGroup.DELETE("/:id", az.GinRequireRole("STUDENT"), handler.WithdrawRequest)

		// Council Admin routes
		verGroup.GET("/council/:councilCode", az.GinRequireRole("COUNCIL_ADMIN"), az.GinRequireCouncilScope("councilCode"), handler.GetCouncilRequests)
		verGroup.PUT("/:id/approve", az.GinRequireRole("COUNCIL_ADMIN", "SUPER_ADMIN"), handler.ApproveRequest)
		verGroup.PUT("/:id/reject", az.GinRequireRole("COUNCIL_ADMIN", "SUPER_ADMIN"), handler.RejectRequest)

		// Verified card
		verGroup.GET("/card/:userID", handler.GetVerifiedCard)
	}
}
