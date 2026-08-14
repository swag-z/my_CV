package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Common domain errors
var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrDocumentNotFound        = errors.New("document not found")
	ErrExtractionNotFound      = errors.New("extraction not found")
	ErrInvalidDocumentType     = errors.New("invalid document type")
	ErrInvalidExportFormat     = errors.New("invalid export format")
	ErrExportNotAllowed        = errors.New("export not allowed for this status")
	ErrReprocessNotAllowed     = errors.New("reprocess not allowed for this status")
	ErrUpdateNotAllowed        = errors.New("update not allowed for this status")
)

// DocumentType represents the type of logistics document
type DocumentType string

const (
	DocumentTypeTransportWaybill DocumentType = "transport_waybill"
	DocumentTypeFreightInvoice   DocumentType = "freight_invoice"
	DocumentTypeUnknown          DocumentType = "unknown"
)

// IsValid checks if the document type is known
func (t DocumentType) IsValid() bool {
	switch t {
	case DocumentTypeTransportWaybill, DocumentTypeFreightInvoice, DocumentTypeUnknown:
		return true
	}
	return false
}

// String returns the string representation
func (t DocumentType) String() string {
	return string(t)
}

// GetLabel returns Russian label for document type
func (t DocumentType) GetLabel() string {
	switch t {
	case DocumentTypeTransportWaybill:
		return "Транспортная накладная"
	case DocumentTypeFreightInvoice:
		return "Счёт/УПД"
	case DocumentTypeUnknown:
		return "Не определён"
	default:
		return "Неизвестно"
	}
}

// DocumentStatus represents the current state of a document
type DocumentStatus string

const (
	StatusUploaded            DocumentStatus = "uploaded"
	StatusQueued              DocumentStatus = "queued"
	StatusOCRInProgress       DocumentStatus = "ocr_in_progress"
	StatusExtractionInProgress DocumentStatus = "extraction_in_progress"
	StatusNeedsReview         DocumentStatus = "needs_review"
	StatusReviewed            DocumentStatus = "reviewed"
	StatusExported            DocumentStatus = "exported"
	StatusFailed              DocumentStatus = "failed"
)

// IsValid checks if status is valid
func (s DocumentStatus) IsValid() bool {
	switch s {
	case StatusUploaded, StatusQueued, StatusOCRInProgress, StatusExtractionInProgress,
		StatusNeedsReview, StatusReviewed, StatusExported, StatusFailed:
		return true
	}
	return false
}

// CanTransitionTo checks if transition to target status is allowed
func (s DocumentStatus) CanTransitionTo(target DocumentStatus) bool {
	transitions := map[DocumentStatus][]DocumentStatus{
		StatusUploaded:             {StatusQueued},
		StatusQueued:               {StatusOCRInProgress},
		StatusOCRInProgress:        {StatusExtractionInProgress, StatusFailed},
		StatusExtractionInProgress: {StatusNeedsReview, StatusReviewed, StatusFailed},
		StatusNeedsReview:          {StatusReviewed},
		StatusReviewed:             {StatusExported},
		StatusExported:             {},
		StatusFailed:               {StatusQueued},
	}

	allowed, exists := transitions[s]
	if !exists {
		return false
	}

	for _, t := range allowed {
		if t == target {
			return true
		}
	}
	return false
}

// GetColor returns Tailwind CSS color class for badge
func (s DocumentStatus) GetColor() string {
	switch s {
	case StatusUploaded:
		return "bg-gray-500"
	case StatusQueued:
		return "bg-blue-500"
	case StatusOCRInProgress, StatusExtractionInProgress:
		return "bg-yellow-500"
	case StatusNeedsReview:
		return "bg-orange-500"
	case StatusReviewed, StatusExported:
		return "bg-green-500"
	case StatusFailed:
		return "bg-red-500"
	default:
		return "bg-gray-500"
	}
}

// GetLabel returns Russian label for status
func (s DocumentStatus) GetLabel() string {
	switch s {
	case StatusUploaded:
		return "Загружен"
	case StatusQueued:
		return "В очереди"
	case StatusOCRInProgress:
		return "Распознавание"
	case StatusExtractionInProgress:
		return "Извлечение данных"
	case StatusNeedsReview:
		return "Требует проверки"
	case StatusReviewed:
		return "Проверен"
	case StatusExported:
		return "Экспортирован"
	case StatusFailed:
		return "Ошибка"
	default:
		return "Неизвестно"
	}
}

// ExtractionSource represents where extraction came from
type ExtractionSource string

const (
	ExtractionSourceLLM  ExtractionSource = "llm"
	ExtractionSourceMock ExtractionSource = "mock"
	ExtractionSourceUser ExtractionSource = "user"
)

// IsValid checks if source is valid
func (s ExtractionSource) IsValid() bool {
	switch s {
	case ExtractionSourceLLM, ExtractionSourceMock, ExtractionSourceUser:
		return true
	}
	return false
}

// ExportFormat represents export format
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
)

// IsValid checks if format is valid
func (f ExportFormat) IsValid() bool {
	switch f {
	case ExportFormatCSV, ExportFormatJSON:
		return true
	}
	return false
}

// ExportStatus represents export job status
type ExportStatus string

const (
	ExportStatusPending   ExportStatus = "pending"
	ExportStatusCompleted ExportStatus = "completed"
	ExportStatusFailed    ExportStatus = "failed"
)

// IsValid checks if export status is valid
func (s ExportStatus) IsValid() bool {
	switch s {
	case ExportStatusPending, ExportStatusCompleted, ExportStatusFailed:
		return true
	}
	return false
}

// Document represents an uploaded logistics document
type Document struct {
	ID             uuid.UUID      `json:"id"`
	CompanyID      *uuid.UUID     `json:"company_id,omitempty"`
	OriginalFilename string       `json:"original_filename"`
	MimeType       string         `json:"mime_type"`
	FileKey        string         `json:"file_key"`
	FileSize       int64          `json:"file_size"`
	DocumentType   DocumentType   `json:"document_type"`
	Status         DocumentStatus `json:"status"`
	ErrorMessage   *string        `json:"error_message,omitempty"`
	Attempts       int            `json:"attempts"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// NewDocument creates a new document with defaults
func NewDocument(filename, mimeType, fileKey string, fileSize int64, companyID *uuid.UUID) *Document {
	now := time.Now().UTC()
	return &Document{
		ID:             uuid.New(),
		CompanyID:      companyID,
		OriginalFilename: filename,
		MimeType:       mimeType,
		FileKey:        fileKey,
		FileSize:       fileSize,
		DocumentType:   DocumentTypeUnknown,
		Status:         StatusUploaded,
		Attempts:       0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateStatus updates document status if transition is valid
func (d *Document) UpdateStatus(newStatus DocumentStatus) error {
	if !d.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid transition from %s to %s: %w", d.Status, newStatus, ErrInvalidStatusTransition)
	}
	d.Status = newStatus
	d.UpdatedAt = time.Now().UTC()
	return nil
}

// SetError marks document as failed
func (d *Document) SetError(errMsg string) {
	d.Status = StatusFailed
	d.ErrorMessage = &errMsg
	d.UpdatedAt = time.Now().UTC()
}

// ResetForReprocess prepares document for reprocessing
func (d *Document) ResetForReprocess() {
	d.Status = StatusQueued
	d.ErrorMessage = nil
	d.Attempts = 0
	d.UpdatedAt = time.Now().UTC()
}

// DocumentExtraction represents extracted data from document
type DocumentExtraction struct {
	ID           uuid.UUID              `json:"id"`
	DocumentID   uuid.UUID              `json:"document_id"`
	DocumentType DocumentType           `json:"document_type"`
	Payload      map[string]interface{} `json:"payload"`
	Confidence   map[string]float64     `json:"confidence"`
	Validation   ValidationResult       `json:"validation"`
	Source       ExtractionSource       `json:"source"`
	Version      int                    `json:"version"`
	CreatedAt    time.Time              `json:"created_at"`
}

// NewDocumentExtraction creates new extraction record
func NewDocumentExtraction(docID uuid.UUID, docType DocumentType, payload map[string]interface{}, confidence map[string]float64, source ExtractionSource) *DocumentExtraction {
	now := time.Now().UTC()
	return &DocumentExtraction{
		ID:           uuid.New(),
		DocumentID:   docID,
		DocumentType: docType,
		Payload:      payload,
		Confidence:   confidence,
		Validation:   ValidationResult{Valid: true},
		Source:       source,
		Version:      1,
		CreatedAt:    now,
	}
}

// ValidationResult contains validation errors and warnings
type ValidationResult struct {
	Valid    bool            `json:"valid"`
	Errors   []ValidationIssue `json:"errors,omitempty"`
	Warnings []ValidationIssue `json:"warnings,omitempty"`
}

// ValidationIssue represents a single validation problem
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AddError adds validation error
func (vr *ValidationResult) AddError(field, message string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, ValidationIssue{Field: field, Message: message})
}

// AddWarning adds validation warning
func (vr *ValidationResult) AddWarning(field, message string) {
	vr.Warnings = append(vr.Warnings, ValidationIssue{Field: field, Message: message})
}

// ReviewCorrection represents user correction
type ReviewCorrection struct {
	ID         uuid.UUID  `json:"id"`
	DocumentID uuid.UUID  `json:"document_id"`
	FieldPath  string     `json:"field_path"`
	OldValue   *string    `json:"old_value,omitempty"`
	NewValue   *string    `json:"new_value,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ExportJob represents export operation
type ExportJob struct {
	ID            uuid.UUID    `json:"id"`
	DocumentID    uuid.UUID    `json:"document_id"`
	Format        ExportFormat `json:"format"`
	Status        ExportStatus `json:"status"`
	ResultFileKey *string      `json:"result_file_key,omitempty"`
	ErrorMessage  *string      `json:"error_message,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// AuditLog represents audit trail entry
type AuditLog struct {
	ID         uuid.UUID                `json:"id"`
	DocumentID *uuid.UUID               `json:"document_id,omitempty"`
	Action     string                   `json:"action"`
	Payload    map[string]interface{}   `json:"payload"`
	CreatedAt  time.Time                `json:"created_at"`
}
