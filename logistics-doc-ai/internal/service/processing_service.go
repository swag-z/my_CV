package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/logistics-doc-ai/internal/domain"
	"github.com/logistics-doc-ai/internal/llm"
	"github.com/logistics-doc-ai/internal/ocr"
	"github.com/logistics-doc-ai/internal/repository/postgres"
	"github.com/logistics-doc-ai/internal/storage"
	"github.com/logistics-doc-ai/internal/validation"
)

// ProcessingService handles document processing pipeline
type ProcessingService struct {
	docRepo           *postgres.DocumentRepository
	extRepo           *postgres.ExtractionRepository
	auditRepo         *postgres.AuditRepository
	storage           storage.Storage
	ocrClient         ocr.Client
	llmClient         llm.Client
	bucket            string
	confidenceThreshold float64
	autoApprove       bool
	maxAttempts       int
	logger            zerolog.Logger
}

// NewProcessingService creates new processing service
func NewProcessingService(
	docRepo *postgres.DocumentRepository,
	extRepo *postgres.ExtractionRepository,
	auditRepo *postgres.AuditRepository,
	storage storage.Storage,
	ocrClient ocr.Client,
	llmClient llm.Client,
	bucket string,
	confidenceThreshold float64,
	autoApprove bool,
	maxAttempts int,
) *ProcessingService {
	return &ProcessingService{
		docRepo:           docRepo,
		extRepo:           extRepo,
		auditRepo:         auditRepo,
		storage:           storage,
		ocrClient:         ocrClient,
		llmClient:         llmClient,
		bucket:            bucket,
		confidenceThreshold: confidenceThreshold,
		autoApprove:       autoApprove,
		maxAttempts:       maxAttempts,
		logger:            zerolog.New(io.Discard).With().Timestamp().Logger(),
	}
}

// ProcessDocument processes a single document through the pipeline
func (s *ProcessingService) ProcessDocument(ctx context.Context, documentID string, attempt int) error {
	log := s.logger.With().Str("document_id", documentID).Int("attempt", attempt).Logger()
	log.Info().Msg("Starting document processing")

	// Get document
	doc, err := s.docRepo.GetByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("get document: %w", err)
	}

	// Update status to OCR in progress
	if err := s.docRepo.UpdateStatus(ctx, documentID, domain.StatusOCRInProgress, nil); err != nil {
		return fmt.Errorf("update status to ocr_in_progress: %w", err)
	}

	// Download file from storage
	fileData, err := s.storage.Download(ctx, s.bucket, doc.FileKey)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to download file: %v", err)
		s.docRepo.UpdateStatus(ctx, documentID, domain.StatusFailed, &errMsg)
		return fmt.Errorf("download file: %w", err)
	}
	defer fileData.Close()

	// Read file content
	fileBytes, err := io.ReadAll(fileData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to read file: %v", err)
		s.docRepo.UpdateStatus(ctx, documentID, domain.StatusFailed, &errMsg)
		return fmt.Errorf("read file: %w", err)
	}

	// Perform OCR
	log.Info().Msg("Performing OCR")
	ocrResult, err := s.ocrClient.Extract(ctx, doc.OriginalFilename, bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		errMsg := fmt.Sprintf("OCR service error: %v", err)
		s.docRepo.UpdateStatus(ctx, documentID, domain.StatusFailed, &errMsg)
		return fmt.Errorf("ocr extract: %w", err)
	}

	log.Info().
		Int("page_count", ocrResult.Metadata.PageCount).
		Int("text_length", len(ocrResult.FullText)).
		Bool("mock", ocrResult.Metadata.Mock).
		Msg("OCR completed")

	// Update status to extraction in progress
	if err := s.docRepo.UpdateStatus(ctx, documentID, domain.StatusExtractionInProgress, nil); err != nil {
		return fmt.Errorf("update status to extraction_in_progress: %w", err)
	}

	// Classify document type using LLM
	log.Info().Msg("Classifying document type")
	classification, err := s.llmClient.Classify(ctx, ocrResult.FullText)
	if err != nil {
		errMsg := fmt.Sprintf("LLM classification error: %v", err)
		s.docRepo.UpdateStatus(ctx, documentID, domain.StatusFailed, &errMsg)
		return fmt.Errorf("llm classify: %w", err)
	}

	docType := classification.DocumentType
	if docType == domain.DocumentTypeUnknown {
		docType = domain.DocumentTypeFreightInvoice // Default to invoice if unknown
	}

	log.Info().Str("document_type", string(docType)).Float64("confidence", classification.Confidence).Msg("Document classified")

	// Update document type
	if err := s.docRepo.SetDocumentType(ctx, documentID, docType); err != nil {
		log.Warn().Err(err).Msg("Failed to set document type")
	}

	// Extract data using LLM
	log.Info().Msg("Extracting data with LLM")
	extractionResult, err := s.llmClient.Extract(ctx, docType, ocrResult.FullText)
	if err != nil {
		errMsg := fmt.Sprintf("LLM extraction error: %v", err)
		s.docRepo.UpdateStatus(ctx, documentID, domain.StatusFailed, &errMsg)
		return fmt.Errorf("llm extract: %w", err)
	}

	log.Info().Str("document_type", string(extractionResult.DocumentType)).Msg("LLM extraction completed")

	// Validate extracted data
	log.Info().Msg("Validating extracted data")
	validationResult := s.validateExtraction(docType, extractionResult.Payload)

	// Determine final status
	finalStatus := domain.StatusNeedsReview
	if s.autoApprove && validationResult.Valid {
		// Check if all critical fields have sufficient confidence
		if s.checkConfidence(extractionResult.Confidence) {
			finalStatus = domain.StatusReviewed
		}
	}

	// Save extraction
	uuidDocID, _ := uuid.Parse(documentID)
	extraction := domain.NewDocumentExtraction(
		uuidDocID,
		extractionResult.DocumentType,
		extractionResult.Payload,
		extractionResult.Confidence,
		domain.ExtractionSourceLLM,
	)
	extraction.Validation = validationResult

	if err := s.extRepo.Create(ctx, extraction); err != nil {
		errMsg := fmt.Sprintf("Failed to save extraction: %v", err)
		s.docRepo.UpdateStatus(ctx, documentID, domain.StatusFailed, &errMsg)
		return fmt.Errorf("save extraction: %w", err)
	}

	// Update document status
	if err := s.docRepo.UpdateStatus(ctx, documentID, finalStatus, nil); err != nil {
		return fmt.Errorf("update final status: %w", err)
	}

	// Log audit event
	auditPayload := map[string]interface{}{
		"document_type": string(docType),
		"status":        string(finalStatus),
		"validation":    validationResult,
		"ocr_pages":     ocrResult.Metadata.PageCount,
	}
	if err := s.auditRepo.Create(ctx, &domain.AuditLog{
		ID:        uuid.New(),
		DocumentID: &uuidDocID,
		Action:    "document_processed",
		Payload:   auditPayload,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().Str("status", string(finalStatus)).Msg("Document processing completed")
	return nil
}

// validateExtraction validates extracted data based on document type
func (s *ProcessingService) validateExtraction(docType domain.DocumentType, payload map[string]interface{}) domain.ValidationResult {
	result := domain.ValidationResult{Valid: true}

	switch docType {
	case domain.DocumentTypeTransportWaybill:
		s.validateTransportWaybill(payload, &result)
	case domain.DocumentTypeFreightInvoice:
		s.validateFreightInvoice(payload, &result)
	}

	return result
}

// validateTransportWaybill validates transport waybill data
func (s *ProcessingService) validateTransportWaybill(payload map[string]interface{}, result *domain.ValidationResult) {
	// Validate shipper INN
	if shipper, ok := payload["shipper"].(map[string]interface{}); ok {
		if inn, ok := shipper["inn"].(string); ok && inn != "" {
			if err := validation.ValidateINN(inn); err != nil {
				result.AddError("shipper.inn", err.Error())
			}
		}
	}

	// Validate consignee INN
	if consignee, ok := payload["consignee"].(map[string]interface{}); ok {
		if inn, ok := consignee["inn"].(string); ok && inn != "" {
			if err := validation.ValidateINN(inn); err != nil {
				result.AddError("consignee.inn", err.Error())
			}
		}
	}

	// Validate carrier INN
	if carrier, ok := payload["carrier"].(map[string]interface{}); ok {
		if inn, ok := carrier["inn"].(string); ok && inn != "" {
			if err := validation.ValidateINN(inn); err != nil {
				result.AddError("carrier.inn", err.Error())
			}
		}
	}

	// Validate dates
	if docDate, ok := payload["document_date"].(string); ok && docDate != "" {
		if err := validation.ValidateDateNotFuture(docDate); err != nil {
			result.AddError("document_date", err.Error())
		}
	}

	// Validate amounts
	if deliveryCost, ok := payload["delivery_cost"].(map[string]interface{}); ok {
		s.validateAmounts(deliveryCost, "delivery_cost", result)
	}

	// Validate critical fields presence
	criticalFields := []string{"document_number", "document_date"}
	for _, field := range criticalFields {
		if val, exists := payload[field]; !exists || val == nil || val == "" {
			result.AddError(field, "Critical field is missing")
		}
	}
}

// validateFreightInvoice validates freight invoice data
func (s *ProcessingService) validateFreightInvoice(payload map[string]interface{}, result *domain.ValidationResult) {
	// Validate provider INN
	if provider, ok := payload["provider"].(map[string]interface{}); ok {
		if inn, ok := provider["inn"].(string); ok && inn != "" {
			if err := validation.ValidateINN(inn); err != nil {
				result.AddError("provider.inn", err.Error())
			}
		}
	}

	// Validate customer INN
	if customer, ok := payload["customer"].(map[string]interface{}); ok {
		if inn, ok := customer["inn"].(string); ok && inn != "" {
			if err := validation.ValidateINN(inn); err != nil {
				result.AddError("customer.inn", err.Error())
			}
		}
	}

	// Validate dates
	if docDate, ok := payload["document_date"].(string); ok && docDate != "" {
		if err := validation.ValidateDateNotFuture(docDate); err != nil {
			result.AddError("document_date", err.Error())
		}
	}

	// Validate amounts
	if amounts, ok := payload["amounts"].(map[string]interface{}); ok {
		s.validateAmounts(amounts, "amounts", result)
	}

	// Validate critical fields presence
	criticalFields := []string{"document_number", "document_date"}
	for _, field := range criticalFields {
		if val, exists := payload[field]; !exists || val == nil || val == "" {
			result.AddError(field, "Critical field is missing")
		}
	}
}

// validateAmounts validates monetary amounts
func (s *ProcessingService) validateAmounts(amounts map[string]interface{}, prefix string, result *domain.ValidationResult) {
	withoutVAT, hasWithoutVAT := amounts["amount_without_vat"].(string)
	vatAmount, hasVAT := amounts["vat_amount"].(string)
	total, hasTotal := amounts["total_with_vat"].(string)

	if hasWithoutVAT && withoutVAT != "" {
		if err := validation.ValidateAmount(withoutVAT); err != nil {
			result.AddError(fmt.Sprintf("%s.amount_without_vat", prefix), err.Error())
		}
	}

	if hasVAT && vatAmount != "" {
		if err := validation.ValidateAmount(vatAmount); err != nil {
			result.AddError(fmt.Sprintf("%s.vat_amount", prefix), err.Error())
		}
	}

	if hasTotal && total != "" {
		if err := validation.ValidateAmount(total); err != nil {
			result.AddError(fmt.Sprintf("%s.total_with_vat", prefix), err.Error())
		}
	}

	// Check sum consistency
	if hasWithoutVAT && hasVAT && hasTotal {
		if err := validation.ValidateAmountSum(withoutVAT, vatAmount, total); err != nil {
			result.AddError(fmt.Sprintf("%s.consistency", prefix), err.Error())
		}
	}
}

// checkConfidence checks if all critical fields meet confidence threshold
func (s *ProcessingService) checkConfidence(confidence map[string]float64) bool {
	for _, conf := range confidence {
		if conf < s.confidenceThreshold {
			return false
		}
	}
	return true
}
