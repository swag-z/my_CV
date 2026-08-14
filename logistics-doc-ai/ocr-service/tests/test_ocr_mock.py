import pytest
from fastapi.testclient import TestClient
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from app.main import app

client = TestClient(app)


def test_ocr_mock_transport():
    """Test OCR mock mode returns transport waybill mock"""
    # Create a mock file
    files = {"file": ("sample_transport_waybill.pdf", b"mock pdf content", "application/pdf")}
    response = client.post("/v1/ocr", files=files)
    
    assert response.status_code == 200
    data = response.json()
    assert "full_text" in data
    assert "ТРАНСПОРТНАЯ НАКЛАДНАЯ" in data["full_text"]
    assert data.get("metadata", {}).get("document_type") == "transport_waybill"


def test_ocr_mock_invoice():
    """Test OCR mock mode returns freight invoice mock"""
    files = {"file": ("sample_freight_invoice.pdf", b"mock pdf content", "application/pdf")}
    response = client.post("/v1/ocr", files=files)
    
    assert response.status_code == 200
    data = response.json()
    assert "full_text" in data
    assert "СЧЁТ-ФАКТУРА" in data["full_text"]
    assert data.get("metadata", {}).get("document_type") == "freight_invoice"
