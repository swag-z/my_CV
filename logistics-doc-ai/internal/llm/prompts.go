package llm

import (
	"fmt"
	"strings"

	"github.com/logistics-doc-ai/internal/domain"
)

// buildClassificationPrompt builds prompt for document classification
func buildClassificationPrompt(text string) string {
	// Truncate text if too long
	if len(text) > 5000 {
		text = text[:5000] + "..."
	}

	return fmt.Sprintf(`Ты — классификатор документов.
Определи тип документа по распознанному тексту.
Возможные типы:
- transport_waybill: транспортная накладная, ТТН, товарно-транспортная накладная;
- freight_invoice: счёт на оплату, УПД, акт выполненных работ, связанный с транспортной услугой;
- unknown: если тип невозможно определить.

Верни только валидный JSON без markdown и пояснений:
{
  "document_type": "transport_waybill",
  "confidence": 0.97
}

Текст документа:
%s`, text)
}

// buildExtractionPrompt builds prompt for data extraction based on document type
func buildExtractionPrompt(docType domain.DocumentType, text string) string {
	// Truncate text if too long
	if len(text) > 8000 {
		text = text[:8000] + "..."
	}

	switch docType {
	case domain.DocumentTypeTransportWaybill:
		return buildTransportWaybillPrompt(text)
	case domain.DocumentTypeFreightInvoice:
		return buildFreightInvoicePrompt(text)
	default:
		return buildFreightInvoicePrompt(text) // Default to invoice
	}
}

// buildTransportWaybillPrompt builds prompt for transport waybill extraction
func buildTransportWaybillPrompt(text string) string {
	return fmt.Sprintf(`Ты — модуль извлечения данных из транспортных накладных и ТТН.
Ниже передан текст, полученный OCR.
Извлеки данные строго в JSON по заданной схеме.
Не выдумывай значения.
Если поле отсутствует или нечитаемо — верни null.
Даты возвращай в формате YYYY-MM-DD.
Денежные суммы возвращай как строки с двумя знаками после запятой, например "12345.67".
Вес, объём и паллеты возвращай как строки.
Для каждого ключевого поля укажи confidence от 0 до 1.
Верни только JSON без markdown, без пояснений и без лишних полей.

Schema:
{
  "document_type": "transport_waybill",
  "document_number": null,
  "document_date": null,
  "shipper": {
    "name": null,
    "inn": null,
    "kpp": null,
    "address": null
  },
  "consignee": {
    "name": null,
    "inn": null,
    "kpp": null,
    "address": null
  },
  "carrier": {
    "name": null,
    "inn": null,
    "kpp": null
  },
  "route": {
    "loading_address": null,
    "loading_date": null,
    "unloading_address": null,
    "unloading_date": null
  },
  "vehicle": {
    "plate": null,
    "model": null
  },
  "driver": {
    "name": null
  },
  "cargo": {
    "description": null,
    "weight_kg": null,
    "volume_m3": null,
    "pallets": null
  },
  "delivery_cost": {
    "currency": null,
    "amount_without_vat": null,
    "vat_amount": null,
    "total_with_vat": null
  },
  "confidence": {}
}

Текст документа:
%s`, text)
}

// buildFreightInvoicePrompt builds prompt for freight invoice extraction
func buildFreightInvoicePrompt(text string) string {
	return fmt.Sprintf(`Ты — модуль извлечения данных из счетов, УПД и актов за транспортные услуги.
Ниже передан текст, полученный OCR.
Извлеки данные строго в JSON по заданной схеме.
Не выдумывай значения.
Если поле отсутствует или нечитаемо — верни null.
Даты возвращай в формате YYYY-MM-DD.
Денежные суммы возвращай как строки с двумя знаками после запятой.
Для каждого ключевого поля укажи confidence от 0 до 1.
Верни только JSON без markdown, без пояснений и без лишних полей.

Schema:
{
  "document_type": "freight_invoice",
  "document_number": null,
  "document_date": null,
  "provider": {
    "name": null,
    "inn": null,
    "kpp": null
  },
  "customer": {
    "name": null,
    "inn": null,
    "kpp": null
  },
  "service": {
    "description": null,
    "contract_number": null,
    "order_reference": null
  },
  "amounts": {
    "currency": null,
    "amount_without_vat": null,
    "vat_amount": null,
    "total_with_vat": null
  },
  "payment_purpose": null,
  "confidence": {}
}

Текст документа:
%s`, text)
}

// GetDocumentTypeKeywords returns keywords for document type detection
func GetDocumentTypeKeywords(docType domain.DocumentType) []string {
	switch docType {
	case domain.DocumentTypeTransportWaybill:
		return []string{"транспортная накладная", "ттн", "товарно-транспортная", "тн"}
	case domain.DocumentTypeFreightInvoice:
		return []string{"счёт", "упд", "акт", "invoice", "счет"}
	default:
		return []string{}
	}
}

// DetectDocTypeFromFilename tries to detect document type from filename
func DetectDocTypeFromFilename(filename string) domain.DocumentType {
	lower := strings.ToLower(filename)
	
	transportKeywords := []string{"transport", "waybill", "ттн", "тн", "накладная"}
	invoiceKeywords := []string{"invoice", "счет", "счёт", "упд", "акт"}
	
	for _, keyword := range transportKeywords {
		if strings.Contains(lower, keyword) {
			return domain.DocumentTypeTransportWaybill
		}
	}
	
	for _, keyword := range invoiceKeywords {
		if strings.Contains(lower, keyword) {
			return domain.DocumentTypeFreightInvoice
		}
	}
	
	return domain.DocumentTypeUnknown
}
