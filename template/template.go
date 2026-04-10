// Package template provides logic for parsing and rendering notification templates.
package template

import (
	"fmt"
	"regexp"
	"strings"
)

// placeholderRe matches {{variable_name}} tokens.
var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ExtractVariables returns the list of unique placeholder names found in content.
func ExtractVariables(content string) []string {
	seen := make(map[string]bool)
	var vars []string
	for _, match := range placeholderRe.FindAllStringSubmatch(content, -1) {
		name := match[1]
		if !seen[name] {
			seen[name] = true
			vars = append(vars, name)
		}
	}
	return vars
}

// Render replaces all {{key}} placeholders in content with values from data.
//
// Rules:
//   - Missing variables  → returns an error listing which keys are absent.
//   - Extra variables    → silently ignored (no error).
func Render(content string, data map[string]string) (string, error) {
	vars := ExtractVariables(content)

	// Collect any variables that are required but not supplied.
	var missing []string
	for _, v := range vars {
		if _, ok := data[v]; !ok {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("missing template variables: %s", strings.Join(missing, ", "))
	}

	// Replace each placeholder with its value.
	result := placeholderRe.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the key name from {{key}}.
		key := placeholderRe.FindStringSubmatch(match)[1]
		return data[key]
	})

	return result, nil
}

// Validate checks whether a template content string is well-formed.
// It returns an error if any placeholder is empty (e.g. {{}}).
func Validate(content string) error {
	// Check for malformed empty placeholders.
	if strings.Contains(content, "{{}}") {
		return fmt.Errorf("template contains empty placeholder: {{}}")
	}
	return nil
}
