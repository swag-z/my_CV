package ocr

import (
"context"
"io"
"strings"
"time"
)

// MockClient implements OCR Client with mock data
type MockClient struct{}

// NewMockClient creates a new mock OCR client
func NewMockClient() *MockClient {
return &MockClient{}
}

// Extract returns mock OCR result based on filename
func (c *MockClient) Extract(ctx context.Context, filename string, reader io.Reader, size int64) (*OCRResult, error) {
filenameLower := strings.ToLower(filename)

if strings.Contains(filenameLower, "transport") || strings.Contains(filenameLower, "ттн") || strings.Contains(filenameLower, "накладная") {
return mockTransportWaybillOCR(), nil
} else if strings.Contains(filenameLower, "invoice") || strings.Contains(filenameLower, "счёт") || strings.Contains(filenameLower, "упд") || strings.Contains(filenameLower, "акт") {
return mockFreightInvoiceOCR(), nil
}

// Default mock - read from reader
fileData, err := io.ReadAll(reader)
if err != nil {
return nil, err
}

return &OCRResult{
FullText:  "Документ не распознан\nТекст извлечён из файла: " + filename + "\nРазмер: " + string(rune(len(fileData))),
PageCount: 1,
Metadata: OCRMetadata{
Engine:      "mock",
PageCount:   1,
ProcessedAt: time.Now().UTC(),
Mock:        true,
},
}, nil
}

func mockTransportWaybillOCR() *OCRResult {
return &OCRResult{
FullText: `ТРАНСПОРТНАЯ НАКЛАДНАЯ № 12345 от 15.08.2026
Отправитель: ООО Отправитель, ИНН 7701234567, КПП 770101001
Адрес отправителя: г. Москва, ул. Складская, д. 1
Получатель: ООО Получатель, ИНН 7709876543, КПП 770901001
Адрес получателя: г. Екатеринбург, ул. Промышленная, д. 5
Перевозчик: ООО Перевозчик, ИНН 7705554433, КПП 770501001
Адрес погрузки: г. Москва, ул. Складская, д. 1
Адрес выгрузки: г. Екатеринбург, ул. Промышленная, д. 5
Дата погрузки: 15.08.2026
Дата выгрузки: 17.08.2026
Автомобиль: Volvo FH, госномер А123ВС777
Водитель: Иванов Иван Иванович
Груз: Мебель офисная
Вес: 1500 кг
Объем: 12.5 м3
Паллеты: 8
Стоимость доставки без НДС: 50000.00 руб.
НДС: 10000.00 руб.
Итого с НДС: 60000.00 руб.`,
PageCount: 1,
Pages: []PageResult{
{
PageNumber: 1,
Text:       "ТРАНСПОРТНАЯ НАКЛАДНАЯ № 12345 от 15.08.2026",
Confidence: 0.95,
},
},
Metadata: OCRMetadata{
Engine:      "mock",
PageCount:   1,
ProcessedAt: time.Now().UTC(),
Mock:        true,
},
}
}

func mockFreightInvoiceOCR() *OCRResult {
return &OCRResult{
FullText: `СЧЁТ НА ОПЛАТУ № 789 от 15.08.2026
Поставщик: ООО Перевозчик, ИНН 7705554433, КПП 770501001
Покупатель: ООО Заказчик, ИНН 7701234567, КПП 770101001
Услуга: Транспортные услуги по маршруту Москва - Екатеринбург
Договор: Д-123
Заявка: Перевозка № 456
Сумма без НДС: 50000.00 руб.
НДС: 10000.00 руб.
Итого с НДС: 60000.00 руб.
Назначение платежа: Оплата за транспортные услуги по договору Д-123`,
PageCount: 1,
Pages: []PageResult{
{
PageNumber: 1,
Text:       "СЧЁТ НА ОПЛАТУ № 789 от 15.08.2026",
Confidence: 0.95,
},
},
Metadata: OCRMetadata{
Engine:      "mock",
PageCount:   1,
ProcessedAt: time.Now().UTC(),
Mock:        true,
},
}
}

var _ Client = (*MockClient)(nil)
