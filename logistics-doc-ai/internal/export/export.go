package export

import (
"bytes"
"context"
"encoding/csv"
"encoding/json"
"fmt"
"io"
"strings"

"github.com/logistics-doc-ai/internal/domain"
)

// Service handles document export operations
type Service struct{}

// NewService creates a new export service
func NewService() *Service {
return &Service{}
}

// ExportToCSV exports documents to CSV format
func (s *Service) ExportToCSV(ctx context.Context, docs []*domain.Document, extractions map[string]*domain.DocumentExtraction) ([]byte, error) {
var buf bytes.Buffer
writer := csv.NewWriter(&buf)
writer.Comma = ';'

// Write header
header := []string{
"ID",
"Тип документа",
"Номер",
"Дата",
"Статус",
"Грузоотправитель",
"ИНН отправителя",
"Грузополучатель",
"ИНН получателя",
"Сумма с НДС",
"Валюта",
}
if err := writer.Write(header); err != nil {
return nil, fmt.Errorf("write header: %w", err)
}

// Write data rows
for _, doc := range docs {
extraction := extractions[doc.ID]

row := []string{
doc.ID,
string(doc.DocumentType),
getNestedString(extraction, "document_number"),
getNestedString(extraction, "document_date"),
string(doc.Status),
getNestedString(extraction, "shipper", "name"),
getNestedString(extraction, "shipper", "inn"),
getNestedString(extraction, "consignee", "name"),
getNestedString(extraction, "consignee", "inn"),
getNestedString(extraction, "delivery_cost", "total_with_vat"),
getNestedString(extraction, "delivery_cost", "currency"),
}

if err := writer.Write(row); err != nil {
return nil, fmt.Errorf("write row: %w", err)
}
}

writer.Flush()
if err := writer.Error(); err != nil {
return nil, fmt.Errorf("flush writer: %w", err)
}

return buf.Bytes(), nil
}

// ExportToJSON exports documents to JSON format
func (s *Service) ExportToJSON(ctx context.Context, docs []*domain.Document, extractions map[string]*domain.DocumentExtraction) ([]byte, error) {
type ExportDoc struct {
Document   *domain.Document           `json:"document"`
Extraction *domain.DocumentExtraction `json:"extraction,omitempty"`
}

exportDocs := make([]ExportDoc, 0, len(docs))
for _, doc := range docs {
exportDocs = append(exportDocs, ExportDoc{
Document:   doc,
Extraction: extractions[doc.ID],
})
}

data, err := json.MarshalIndent(exportDocs, "", "  ")
if err != nil {
return nil, fmt.Errorf("marshal JSON: %w", err)
}

return data, nil
}

func getNestedString(ext *domain.DocumentExtraction, path ...string) string {
if ext == nil || ext.Payload == nil {
return ""
}

current := ext.Payload
for i, key := range path {
if i == len(path)-1 {
if v, ok := current[key].(string); ok {
return v
}
return ""
}
if next, ok := current[key].(map[string]interface{}); ok {
current = next
} else {
return ""
}
}
return ""
}

// FormatFromFilename determines export format from filename extension
func FormatFromFilename(filename string) domain.ExportFormat {
if strings.HasSuffix(strings.ToLower(filename), ".csv") {
return domain.ExportFormatCSV
}
return domain.ExportFormatJSON
}
