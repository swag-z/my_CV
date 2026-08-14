from fastapi import FastAPI, File, UploadFile, HTTPException, Header
from pydantic import BaseModel, Field
from typing import Optional, List
import os

from app.core.config import settings
from app.services.ocr_service import OCRService

app = FastAPI(title="OCR Service", version="1.0.0")
ocr_service = OCRService()


class OCRResponse(BaseModel):
    full_text: str
    page_count: int = 1
    pages: Optional[List[dict]] = None
    metadata: Optional[dict] = None


@app.get("/healthz")
async def health_check():
    return {"status": "ok"}


@app.post("/v1/ocr", response_model=OCRResponse)
async def recognize(
    file: UploadFile = File(...),
    x_internal_token: Optional[str] = Header(None)
):
    # Validate internal token if configured
    if settings.ocr_internal_token and x_internal_token != settings.ocr_internal_token:
        raise HTTPException(status_code=401, detail="Invalid or missing X-Internal-Token")
    
    # Read file
    content = await file.read()
    if not content:
        raise HTTPException(status_code=400, detail="Empty file")
    
    try:
        result = ocr_service.recognize(file.filename, content)
        return OCRResponse(**result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"OCR failed: {str(e)}")
