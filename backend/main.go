
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
)

func main() {
    cfg, err := config.LoadAppConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    dbConn, err := db.NewDB(&cfg.Database) // FIX: pass pointer
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer dbConn.Close() // FIX: use dbConn

    router := gin.Default()

    // Health endpoint using DB
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

    // Run server
    router.Run(":" + fmt.Sprint(cfg.Server.PORT))
    fmt.Println("Port:", cfg.Server.PORT)
    fmt.Println("Environment:", cfg.Server.APP_ENV)
}
