package handlers

import (
	"github.com/gin-gonic/gin"
)

func SolutionsExecutionsHandlerBuilder(api *gin.RouterGroup) {
	buildAddSolutionsExecutionsHandlerBySolutionId(api)
}

func buildAddSolutionsExecutionsHandlerBySolutionId(api *gin.RouterGroup) {

	// Execute a solution from Challenges Microservice
	api.POST("/solutions/:solutionId/executions", AddSolutionsExecutionsHandler)
}

// @Title Add Solutions Executions Handler
// @Summary Execute a solution from Challenges Microservice
// @Description Endpoint to execute a solution from Challenges Microservice
// @Tags Solutions Executions
// @Produce json
// @Param solutionId path string true "Solution ID"
// @Success 200 {object} solution_execution.ExecuteSolutionResource
// @Router /api/v1/solutions/{solutionId}/executions [post]
func AddSolutionsExecutionsHandler(c *gin.Context) {

}
