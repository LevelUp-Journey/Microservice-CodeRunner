package app

import (
	variables "code-runner/env"
	handlers "code-runner/internal/handlers"
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Configuration related to server routes
func SetUpRouter() *gin.Engine {
	variables.LoadConfig()

	router := gin.Default()

	api := router.Group("/api/" + variables.AppConfig.APIVersion)
	{
		handlers.HealthHandlerBuilder(api)
		handlers.ChallengeExecutionHandlerBuilder(api)
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	fmt.Println("Swagger UI disponible en: http://localhost:" + variables.AppConfig.Port + "/swagger/index.html")

	return router
}
