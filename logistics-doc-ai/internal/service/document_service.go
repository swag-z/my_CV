package service

import (
"context"
"fmt"
"io"
"time"

"github.com/google/uuid"
"github.com/logistics-doc-ai/internal/domain"
"github.com/logistics-doc-ai/internal/queue"
"github.com/logistics-doc-ai/internal/repository/postgres"
"github.com/logistics-doc-ai/internal/storage"
)

// DocumentService handles document business logic
type DocumentService struct {
docRepo      *repository.DocumentRepository
extRepo      *repository.ExtractionRepository
storage      storage.Storage
producer     *queue.Producer
defaultBucket string
}

// NewDocumentService creates a new document service
func NewDocumentService(
docRepo *repository.DocumentRepository,
extRepo *repository.ExtractionRepository,
storage storage.Storage,
producer *queue.Producer,
bucket string,
) *DocumentService {
return &DocumentService{
docRepo:       docRepo,
extRepo:       extRepo,
storage:       storage,
producer:      producer,
defaultBucket: bucket,
}
}

// UploadDocument uploads a new document
func (s *DocumentService) UploadDocument(ctx context.Context, filename, mimeType string, fileData io.Reader, fileSize int64) (*domain.Document, error) {
doc := &domain.Document{
ID:             uuid.New().String(),
OriginalFilename: filename,
MimeType:       mimeType,
FileSize:       fileSize,
DocumentType:   domain.DocumentTypeUnknown,
Status:         domain.StatusUploaded,
Attempts:       0,
CreatedAt:      time.Now().UTC(),
UpdatedAt:      time.Now().UTC(),
}

// Generate storage key
fileKey := fmt.Sprintf("documents/%s/%s", doc.ID[:8], filename)
doc.FileKey = fileKey

// Upload to storage
if err := s.storage.Upload(ctx, fileKey, fileData, fileSize, mimeType); err != nil {
return nil, fmt.Errorf("upload file: %w", err)
}

// Save to database
if err := s.docRepo.Create(ctx, doc); err != nil {
return nil, fmt.Errorf("create document: %w", err)
}

// Queue for processing
if err := s.producer.Publish(ctx, doc.ID, doc.Attempts); err != nil {
// Log but don't fail - document can be reprocessed later
}

return doc, nil
}

// GetDocument retrieves a document by ID
func (s *DocumentService) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
return s.docRepo.GetByID(ctx, id)
}

// ListDocuments lists documents with pagination
func (s *DocumentService) ListDocuments(ctx context.Context, limit, offset int, status, docType *string) ([]*domain.Document, int, error) {
return s.docRepo.List(ctx, limit, offset, status, docType)
}

// UpdateDocumentStatus updates document status
func (s *DocumentService) UpdateDocumentStatus(ctx context.Context, id string, status domain.DocumentStatus, errorMessage *string) error {
return s.docRepo.UpdateStatus(ctx, id, status, errorMessage)
}

// SetDocumentError sets error message on document
func (s *DocumentService) SetDocumentError(ctx context.Context, id string, errorMessage string) error {
return s.docRepo.SetError(ctx, id, errorMessage)
}

// IncrementAttempts increments document attempt counter
func (s *DocumentService) IncrementAttempts(ctx context.Context, id string) error {
return s.docRepo.IncrementAttempts(ctx, id)
}

// SetDocumentType sets document type
func (s *DocumentService) SetDocumentType(ctx context.Context, id string, docType domain.DocumentType) error {
return s.docRepo.SetDocumentType(ctx, id, docType)
}

// SaveExtraction saves extraction result
func (s *DocumentService) SaveExtraction(ctx context.Context, ext *domain.DocumentExtraction) error {
return s.extRepo.Create(ctx, ext)
}

// GetLatestExtraction gets latest extraction for document
func (s *DocumentService) GetLatestExtraction(ctx context.Context, documentID string) (*domain.DocumentExtraction, error) {
return s.extRepo.GetLatestByDocumentID(ctx, documentID)
}

// GetStorageDownloadURL gets presigned download URL
func (s *DocumentService) GetStorageDownloadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error) {
return s.storage.PresignedGetURL(ctx, fileKey, expiry)
}
