// @title Microservice CodeRunner API
// @version 1.0
// @schemes http https
package main

import (
	_ "code-runner/docs"
	"code-runner/internal/app"
)

func main() {
	app.Run()
}
