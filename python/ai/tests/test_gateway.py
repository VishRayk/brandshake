import pytest
from pydantic import BaseModel

from brandshake_ai.gateway import (
    FakeProvider,
    GatewayError,
    LLMGateway,
    LLMRequest,
    ProviderError,
    RateLimiter,
    RawResult,
    Usage,
)


class Answer(BaseModel):
    value: int


def req(prompt: str = "q") -> LLMRequest:
    return LLMRequest(feature="test", prompt_version="v1", system="s", prompt=prompt, schema=Answer)


class FlakyProvider:
    """Fails with the given errors first, then returns `final` text."""

    name = "flaky"

    def __init__(self, failures: list[Exception], final: str = '{"value": 7}'):
        self.failures = list(failures)
        self.final = final
        self.calls = 0

    async def generate_json(self, r: LLMRequest) -> RawResult:
        self.calls += 1
        if self.failures:
            raise self.failures.pop(0)
        return RawResult(text=self.final, model="m", usage=Usage())


def gateway(provider, **kw) -> LLMGateway:
    return LLMGateway(provider, requests_per_minute=1000, base_backoff_s=0, **kw)


async def test_caches_identical_requests():
    fake = FakeProvider({Answer: lambda r: Answer(value=len(r.prompt))})
    gw = gateway(fake)

    first = await gw.run(req("hello"), Answer)
    second = await gw.run(req("hello"), Answer)
    third = await gw.run(req("different"), Answer)

    assert first.output.value == 5 and not first.cached
    assert second.cached and second.output == first.output
    assert third.output.value == 9
    assert len(fake.calls) == 2


async def test_retries_retryable_errors():
    p = FlakyProvider([ProviderError("429", retryable=True), ProviderError("503", retryable=True)])
    res = await gateway(p, max_retries=3).run(req(), Answer)
    assert res.output.value == 7
    assert p.calls == 3


async def test_does_not_retry_permanent_errors():
    p = FlakyProvider([ProviderError("400", retryable=False)])
    with pytest.raises(GatewayError):
        await gateway(p, max_retries=3).run(req(), Answer)
    assert p.calls == 1


async def test_invalid_output_is_retried_then_fails():
    p = FlakyProvider([], final='{"value": "not a number"}')
    gw = gateway(p, max_retries=2)
    with pytest.raises(GatewayError):
        await gw.run(req(), Answer)
    assert p.calls == 3
    assert gw.metrics["test.invalid_output"] == 3


async def test_rate_limiter_waits_when_empty():
    now = [0.0]
    limiter = RateLimiter(rate=2, per=60, clock=lambda: now[0])

    assert await limiter.acquire() == 0
    assert await limiter.acquire() == 0
    assert limiter.tokens < 1

    now[0] = 30.0  # half a minute later: one token refilled
    assert await limiter.acquire() == 0
