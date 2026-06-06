package admin

import (
	"github.com/gin-gonic/gin"
	"student_portal/backend/internals/middleware"
)

func RegisterAdminRoutes(r *gin.Engine, handler *AdminHandler, auth *middleware.Authenticator, az *middleware.Authorizer, ipGuard *middleware.CampusIPGuard) {
	adminGroup := r.Group("/admin")
	adminGroup.Use(ipGuard.GinGuard(), auth.GinAuthenticate())
	{
		// Council Admin management
		adminGroup.POST("/council-admins", az.GinRequireRole("SUPER_ADMIN"), az.GinRequireReAuth(), middleware.GinHTTPHandler(handler.CreateCouncilAdmin))
		adminGroup.GET("/council-admins", az.GinRequireRole("SUPER_ADMIN"), middleware.GinHTTPHandler(handler.ListCouncilAdmins))
		adminGroup.DELETE("/council-admins/:id", az.GinRequireRole("SUPER_ADMIN"), az.GinRequireReAuth(), middleware.GinHTTPHandler(handler.RemoveCouncilAdmin))

		// Student management
		adminGroup.GET("/students", az.GinRequireRole("SUPER_ADMIN", "COUNCIL_ADMIN"), middleware.GinHTTPHandler(handler.ListAllStudents))
		adminGroup.GET("/students/:id", az.GinRequireRole("SUPER_ADMIN", "COUNCIL_ADMIN"), middleware.GinHTTPHandler(handler.GetStudentDetail))
		adminGroup.POST("/students/:id/deactivate", az.GinRequireRole("SUPER_ADMIN"), az.GinRequireReAuth(), middleware.GinHTTPHandler(handler.DeactivateStudent))

		// Verification management (Super Admin override)
		adminGroup.GET("/verification", az.GinRequireRole("SUPER_ADMIN"), middleware.GinHTTPHandler(handler.ListAllRequests))
		adminGroup.PUT("/verification/:id/approve", az.GinRequireRole("SUPER_ADMIN"), middleware.GinHTTPHandler(handler.AdminApprove))
		adminGroup.PUT("/verification/:id/reject", az.GinRequireRole("SUPER_ADMIN"), middleware.GinHTTPHandler(handler.AdminReject))

		// PDF report
		adminGroup.GET("/students/:id/report", az.GinRequireRole("SUPER_ADMIN", "COUNCIL_ADMIN"), middleware.GinHTTPHandler(handler.GenerateStudentReport))

		// Audit log
		adminGroup.GET("/audit-logs", az.GinRequireRole("SUPER_ADMIN"), middleware.GinHTTPHandler(handler.GetAuditLogs))
	}
}
