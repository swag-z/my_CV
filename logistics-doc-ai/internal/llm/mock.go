package llm

import (
"context"
"strings"
"time"

"github.com/logistics-doc-ai/internal/domain"
)

// MockClient implements LLM Client interface with deterministic mock data
type MockClient struct{}

// NewMockClient creates a new mock LLM client
func NewMockClient() *MockClient {
return &MockClient{}
}

// Classify classifies document type based on text
func (c *MockClient) Classify(ctx context.Context, text string) (*ClassificationResult, error) {
textLower := strings.ToLower(text)

docType := domain.DocumentTypeUnknown
confidence := 0.5

if strings.Contains(textLower, "транспортная накладная") || 
   strings.Contains(textLower, "ттн") ||
   strings.Contains(textLower, "товарно-транспортная") {
docType = domain.DocumentTypeTransportWaybill
confidence = 0.95
} else if strings.Contains(textLower, "счёт") ||
          strings.Contains(textLower, "упд") ||
          strings.Contains(textLower, "акт") ||
          strings.Contains(textLower, "invoice") {
docType = domain.DocumentTypeFreightInvoice
confidence = 0.95
}

return &ClassificationResult{
DocumentType: docType,
Confidence:   confidence,
}, nil
}

// Extract extracts structured data from text
func (c *MockClient) Extract(ctx context.Context, docType domain.DocumentType, text string) (*ExtractionResult, error) {
var payload map[string]interface{}
var confidence map[string]float64

if docType == domain.DocumentTypeTransportWaybill {
payload = mockTransportWaybillData()
confidence = mockTransportWaybillConfidence()
} else {
payload = mockFreightInvoiceData()
confidence = mockFreightInvoiceConfidence()
}

return &ExtractionResult{
DocumentType: docType,
Payload:      payload,
Confidence:   confidence,
Raw:          "",
}, nil
}

func mockTransportWaybillData() map[string]interface{} {
return map[string]interface{}{
"document_type":   "transport_waybill",
"document_number": "12345",
"document_date":   "2026-08-15",
"shipper": map[string]interface{}{
"name":    "ООО Отправитель",
"inn":     "7701234567",
"kpp":     "770101001",
"address": "г. Москва, ул. Складская, д. 1",
},
"consignee": map[string]interface{}{
"name":    "ООО Получатель",
"inn":     "7709876543",
"kpp":     "770901001",
"address": "г. Екатеринбург, ул. Промышленная, д. 5",
},
"carrier": map[string]interface{}{
"name": "ООО Перевозчик",
"inn":  "7705554433",
"kpp":  "770501001",
},
"route": map[string]interface{}{
"loading_address":   "г. Москва, ул. Складская, д. 1",
"loading_date":      "2026-08-15",
"unloading_address": "г. Екатеринбург, ул. Промышленная, д. 5",
"unloading_date":    "2026-08-17",
},
"vehicle": map[string]interface{}{
"plate": "А123ВС777",
"model": "Volvo FH",
},
"driver": map[string]interface{}{
"name": "Иванов Иван Иванович",
},
"cargo": map[string]interface{}{
"description": "Мебель офисная",
"weight_kg":   "1500",
"volume_m3":   "12.5",
"pallets":     "8",
},
"delivery_cost": map[string]interface{}{
"currency":           "RUB",
"amount_without_vat": "50000.00",
"vat_amount":         "10000.00",
"total_with_vat":     "60000.00",
},
}
}

func mockTransportWaybillConfidence() map[string]float64 {
return map[string]float64{
"document_number":      0.98,
"document_date":        0.97,
"shipper_inn":          0.99,
"consignee_inn":        0.98,
"carrier_inn":          0.98,
"loading_address":      0.93,
"unloading_address":    0.93,
"vehicle_plate":        0.91,
"cargo_weight_kg":      0.90,
"delivery_cost_total":  0.95,
}
}

func mockFreightInvoiceData() map[string]interface{} {
return map[string]interface{}{
"document_type":   "freight_invoice",
"document_number": "789",
"document_date":   "2026-08-15",
"provider": map[string]interface{}{
"name": "ООО Перевозчик",
"inn":  "7705554433",
"kpp":  "770501001",
},
"customer": map[string]interface{}{
"name": "ООО Заказчик",
"inn":  "7701234567",
"kpp":  "770101001",
},
"service": map[string]interface{}{
"description":     "Транспортные услуги по маршруту Москва - Екатеринбург",
"contract_number": "Д-123",
"order_reference": "Перевозка № 456",
},
"amounts": map[string]interface{}{
"currency":           "RUB",
"amount_without_vat": "50000.00",
"vat_amount":         "10000.00",
"total_with_vat":     "60000.00",
},
"payment_purpose": "Оплата за транспортные услуги по договору Д-123",
}
}

func mockFreightInvoiceConfidence() map[string]float64 {
return map[string]float64{
"document_number":    0.98,
"document_date":      0.97,
"provider_inn":       0.99,
"customer_inn":       0.98,
"service_description": 0.92,
"amounts_total":      0.96,
}
}

var _ Client = (*MockClient)(nil)
