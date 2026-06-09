package auth

import (
	"github.com/gin-gonic/gin"
	"student_portal/backend/internals/middleware"
)

func RegisterAuthRoutes(r *gin.Engine, handler *AuthHandler, auth *middleware.Authenticator) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/magic-link", middleware.GinHandler(handler.RequestLink))
		authGroup.GET("/verify", middleware.GinHandler(handler.VerifyLink))
		authGroup.POST("/refresh", middleware.GinHandler(handler.RefreshSession))
		authGroup.POST("/logout", auth.GinAuthenticate(), middleware.GinHandler(handler.Logout))
		authGroup.POST("/logout-all", auth.GinAuthenticate(), middleware.GinHandler(handler.LogoutAll))
		authGroup.GET("/confirm", middleware.GinHandler(handler.ConfirmSession))
		authGroup.GET("/invalidate", middleware.GinHandler(handler.InvalidateSession))
	}
}
