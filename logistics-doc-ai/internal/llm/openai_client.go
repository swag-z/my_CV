package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/logistics-doc-ai/internal/domain"
)

// OpenAICompatibleClient implements LLM Client for OpenAI-compatible APIs
type OpenAICompatibleClient struct {
	baseURL     string
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	timeout     time.Duration
	client      *http.Client
}

// NewOpenAICompatibleClient creates new OpenAI-compatible client
func NewOpenAICompatibleClient(
	baseURL, apiKey, model string,
	temperature float64,
	maxTokens int,
	timeout time.Duration,
) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		apiKey:      apiKey,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
		timeout:     timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Classify classifies document type
func (c *OpenAICompatibleClient) Classify(ctx context.Context, text string) (*ClassificationResult, error) {
	prompt := buildClassificationPrompt(text)
	
	response, err := c.callLLM(ctx, prompt, true)
	if err != nil {
		return nil, fmt.Errorf("classify: %w", err)
	}

	jsonBytes, err := ParseJSONFromLLMResponse(response)
	if err != nil {
		return nil, fmt.Errorf("parse classification JSON: %w", err)
	}

	var result ClassificationResult
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal classification: %w", err)
	}

	if !result.DocumentType.IsValid() {
		result.DocumentType = domain.DocumentTypeUnknown
	}

	return &result, nil
}

// Extract extracts structured data from text
func (c *OpenAICompatibleClient) Extract(ctx context.Context, docType domain.DocumentType, text string) (*ExtractionResult, error) {
	prompt := buildExtractionPrompt(docType, text)
	
	// Try up to 2 times if JSON parsing fails
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		response, err := c.callLLM(ctx, prompt, true)
		if err != nil {
			lastErr = err
			continue
		}

		jsonBytes, err := ParseJSONFromLLMResponse(response)
		if err != nil {
			lastErr = err
			continue
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &payload); err != nil {
			lastErr = err
			continue
		}

		// Extract confidence if present, otherwise use defaults
		confidence := extractConfidence(payload)
		
		// Remove confidence from payload as it's separate
		delete(payload, "confidence")

		return &ExtractionResult{
			DocumentType: docType,
			Payload:      payload,
			Confidence:   confidence,
			Raw:          string(jsonBytes),
		}, nil
	}

	return nil, fmt.Errorf("extract after retries: %w", lastErr)
}

// callLLM makes HTTP request to LLM endpoint
func (c *OpenAICompatibleClient) callLLM(ctx context.Context, prompt string, jsonMode bool) (string, error) {
	requestBody := map[string]interface{}{
		"model":       c.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": c.temperature,
		"max_tokens":  c.maxTokens,
	}

	// Try to use JSON mode if supported
	if jsonMode {
		requestBody["response_format"] = map[string]string{"type": "json_object"}
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// extractConfidence extracts confidence values from payload or generates defaults
func extractConfidence(payload map[string]interface{}) map[string]float64 {
	confidence := make(map[string]float64)
	
	// Check if confidence is already in payload
	if conf, ok := payload["confidence"].(map[string]interface{}); ok {
		for k, v := range conf {
			if f, ok := v.(float64); ok {
				confidence[k] = f
			}
		}
		delete(payload, "confidence")
		return confidence
	}

	// Generate default confidence for key fields
	defaultConfidence := 0.85
	
	// Check common fields
	if _, ok := payload["document_number"]; ok {
		confidence["document_number"] = defaultConfidence
	}
	if _, ok := payload["document_date"]; ok {
		confidence["document_date"] = defaultConfidence
	}
	
	// Check nested structures
	for field, value := range payload {
		if nested, ok := value.(map[string]interface{}); ok {
			for key := range nested {
				confidence[fmt.Sprintf("%s_%s", field, key)] = defaultConfidence
			}
		}
	}

	return confidence
}

var _ Client = (*OpenAICompatibleClient)(nil)
