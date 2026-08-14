from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    port: int = 8000
    ocr_mock: bool = True
    ocr_langs: str = "rus,eng"
    ocr_internal_token: Optional[str] = None
    
    class Config:
        env_file = ".env"


settings = Settings()
