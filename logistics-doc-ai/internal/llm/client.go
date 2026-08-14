package llm

import (
"context"
"encoding/json"
"fmt"
"strings"
"time"

"github.com/logistics-doc-ai/internal/domain"
)

// ClassificationResult represents document classification result
type ClassificationResult struct {
DocumentType domain.DocumentType `json:"document_type"`
Confidence   float64             `json:"confidence"`
}

// ExtractionResult represents LLM extraction result
type ExtractionResult struct {
DocumentType domain.DocumentType      `json:"document_type"`
Payload      map[string]interface{}   `json:"payload"`
Confidence   map[string]float64       `json:"confidence"`
Raw          string                   `json:"raw"`
}

// Client interface for LLM operations
type Client interface {
Classify(ctx context.Context, text string) (*ClassificationResult, error)
Extract(ctx context.Context, docType domain.DocumentType, text string) (*ExtractionResult, error)
}

// ParseJSONFromLLMResponse extracts JSON from LLM response
func ParseJSONFromLLMResponse(response string) ([]byte, error) {
response = strings.TrimSpace(response)

// Try direct parsing first
var data interface{}
if err := json.Unmarshal([]byte(response), &data); err == nil {
return []byte(response), nil
}

// Look for ```json blocks
startIdx := strings.Index(response, "```json")
endIdx := strings.Index(response, "```")

if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
jsonStr := response[startIdx+7 : endIdx]
jsonStr = strings.TrimSpace(jsonStr)
if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
return []byte(jsonStr), nil
}
}

// Look for generic code blocks
startIdx = strings.Index(response, "```")
if startIdx != -1 {
rest := response[startIdx+3:]
endIdx = strings.Index(rest, "```")
if endIdx != -1 {
jsonStr := strings.TrimSpace(rest[:endIdx])
// Remove language identifier if present
lines := strings.Split(jsonStr, "\n")
if len(lines) > 0 && !strings.HasPrefix(lines[0], "{") && !strings.HasPrefix(lines[0], "[") {
jsonStr = strings.Join(lines[1:], "\n")
}
jsonStr = strings.TrimSpace(jsonStr)
if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
return []byte(jsonStr), nil
}
}
}

// Find first { and last }
firstBrace := strings.Index(response, "{")
lastBrace := strings.LastIndex(response, "}")
if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
potentialJSON := response[firstBrace : lastBrace+1]
if err := json.Unmarshal([]byte(potentialJSON), &data); err == nil {
return []byte(potentialJSON), nil
}
}

return nil, fmt.Errorf("no valid JSON found in response")
}
