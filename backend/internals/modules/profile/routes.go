package profile

import (
    "github.com/gin-gonic/gin"
    "student_portal/backend/internals/middleware"
)

func RegisterProfileRoutes(r *gin.Engine, handler *ProfileHandler, auth *middleware.Authenticator, az *middleware.Authorizer) {
    profileGroup := r.Group("/profile")
    profileGroup.Use(auth.GinAuthenticate())
    {
        profileGroup.GET("", handler.GetMyProfile)
        profileGroup.POST("", handler.CreateProfile)
        profileGroup.PUT("", handler.UpdateProfile)

        // Council Admin or Super Admin can view any profile
        profileGroup.GET("/:userID", az.GinRequireRole("COUNCIL_ADMIN", "SUPER_ADMIN"), handler.GetProfileByID)
    }
}
