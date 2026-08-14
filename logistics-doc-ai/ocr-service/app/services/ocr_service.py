import os
from typing import Dict, Any


class OCRService:
    def __init__(self):
        self.mock_mode = os.getenv("OCR_MOCK", "true").lower() == "true"
        self.langs = os.getenv("OCR_LANGS", "rus,eng")
    
    def recognize(self, filename: str, file_data: bytes) -> Dict[str, Any]:
        if self.mock_mode:
            return self._mock_recognize(filename, file_data)
        else:
            return self._tesseract_recognize(file_data)
    
    def _mock_recognize(self, filename: str, file_data: bytes) -> Dict[str, Any]:
        filename_lower = filename.lower()
        
        if "transport" in filename_lower or "ттн" in filename_lower:
            return {
                "full_text": """ТРАНСПОРТНАЯ НАКЛАДНАЯ № 12345
Дата: 15.08.2026

Грузоотправитель: ООО Отправитель
ИНН: 7701234567 КПП: 770101001
Адрес: г. Москва, ул. Складская, д. 1

Грузополучатель: ООО Получатель
ИНН: 7709876543 КПП: 770901001
Адрес: г. Екатеринбург, ул. Промышленная, д. 5

Перевозчик: ООО Перевозчик
ИНН: 7705554433 КПП: 770501001

Маршрут:
Погрузка: г. Москва, ул. Складская, д. 1, 15.08.2026
Выгрузка: г. Екатеринбург, ул. Промышленная, д. 5, 17.08.2026

Автомобиль: Volvo FH А123ВС777
Водитель: Иванов Иван Иванович

Груз: Мебель офисная
Вес: 1500 кг
Объём: 12.5 м3

Стоимость доставки:
Без НДС: 50000.00 руб.
НДС: 10000.00 руб.
С НДС: 60000.00 руб.""",
                "page_count": 1,
                "metadata": {"document_type": "transport_waybill"}
            }
        elif "invoice" in filename_lower or "счёт" in filename_lower or "упд" in filename_lower:
            return {
                "full_text": """СЧЁТ-ФАКТУРА № 789
Дата: 15.08.2026

Поставщик: ООО Перевозчик
ИНН: 7705554433 КПП: 770501001

Покупатель: ООО Заказчик
ИНН: 7701234567 КПП: 770101001

Услуга: Транспортные услуги по маршруту Москва - Екатеринбург
Договор: Д-123

Стоимость:
Без НДС: 50000.00 руб.
НДС: 10000.00 руб.
С НДС: 60000.00 руб.""",
                "page_count": 1,
                "metadata": {"document_type": "freight_invoice"}
            }
        else:
            return {
                "full_text": f"Документ не распознан\nТекст из файла: {filename}",
                "page_count": 1
            }
    
    def _tesseract_recognize(self, file_data: bytes) -> Dict[str, Any]:
        try:
            from PIL import Image
            import pytesseract
            import io
            
            img = Image.open(io.BytesIO(file_data))
            text = pytesseract.image_to_string(img, lang=self.langs.replace(",", "+"))
            
            return {
                "full_text": text,
                "page_count": 1
            }
        except Exception as e:
            raise RuntimeError(f"Tesseract OCR failed: {str(e)}")
