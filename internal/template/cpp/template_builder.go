package template

import (
	"fmt"

	"code-runner/internal/types"
)

// TemplateBuilder handles building the complete C++ template
type TemplateBuilder struct{}

// NewTemplateBuilder creates a new template builder
func NewTemplateBuilder() *TemplateBuilder {
	return &TemplateBuilder{}
}

// BuildTemplate constructs the complete template by replacing sections
func (b *TemplateBuilder) BuildTemplate(solutionCode, testCode string) string {
	template := `// Start Test
#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN
#include "doctest.h"
#include <cstring>

// Solution - Start
%s
// Solution - End

// Tests - Start
%s
// Tests - End
`

	return fmt.Sprintf(template, solutionCode, testCode)
}

// HasCustomValidation checks if any test has custom validation
func (b *TemplateBuilder) HasCustomValidation(tests []*types.TestCase) bool {
	for _, test := range tests {
		if test.HasCustomValidation() {
			return true
		}
	}
	return false
}
