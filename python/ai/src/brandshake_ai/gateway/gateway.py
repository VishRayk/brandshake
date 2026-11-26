"""The single entry point for every LLM call in Brandshake.

Adds, on top of any provider: a shared rate limit, result caching,
retries with backoff and jitter, schema validation and metrics.
"""

import asyncio
import hashlib
import logging
import random
import time
from collections import Counter, OrderedDict
from dataclasses import dataclass, replace

from pydantic import BaseModel, ValidationError

from .base import LLMProvider, LLMRequest, ProviderError, Usage

log = logging.getLogger(__name__)


class RateLimiter:
    """Async token bucket: `rate` requests per `per` seconds, shared by callers."""

    def __init__(self, rate: int, per: float = 60.0, clock=time.monotonic):
        self.capacity = float(rate)
        self.tokens = float(rate)
        self.refill_per_s = rate / per
        self._clock = clock
        self._last = clock()
        self._lock = asyncio.Lock()

    def _refill(self) -> None:
        now = self._clock()
        self.tokens = min(self.capacity, self.tokens + (now - self._last) * self.refill_per_s)
        self._last = now

    async def acquire(self) -> float:
        """Wait for a token. Returns seconds spent waiting."""
        waited = 0.0
        async with self._lock:
            while True:
                self._refill()
                if self.tokens >= 1:
                    self.tokens -= 1
                    return waited
                delay = (1 - self.tokens) / self.refill_per_s
                waited += delay
                await asyncio.sleep(delay)


class GatewayError(Exception):
    """The call could not produce a valid result; callers should fall back."""


@dataclass(frozen=True)
class LLMResult[T: BaseModel]:
    output: T
    provider: str
    model: str
    prompt_version: str
    usage: Usage
    cached: bool
    latency_ms: int


class LLMGateway:
    def __init__(
        self,
        provider: LLMProvider,
        *,
        requests_per_minute: int = 10,
        max_retries: int = 3,
        cache_size: int = 512,
        base_backoff_s: float = 1.0,
    ):
        self.provider = provider
        self.limiter = RateLimiter(requests_per_minute)
        self.max_retries = max_retries
        self.base_backoff_s = base_backoff_s
        self._cache: OrderedDict[str, LLMResult] = OrderedDict()
        self._cache_size = cache_size
        self.metrics: Counter[str] = Counter()

    @staticmethod
    def cache_key(req: LLMRequest) -> str:
        h = hashlib.sha256()
        for part in (req.feature, req.prompt_version, req.tier, req.schema.__name__, req.system, req.prompt):
            h.update(part.encode())
            h.update(b"\x00")
        return h.hexdigest()

    async def run[T: BaseModel](self, req: LLMRequest, schema: type[T]) -> LLMResult[T]:
        key = self.cache_key(req)
        if (hit := self._cache.get(key)) is not None:
            self._cache.move_to_end(key)
            self.metrics[f"{req.feature}.cache_hit"] += 1
            return replace(hit, cached=True)

        start = time.perf_counter()
        last_err: Exception | None = None

        for attempt in range(self.max_retries + 1):
            if attempt:
                delay = self.base_backoff_s * 2 ** (attempt - 1) * (0.5 + random.random())
                await asyncio.sleep(delay)

            await self.limiter.acquire()
            self.metrics[f"{req.feature}.requests"] += 1
            try:
                raw = await self.provider.generate_json(req)
                output = schema.model_validate_json(raw.text)
            except ProviderError as e:
                last_err = e
                self.metrics[f"{req.feature}.provider_error"] += 1
                log.warning("llm provider error", extra={"feature": req.feature, "attempt": attempt, "err": str(e)})
                if not e.retryable:
                    break
                continue
            except ValidationError as e:
                # The model returned JSON that does not match the schema: retry once more.
                last_err = e
                self.metrics[f"{req.feature}.invalid_output"] += 1
                continue

            result = LLMResult(
                output=output,
                provider=self.provider.name,
                model=raw.model,
                prompt_version=req.prompt_version,
                usage=raw.usage,
                cached=False,
                latency_ms=int((time.perf_counter() - start) * 1000),
            )
            self._remember(key, result)
            self.metrics[f"{req.feature}.success"] += 1
            return result

        self.metrics[f"{req.feature}.failed"] += 1
        raise GatewayError(f"{req.feature}: no valid result after retries") from last_err

    def _remember(self, key: str, result: LLMResult) -> None:
        self._cache[key] = result
        self._cache.move_to_end(key)
        while len(self._cache) > self._cache_size:
            self._cache.popitem(last=False)
