package resources

type TestResource struct {
	TestId               string  `json:"id"`
	CodeVersionId        string  `json:"code_version_id"`
	Input                string  `json:"input"`
	ExpectedOutput       string  `json:"expected_output"`
	CustomValidationCode *string `json:"custom_validation_code,omitempty"`
}
