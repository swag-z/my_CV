package validation

import (
"github.com/logistics-doc-ai/internal/domain"
)

// ValidateTransportWaybill validates extracted data for transport waybill
func ValidateTransportWaybill(payload map[string]interface{}) domain.ValidationResult {
result := domain.ValidationResult{
Valid:    true,
Errors:   []domain.ValidationIssue{},
Warnings: []domain.ValidationIssue{},
}

// Helper to get nested string value
getString := func(m map[string]interface{}, path ...string) string {
current := m
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

// Validate document number
docNum := getString(payload, "document_number")
if docNum == "" {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "document_number",
Message: "Номер документа обязателен",
})
result.Valid = false
}

// Validate document date
docDate := getString(payload, "document_date")
if docDate == "" {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "document_date",
Message: "Дата документа обязательна",
})
result.Valid = false
} else if err := ValidateDateNotFuture(docDate); err != nil {
result.Warnings = append(result.Warnings, domain.ValidationIssue{
Field:   "document_date",
Message: err.Error(),
})
}

// Validate shipper INN
shipperInn := getString(payload, "shipper", "inn")
if shipperInn != "" && !ValidateINN(shipperInn) {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "shipper.inn",
Message: "ИНН отправителя не проходит проверку",
})
result.Valid = false
}

// Validate consignee INN
consigneeInn := getString(payload, "consignee", "inn")
if consigneeInn != "" && !ValidateINN(consigneeInn) {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "consignee.inn",
Message: "ИНН получателя не проходит проверку",
})
result.Valid = false
}

// Validate carrier INN
carrierInn := getString(payload, "carrier", "inn")
if carrierInn != "" && !ValidateINN(carrierInn) {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "carrier.inn",
Message: "ИНН перевозчика не проходит проверку",
})
result.Valid = false
}

// Validate loading/unloading dates
loadingDate := getString(payload, "route", "loading_date")
unloadingDate := getString(payload, "route", "unloading_date")
if err := ValidateDateRange(loadingDate, unloadingDate); err != nil {
result.Warnings = append(result.Warnings, domain.ValidationIssue{
Field:   "route",
Message: err.Error(),
})
}

// Validate amounts consistency
withoutVAT := getString(payload, "delivery_cost", "amount_without_vat")
vat := getString(payload, "delivery_cost", "vat_amount")
total := getString(payload, "delivery_cost", "total_with_vat")

if errs := ValidateAmountsConsistency(withoutVAT, vat, total); len(errs) > 0 {
for _, e := range errs {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "delivery_cost",
Message: e,
})
}
result.Valid = false
}

return result
}

// ValidateFreightInvoice validates extracted data for freight invoice
func ValidateFreightInvoice(payload map[string]interface{}) domain.ValidationResult {
result := domain.ValidationResult{
Valid:    true,
Errors:   []domain.ValidationIssue{},
Warnings: []domain.ValidationIssue{},
}

getString := func(m map[string]interface{}, path ...string) string {
current := m
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

// Validate document number
docNum := getString(payload, "document_number")
if docNum == "" {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "document_number",
Message: "Номер документа обязателен",
})
result.Valid = false
}

// Validate document date
docDate := getString(payload, "document_date")
if docDate == "" {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "document_date",
Message: "Дата документа обязательна",
})
result.Valid = false
}

// Validate provider INN
providerInn := getString(payload, "provider", "inn")
if providerInn != "" && !ValidateINN(providerInn) {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "provider.inn",
Message: "ИНН поставщика не проходит проверку",
})
result.Valid = false
}

// Validate customer INN
customerInn := getString(payload, "customer", "inn")
if customerInn != "" && !ValidateINN(customerInn) {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "customer.inn",
Message: "ИНН покупателя не проходит проверку",
})
result.Valid = false
}

// Validate amounts consistency
withoutVAT := getString(payload, "amounts", "amount_without_vat")
vat := getString(payload, "amounts", "vat_amount")
total := getString(payload, "amounts", "total_with_vat")

if errs := ValidateAmountsConsistency(withoutVAT, vat, total); len(errs) > 0 {
for _, e := range errs {
result.Errors = append(result.Errors, domain.ValidationIssue{
Field:   "amounts",
Message: e,
})
}
result.Valid = false
}

return result
}

// Validate performs validation based on document type
func Validate(docType string, payload map[string]interface{}) domain.ValidationResult {
switch docType {
case "transport_waybill":
return ValidateTransportWaybill(payload)
case "freight_invoice":
return ValidateFreightInvoice(payload)
default:
return domain.ValidationResult{Valid: true}
}
}
