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
	"student_portal/backend/internals/mail"
	"student_portal/backend/internals/middleware"
	"student_portal/backend/internals/modules/auth"
	"student_portal/backend/internals/modules/councils"
	"student_portal/backend/internals/modules/profile"
	"student_portal/backend/internals/modules/verification"
)

func main() {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	dbConn, err := db.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to configure database pool: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database schema applied")

	router := gin.Default()

	// CORS — allow frontend origin with credentials
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

	registerHealth(router, dbConn)

	router.GET("/api/student/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "student-api",
			"status":  "ok",
		})
	})

	auditLogger := middleware.NewAuditLogger(dbConn, &cfg)
	authenticator := middleware.NewAuthenticator(dbConn, &cfg, auditLogger)
	authorizer := middleware.NewAuthorizer(dbConn.Pool(), &cfg, auditLogger)

	mailer, err := mail.NewMailer(&cfg.Mail)
	if err != nil {
		log.Fatalf("Failed to configure mailer: %v", err)
	}
	mailService, err := mail.NewMailService(mailer, &cfg)
	if err != nil {
		log.Fatalf("Failed to configure mail service: %v", err)
	}

	authService := auth.NewAuthService(dbConn, &cfg, mailService)
	auth.RegisterAuthRoutes(router, auth.NewAuthHandler(authService, auditLogger, cfg.Server.FRONTEND_URL), authenticator)

	councilsService := councils.NewCouncilsService(dbConn.Pool())
	councils.RegisterCouncilsRoutes(router, councils.NewCouncilsHandler(councilsService))

	profileService := profile.NewProfileService(dbConn.Pool(), auditLogger)
	profile.RegisterProfileRoutes(router, profile.NewProfileHandler(profileService), authenticator, authorizer)

	verificationService := verification.NewVerificationService(dbConn.Pool(), auditLogger)
	verification.RegisterVerificationRoutes(router, verification.NewVerificationHandler(verificationService), authenticator, authorizer)

	addr := ":" + fmt.Sprint(cfg.Server.PORT)
	log.Printf("Starting student API on %s (%s)", addr, cfg.Server.APP_ENV)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Student API stopped: %v", err)
	}
}

func registerHealth(router *gin.Engine, dbConn *db.DB) {
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		status, _ := dbConn.HealthCheck(ctx)
		if status.Database == "ok" {
			c.JSON(http.StatusOK, status)
			return
		}
		c.JSON(http.StatusServiceUnavailable, status)
	})
}
