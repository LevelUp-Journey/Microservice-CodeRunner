package app

import (
	"code-runner/docs"
	variables "code-runner/env"
	"code-runner/internal/handlers"
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Configuration related to server routes
func SetUpRouter() *gin.Engine {
	variables.LoadConfig()

	router := gin.Default()

	// Configuración dinámica de Swagger usando variables de entorno
	docs.SwaggerInfo.Title = variables.AppConfig.AppName
	docs.SwaggerInfo.Description = "API para ejecutar código de soluciones de desafíos de programación"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = variables.AppConfig.Host + ":" + variables.AppConfig.Port
	docs.SwaggerInfo.BasePath = variables.AppConfig.BasePath
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	api := router.Group(variables.AppConfig.BasePath)
	{
		handlers.HealthHandlerBuilder(api)
		handlers.SolutionsExecutionsHandlerBuilder(api)
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	fmt.Println("Swagger UI disponible en: http://localhost:" + variables.AppConfig.Port + "/swagger/index.html")
	fmt.Println("API Base Path: " + variables.AppConfig.BasePath)
	fmt.Println("API Version: " + variables.AppConfig.APIVersion)

	return router
}
