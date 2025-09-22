package handlers

import "github.com/gin-gonic/gin"

// HealthHandlerBuilder creates a health handler for the API.
func HealthHandlerBuilder(api *gin.RouterGroup) {
	buildPingHandler(api)
}

func buildPingHandler(api *gin.RouterGroup) {
	api.GET("/health/ping", pingHandler)
}

// @Title Ping
// @Summary Returns pong
// @Description Endpoint to test the server
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/health/ping [get]
func pingHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}
