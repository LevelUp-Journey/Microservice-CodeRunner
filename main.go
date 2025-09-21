package main

import (
	variables "code-runner/env"
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files" // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "code-runner/docs"
)

func main() {
	variables.LoadConfig()

	router := gin.Default()

	// Grupo con prefijo de API
	api := router.Group("/api/" + variables.AppConfig.APIVersion)
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	fmt.Println("Swagger UI disponible en: http://localhost:" + variables.AppConfig.Port + "/swagger/index.html")

	router.Run(":" + variables.AppConfig.Port)
}
