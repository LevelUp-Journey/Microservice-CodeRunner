package docker

import (
	"fmt"
	"log"
)

// logExecutionResults imprime los resultados de la ejecución
func (e *DockerExecutor) logExecutionResults(result *ExecutionResult) {
	log.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📊 EXECUTION COMPLETED")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("   ⏱️  Duration: %dms", result.ExecutionTimeMS)
	log.Printf("   📊 Exit code: %d", result.ExitCode)
	log.Printf("   📝 Stdout length: %d bytes", len(result.StdOut))
	log.Printf("   📝 Stderr length: %d bytes", len(result.StdErr))

	if len(result.StdOut) > 0 {
		log.Printf("\n💬 STDOUT OUTPUT:")
		log.Printf("─────────────────────────────────────────────────────")
		stdoutPreview := result.StdOut
		if len(stdoutPreview) > 1000 {
			stdoutPreview = stdoutPreview[:1000] + fmt.Sprintf("\n... (truncated, total: %d bytes)", len(result.StdOut))
		}
		log.Printf("%s", stdoutPreview)
		log.Printf("─────────────────────────────────────────────────────")
	}

	if len(result.StdErr) > 0 {
		log.Printf("\n⚠️  STDERR OUTPUT:")
		log.Printf("─────────────────────────────────────────────────────")
		log.Printf("%s", result.StdErr)
		log.Printf("─────────────────────────────────────────────────────")
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
