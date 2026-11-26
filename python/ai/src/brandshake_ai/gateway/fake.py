"""Deterministic offline provider for local development and tests."""

from collections.abc import Callable

from pydantic import BaseModel

from .base import LLMRequest, RawResult, Usage

Responder = Callable[[LLMRequest], BaseModel]


class FakeProvider:
    """Answers each schema with a registered responder function.

    Responders are plain Python (regexes, fixtures), so results are free,
    instant and reproducible.
    """

    name = "fake"

    def __init__(self, responders: dict[type[BaseModel], Responder] | None = None):
        self._responders = dict(responders or {})
        self.calls: list[LLMRequest] = []

    def register(self, schema: type[BaseModel], responder: Responder) -> None:
        self._responders[schema] = responder

    async def generate_json(self, req: LLMRequest) -> RawResult:
        self.calls.append(req)
        responder = self._responders.get(req.schema)
        if responder is None:
            raise LookupError(f"FakeProvider has no responder for {req.schema.__name__}")
        out = responder(req)
        return RawResult(
            text=out.model_dump_json(),
            model="fake",
            usage=Usage(input_tokens=len(req.prompt) // 4, output_tokens=0),
        )
