package handlers

import (
	solution_execution "code-runner/internal/pipelines/solution-execution"

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
	solutionId := c.Param("solutionId")

	// Crear una respuesta usando el tipo ExecuteSolutionResource
	response := solution_execution.ExecuteSolutionResource{
		ExecutionID: "exec-" + solutionId,
		SolutionID:  solutionId,
		Status:      "success",
		Message:     "Solution executed successfully",
		Output:      "Hello World!",
		Error:       "",
	}

	c.JSON(200, response)
}
