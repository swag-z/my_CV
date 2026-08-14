package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// OCRResult represents the result of OCR processing
type OCRResult struct {
	FullText  string            `json:"full_text"`
	Pages     []PageResult      `json:"pages,omitempty"`
	PageCount int               `json:"page_count"`
	Metadata  OCRMetadata       `json:"metadata,omitempty"`
}

// OCRMetadata contains OCR processing metadata
type OCRMetadata struct {
	Engine      string    `json:"engine"`
	PageCount   int       `json:"page_count"`
	ProcessedAt time.Time `json:"processed_at"`
	Mock        bool      `json:"mock"`
}

// PageResult represents a single page OCR result
type PageResult struct {
	PageNumber int     `json:"page_number"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Extract adapts OCRResult to the interface expected by processing service
func (r *OCRResult) Extract(ctx context.Context, filename string, reader io.Reader, size int64) (*OCRResult, error) {
	// This method is for interface compatibility
	// The actual extraction is done by Recognize
	return r, nil
}

// Client interface for OCR operations
type Client interface {
	Extract(ctx context.Context, filename string, reader io.Reader, size int64) (*OCRResult, error)
}

// HTTPClient implements OCR Client interface
type HTTPClient struct {
	baseURL       string
	internalToken string
	timeout       time.Duration
	httpClient    *http.Client
}

// NewClient creates a new OCR HTTP client
func NewClient(baseURL, internalToken string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		timeout:       timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Extract sends document to OCR service and returns text
func (c *HTTPClient) Extract(ctx context.Context, filename string, reader io.Reader, size int64) (*OCRResult, error) {
	// Read all data from reader
	fileData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read file data: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/ocr", body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add internal token if configured
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OCR service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result OCRResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

var _ Client = (*HTTPClient)(nil)
