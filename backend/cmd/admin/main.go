package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"student_portal/backend/config"
	"student_portal/backend/db"
	"student_portal/backend/internals/middleware"
	"student_portal/backend/internals/modules/admin"
)

func main() {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	dbConn, err := db.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database schema applied")

	router := gin.Default()

	allowedOrigins := make(map[string]bool)
	for _, o := range cfg.Server.ALLOWED_ORIGINS {
		allowedOrigins[o] = true
	}
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Captcha-Token")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		status, _ := dbConn.HealthCheck(ctx)
		if status.Database == "ok" {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})

	router.GET("/api/admin/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "admin-api",
			"status":  "ok",
		})
	})

	auditLogger := middleware.NewAuditLogger(dbConn, &cfg)
	authenticator := middleware.NewAuthenticator(dbConn, &cfg, auditLogger)
	authorizer := middleware.NewAuthorizer(dbConn.Pool(), &cfg, auditLogger)
	ipGuard, err := middleware.NewCampusIPGuard(&cfg, auditLogger)
	if err != nil {
		log.Fatalf("Failed to configure campus IP guard: %v", err)
	}
	adminService := admin.NewAdminService(dbConn)
	admin.RegisterAdminRoutes(router, admin.NewAdminHandler(adminService), authenticator, authorizer, ipGuard)

	port := config.EnvInt("ADMIN_PORT", 8081)
	addr := ":" + fmt.Sprint(port)
	log.Printf("Starting admin API on %s (%s)", addr, cfg.Server.APP_ENV)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Admin API stopped: %v", err)
	}
}
