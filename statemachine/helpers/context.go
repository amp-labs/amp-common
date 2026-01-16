// Package helpers provides utility functions for state machine operations.
package helpers

import (
	"bytes"
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/amp-labs/amp-common/statemachine"
)

// ContextChunk represents a piece of context information.
type ContextChunk struct {
	Title   string
	Content string
}

// FormatContextChunks formats context chunks into a readable string.
func FormatContextChunks(chunks []ContextChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	var sb strings.Builder

	for i, chunk := range chunks {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		if chunk.Title != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n", chunk.Title))
		}

		sb.WriteString(chunk.Content)
	}

	return sb.String()
}

// TruncateToTokenLimit truncates content to approximately fit within a token limit
// Uses rough heuristic: 1 token ≈ 4 characters.
func TruncateToTokenLimit(content string, maxTokens int) string {
	maxChars := maxTokens * 4 //nolint:mnd // Token-to-character conversion heuristic
	if len(content) <= maxChars {
		return content
	}

	truncated := content[:maxChars]
	// Try to cut at a word boundary
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxChars-100 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "\n\n[... truncated ...]"
}

// BuildSystemPrompt builds a system prompt from a template and context data.
func BuildSystemPrompt(tmplStr string, contextData map[string]any) (string, error) {
	tmpl, err := template.New("system_prompt").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse system prompt template: %w", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, contextData)
	if err != nil {
		return "", fmt.Errorf("failed to execute system prompt template: %w", err)
	}

	return buf.String(), nil
}

// ExtractContextFromState extracts multiple values from state machine context.
func ExtractContextFromState(smCtx *statemachine.Context, keys []string) map[string]any {
	result := make(map[string]any)

	for _, key := range keys {
		if value, exists := smCtx.Get(key); exists {
			result[key] = value
		}
	}

	return result
}

// MergeContextData merges multiple context data maps, with later maps taking precedence.
func MergeContextData(contexts ...map[string]any) map[string]any {
	result := make(map[string]any)

	for _, ctx := range contexts {
		maps.Copy(result, ctx)
	}

	return result
}

// FormatContextForPrompt formats context data as a readable string for inclusion in prompts.
func FormatContextForPrompt(contextData map[string]any) string {
	if len(contextData) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("**Context:**\n")

	for key, value := range contextData {
		sb.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}

	return sb.String()
}
