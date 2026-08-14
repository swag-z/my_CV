package api

import (
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"strconv"
"strings"
"time"

"github.com/gofiber/fiber/v2"
"github.com/google/uuid"
"github.com/logistics-doc-ai/internal/domain"
"github.com/logistics-doc-ai/internal/service"
)

// Handlers contains HTTP handlers
type Handlers struct {
docService *service.DocumentService
}

// NewHandlers creates new handlers
func NewHandlers(docService *service.DocumentService) *Handlers {
return &Handlers{docService: docService}
}

// HealthCheck returns health status
func (h *Handlers) HealthCheck(c *fiber.Ctx) error {
return c.JSON(fiber.Map{"status": "ok"})
}

// UploadDocument handles document upload
func (h *Handlers) UploadDocument(c *fiber.Ctx) error {
file, err := c.FormFile("file")
if err != nil {
return c.Status(400).JSON(fiber.Map{"error": "Файл не найден"})
}

filename := file.Filename
mimeType := file.Header.Get("Content-Type")

// Validate MIME type
if !isValidMimeType(mimeType) {
return c.Status(400).JSON(fiber.Map{"error": "Недопустимый тип файла"})
}

f, err := file.Open()
if err != nil {
return c.Status(500).JSON(fiber.Map{"error": "Не удалось открыть файл"})
}
defer f.Close()

data, err := io.ReadAll(f)
if err != nil {
return c.Status(500).JSON(fiber.Map{"error": "Не удалось прочитать файл"})
}

ctx := context.Background()
doc, err := h.docService.UploadDocument(ctx, filename, mimeType, strings.NewReader(string(data)), int64(len(data)))
if err != nil {
return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Ошибка загрузки: %v", err)})
}

return c.Status(201).JSON(doc)
}

// ListDocuments lists documents
func (h *Handlers) ListDocuments(c *fiber.Ctx) error {
limit, _ := strconv.Atoi(c.Query("limit", "20"))
offset, _ := strconv.Atoi(c.Query("offset", "0"))
status := c.Query("status")
docType := c.Query("document_type")

var statusPtr, docTypePtr *string
if status != "" {
statusPtr = &status
}
if docType != "" {
docTypePtr = &docType
}

docs, total, err := h.docService.ListDocuments(c.Context(), limit, offset, statusPtr, docTypePtr)
if err != nil {
return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Ошибка получения списка: %v", err)})
}

return c.JSON(fiber.Map{
"documents": docs,
"total":     total,
})
}

// GetDocument gets a single document
func (h *Handlers) GetDocument(c *fiber.Ctx) error {
id := c.Params("id")

doc, err := h.docService.GetDocument(c.Context(), id)
if err != nil {
return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Ошибка получения: %v", err)})
}
if doc == nil {
return c.Status(404).JSON(fiber.Map{"error": "Документ не найден"})
}

extraction, _ := h.docService.GetLatestExtraction(c.Context(), id)

return c.JSON(fiber.Map{
"document":   doc,
"extraction": extraction,
})
}

// UpdateExtraction updates extraction data
func (h *Handlers) UpdateExtraction(c *fiber.Ctx) error {
documentID := c.Params("id")

var payload map[string]interface{}
if err := json.Unmarshal(c.Body(), &payload); err != nil {
return c.Status(400).JSON(fiber.Map{"error": "Невалидный JSON"})
}

ext := &domain.DocumentExtraction{
ID:           uuid.New().String(),
DocumentID:   documentID,
Payload:      payload,
Confidence:   map[string]float64{"user_edit": 1.0},
Source:       domain.SourceUser,
Version:      1,
CreatedAt:    time.Now().UTC(),
}

if err := h.docService.SaveExtraction(c.Context(), ext); err != nil {
return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Ошибка сохранения: %v", err)})
}

if err := h.docService.UpdateDocumentStatus(c.Context(), documentID, domain.StatusReviewed, nil); err != nil {
return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Ошибка обновления статуса: %v", err)})
}

return c.JSON(fiber.Map{"status": "ok"})
}

// ReprocessDocument requeues document for processing
func (h *Handlers) ReprocessDocument(c *fiber.Ctx) error {
id := c.Params("id")

doc, err := h.docService.GetDocument(c.Context(), id)
if err != nil || doc == nil {
return c.Status(404).JSON(fiber.Map{"error": "Документ не найден"})
}

if err := h.docService.UpdateDocumentStatus(c.Context(), id, domain.StatusQueued, nil); err != nil {
return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Ошибка обновления статуса: %v", err)})
}

return c.JSON(fiber.Map{"status": "queued"})
}

// ExportDocument exports document
func (h *Handlers) ExportDocument(c *fiber.Ctx) error {
format := c.Query("format", "json")

if format != "json" && format != "csv" {
return c.Status(400).JSON(fiber.Map{"error": "Недопустимый формат"})
}

return c.JSON(fiber.Map{
"status": "pending",
"format": format,
})
}

func isValidMimeType(mimeType string) bool {
validTypes := []string{
"application/pdf",
"image/jpeg",
"image/png",
"image/jpg",
}
for _, t := range validTypes {
if mimeType == t {
return true
}
}
return false
}
