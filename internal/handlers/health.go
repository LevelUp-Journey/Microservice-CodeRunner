package handlers

import "github.com/gin-gonic/gin"

// HealthHandlerBuilder creates a health handler for the API.
func HealthHandlerBuilder(api *gin.RouterGroup) {
	buildPingHandler(api)
}

func buildPingHandler(api *gin.RouterGroup) {
	api.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
}
