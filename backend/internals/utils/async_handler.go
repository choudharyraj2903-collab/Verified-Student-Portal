package utils

import (
    "github.com/gin-gonic/gin"
    "student_portal/backend/config"
)

// GinHandlerFunc is a Gin handler signature that returns an error
type GinHandlerFunc func(c *gin.Context) error

// HandleGin wraps a GinHandlerFunc into a standard gin.HandlerFunc
// If fn returns an error, SendInternalError is called.
// If fn returns nil, the handler is assumed to have already written its response.
func HandleGin(fn GinHandlerFunc, cfg *config.AppConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        if err := fn(c); err != nil {
            // Use our standardized response utility
            SendInternalError(c.Writer, err, cfg)
            // Optionally abort the context so Gin doesn’t continue other handlers
            c.Abort()
        }
    }
}
