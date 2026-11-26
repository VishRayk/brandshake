"""Gemini provider (google-genai SDK)."""

from google import genai
from google.genai import errors, types

from .base import LLMRequest, ProviderError, RawResult, Usage


class GeminiProvider:
    name = "gemini"

    def __init__(self, api_key: str, model_fast: str, model_smart: str, timeout_s: float):
        self._client = genai.Client(
            api_key=api_key,
            http_options=types.HttpOptions(timeout=int(timeout_s * 1000)),
        )
        self._models = {"fast": model_fast, "smart": model_smart}

    async def generate_json(self, req: LLMRequest) -> RawResult:
        model = self._models[req.tier]
        try:
            resp = await self._client.aio.models.generate_content(
                model=model,
                contents=req.prompt,
                config=types.GenerateContentConfig(
                    system_instruction=req.system,
                    response_mime_type="application/json",
                    response_json_schema=req.schema.model_json_schema(),
                    temperature=0.0,
                ),
            )
        except errors.APIError as e:
            # 429 (quota) and 5xx are worth retrying; other 4xx are not.
            retryable = e.code == 429 or e.code >= 500
            raise ProviderError(f"gemini: {e}", retryable=retryable, status=e.code) from e

        text = resp.text
        if not text:
            raise ProviderError("gemini: empty response", retryable=True)

        meta = resp.usage_metadata
        usage = Usage(
            input_tokens=(meta.prompt_token_count or 0) if meta else 0,
            output_tokens=(meta.candidates_token_count or 0) if meta else 0,
        )
        return RawResult(text=text, model=model, usage=usage)
