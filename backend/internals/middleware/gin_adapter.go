// Adapter: convert RequireRole into gin.HandlerFunc

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Adapter for RequireRole
func (az *Authorizer) GinRequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap Gin context into net/http style
		handler := az.RequireRole(roles...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next() // continue Gin chain
		}))
		handler.ServeHTTP(c.Writer, c.Request)

		// If blocked, stop Gin chain
		if c.Writer.Status() == http.StatusForbidden || c.Writer.Status() == http.StatusUnauthorized {
			c.Abort()
		}
	}
}

// Adapter for RequireCouncilScope
func (az *Authorizer) GinRequireCouncilScope(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := az.RequireCouncilScope(param)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		}))
		handler.ServeHTTP(c.Writer, c.Request)
		if c.Writer.Status() == http.StatusForbidden || c.Writer.Status() == http.StatusUnauthorized {
			c.Abort()
		}
	}
}

// Adapter for RequireReAuth
func (az *Authorizer) GinRequireReAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := az.RequireReAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Next()
		}))
		handler.ServeHTTP(c.Writer, c.Request)
		if c.Writer.Status() == http.StatusForbidden {
			c.Abort()
		}
	}
}
func GinMiddleware(mw func(http.Handler) http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		writtenBefore := c.Writer.Written()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
		})

		handler := mw(next)
		handler.ServeHTTP(c.Writer, c.Request)

		if c.Writer.Written() && !writtenBefore {
			c.Abort()
			return
		}
	}
}

func (a *Authenticator) GinAuthenticate() gin.HandlerFunc {
	return GinMiddleware(a.Authenticate)
}

func (g *CampusIPGuard) GinGuard() gin.HandlerFunc {
	return GinMiddleware(g.Guard)
}

// Adapter: convert func(http.ResponseWriter, *http.Request) error into gin.HandlerFunc
func GinHandler(fn func(http.ResponseWriter, *http.Request) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call the original handler
		err := fn(c.Writer, c.Request)
		if err != nil {
			// If your handler returns an error, decide how to handle it
			// For example, send a 500 response
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}
	}
}

func GinHTTPHandler(fn func(http.ResponseWriter, *http.Request)) gin.HandlerFunc {
	return func(c *gin.Context) {
		fn(c.Writer, c.Request)
	}
}
