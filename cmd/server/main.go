// @Title Microservice CodeRunner API
// @Version 1.0
// @Schemes http https
package main

import (
	_ "code-runner/docs"
	"code-runner/internal/app"
)

func main() {
	app.Run()
}
