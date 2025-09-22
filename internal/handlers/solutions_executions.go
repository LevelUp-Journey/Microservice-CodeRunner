package handlers

import "github.com/gin-gonic/gin"

func SolutionsExecutionsHandlerBuilder(api *gin.RouterGroup) {
	buildAddSolutionsExecutionsHandlerBySolutionId(api)
}

func buildAddSolutionsExecutionsHandlerBySolutionId(api *gin.RouterGroup) {

	// Execute a solution from Challenges Microservice
	api.POST("/solutions/:solutionId/executions", func(c *gin.Context) {
		solutionId := c.Param("solutionId")

		c.JSON(200, gin.H{
			"message":    "Welcome to the Challenge Execution API!",
			"status":     "success",
			"solutionId": solutionId,
		})
	})
}
