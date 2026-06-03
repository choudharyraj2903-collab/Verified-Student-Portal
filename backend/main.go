package main 

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"student_portal/backend/config"
	"fmt"
)

func main() {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		panic(err)
	}
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	router.Run(":" + fmt.Sprint(cfg.Server.PORT))
	fmt.Println("Port:", cfg.Server.PORT)
	fmt.Println("Environment:", cfg.Server.APP_ENV)
}