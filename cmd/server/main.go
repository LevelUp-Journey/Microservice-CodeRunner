// @title Microservice CodeRunner API
// @version 1.0
// @description API para ejecutar código de soluciones de desafíos de programación
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8084
// @BasePath /api/v1

// @schemes http https
package main

import (
	_ "code-runner/docs"
	"code-runner/internal/app"
)

func main() {
	app.Run()
}
