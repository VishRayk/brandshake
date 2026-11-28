"""HTTP API for the Brandshake AI service."""

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, Field

from brandshake_ai.config import Settings, get_settings
from brandshake_ai.features.deals import DraftRequest, DraftResult, draft_deal
from brandshake_ai.features.terms import TermsResult, extract_terms
from brandshake_ai.gateway import GatewayError, LLMGateway, LLMProvider
from brandshake_ai.gateway.fake_responders import default_fake_provider
from brandshake_ai.prompts.extract_terms import ChatMessage

log = logging.getLogger("brandshake_ai")


def build_provider(s: Settings) -> LLMProvider:
    if s.provider == "gemini":
        from brandshake_ai.gateway.gemini import GeminiProvider

        return GeminiProvider(
            api_key=s.gemini_api_key or "",
            model_fast=s.gemini_model_fast or "",
            model_smart=s.gemini_model_smart or "",
            timeout_s=s.request_timeout_s,
        )
    return default_fake_provider()


def create_app(gateway: LLMGateway | None = None) -> FastAPI:
    @asynccontextmanager
    async def lifespan(app: FastAPI):
        if gateway is None:
            s = get_settings()
            app.state.gateway = LLMGateway(
                build_provider(s),
                requests_per_minute=s.requests_per_minute,
                max_retries=s.max_retries,
                cache_size=s.cache_size,
            )
        else:
            app.state.gateway = gateway
        log.info("ai service ready", extra={"provider": app.state.gateway.provider.name})
        yield

    app = FastAPI(title="Brandshake AI", version="0.1.0", lifespan=lifespan)

    @app.get("/healthz")
    async def healthz(request: Request) -> dict:
        return {"status": "ok", "provider": request.app.state.gateway.provider.name}

    class ExtractTermsRequest(BaseModel):
        messages: list[ChatMessage] = Field(min_length=1, max_length=200)

    @app.post("/v1/terms/extract", response_model=TermsResult)
    async def terms_extract(body: ExtractTermsRequest, request: Request) -> TermsResult:
        try:
            return await extract_terms(request.app.state.gateway, body.messages)
        except GatewayError as e:
            log.warning("terms extraction failed", exc_info=e)
            raise HTTPException(
                status_code=503,
                detail="Automatic term extraction is unavailable right now. Please fill in the terms manually.",
            ) from e

    @app.post("/v1/deals/draft", response_model=DraftResult)
    async def deals_draft(body: DraftRequest, request: Request) -> DraftResult:
        try:
            return await draft_deal(request.app.state.gateway, body)
        except GatewayError as e:
            log.warning("deal draft failed", exc_info=e)
            raise HTTPException(
                status_code=503,
                detail="Automatic deal drafting is unavailable right now. Please fill in the deal manually.",
            ) from e

    @app.get("/v1/metrics")
    async def metrics(request: Request) -> dict[str, int]:
        return dict(request.app.state.gateway.metrics)

    return app


app = create_app()
