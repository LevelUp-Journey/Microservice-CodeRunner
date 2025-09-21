package main

import (
	variables "code-runner/env"

	"github.com/gin-gonic/gin"
)

func main() {
	variables.LoadConfig()

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.Run(":" + variables.AppConfig.Port)
}
