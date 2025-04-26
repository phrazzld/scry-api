package gemini_tests

import (
	"strings"
	"testing"
	"text/template" // Use text/template, not html/template for this test

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the template execution functionality used by createPrompt
func TestTemplateExecution(t *testing.T) {
	templateContent := "Generate flashcards for: {{.MemoText}}"
	tmpl, err := template.New("test").Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	data := struct{ MemoText string }{"This is a test memo."}

	var output string
	buf := new(strings.Builder)
	err = tmpl.Execute(buf, data)
	require.NoError(t, err, "Failed to execute template")
	output = buf.String()

	assert.Equal(
		t,
		"Generate flashcards for: This is a test memo.",
		output,
		"Template output should match",
	)
}

// Test template with invalid syntax
func TestInvalidTemplate(t *testing.T) {
	invalidTemplateContent := "Prompt: {{.MissingBrace"
	_, err := template.New("test").Parse(invalidTemplateContent)
	assert.Error(t, err, "Should error on invalid template syntax")
}

// Test template with missing field
func TestTemplateMissingField(t *testing.T) {
	templateContent := "Prompt: {{.MissingField}}"
	tmpl, err := template.New("test").Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	data := struct{ MemoText string }{"This is a test memo."}

	buf := new(strings.Builder)
	err = tmpl.Execute(buf, data)
	assert.Error(t, err, "Should error when template references missing field")
}

// Test template with HTML content (no escaping in text/template)
func TestTemplateWithHTMLContent(t *testing.T) {
	templateContent := "Prompt: {{.MemoText}}"
	tmpl, err := template.New("test").Parse(templateContent)
	require.NoError(t, err, "Failed to parse template")

	data := struct{ MemoText string }{"<script>alert('XSS')</script>"}

	buf := new(strings.Builder)
	err = tmpl.Execute(buf, data)
	require.NoError(t, err, "Failed to execute template")
	output := buf.String()

	// text/template doesn't escape HTML by default
	assert.Equal(
		t,
		"Prompt: <script>alert('XSS')</script>",
		output,
		"Template should not escape HTML in text templates",
	)
}
