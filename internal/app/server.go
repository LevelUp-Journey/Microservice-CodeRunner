package app

import (
	variables "code-runner/env"
)

// Configuration Related to Server
func Run() {
	router := SetUpRouter()

	router.Run(":" + variables.AppConfig.Port)
}
