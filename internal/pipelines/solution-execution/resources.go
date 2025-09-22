package solution_execution

type ExecuteSolutionResource struct {
	ExecutionID string `json:"executionId" example:"exec-123"`
	SolutionID  string `json:"solutionId" example:"sol-456"`
	Status      string `json:"status" example:"success"`
	Message     string `json:"message" example:"Solution executed successfully"`
	Output      string `json:"output,omitempty" example:"Hello World!"`
	Error       string `json:"error,omitempty" example:""`
}
