// Adapter: convert RequireRole into gin.HandlerFunc

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Adapter for RequireRole
func (az *Authorizer) GinRequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		blocked := false
		handler := az.RequireRole(roles...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// only reached if RequireRole allowed the request through
		}))
		handler.ServeHTTP(c.Writer, c.Request)

		if c.Writer.Status() == http.StatusForbidden || c.Writer.Status() == http.StatusUnauthorized {
			blocked = true
		}
		if blocked {
			c.Abort()
			return
		}
		c.Next()
	}
}

// Adapter for RequireCouncilScope
func (az *Authorizer) GinRequireCouncilScope(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := az.RequireCouncilScope(param)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		handler.ServeHTTP(c.Writer, c.Request)
		if c.Writer.Status() == http.StatusForbidden || c.Writer.Status() == http.StatusUnauthorized {
			c.Abort()
			return
		}
		c.Next()
	}
}

// Adapter for RequireReAuth
func (az *Authorizer) GinRequireReAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := az.RequireReAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		handler.ServeHTTP(c.Writer, c.Request)
		if c.Writer.Status() == http.StatusForbidden {
			c.Abort()
			return
		}
		c.Next()
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
