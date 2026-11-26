from functools import lru_cache
from typing import Literal

from pydantic import Field, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="BRANDSHAKE_AI_", env_file=".env", extra="ignore")

    provider: Literal["fake", "gemini"] = "fake"

    gemini_api_key: str | None = Field(default=None, validation_alias="GEMINI_API_KEY")
    # Model names change often; set them explicitly (see ai.google.dev/models).
    gemini_model_fast: str | None = None
    gemini_model_smart: str | None = None

    # Shared request budget across all features (matches provider quota).
    requests_per_minute: int = 10
    max_retries: int = 3
    request_timeout_s: float = 60.0
    cache_size: int = 512

    @model_validator(mode="after")
    def _check_gemini(self) -> "Settings":
        if self.provider == "gemini":
            missing = [
                name
                for name, value in (
                    ("GEMINI_API_KEY", self.gemini_api_key),
                    ("BRANDSHAKE_AI_GEMINI_MODEL_FAST", self.gemini_model_fast),
                    ("BRANDSHAKE_AI_GEMINI_MODEL_SMART", self.gemini_model_smart),
                )
                if not value
            ]
            if missing:
                raise ValueError(f"provider=gemini requires: {', '.join(missing)}")
        return self


@lru_cache
def get_settings() -> Settings:
    return Settings()
