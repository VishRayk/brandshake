"""Provider-agnostic types for structured LLM calls."""

from dataclasses import dataclass
from typing import Literal, Protocol

from pydantic import BaseModel

Tier = Literal["fast", "smart"]


@dataclass(frozen=True)
class LLMRequest:
    feature: str  # e.g. "extract_terms" (for metrics and budgets)
    prompt_version: str
    system: str
    prompt: str
    schema: type[BaseModel]
    tier: Tier = "smart"


@dataclass(frozen=True)
class Usage:
    input_tokens: int = 0
    output_tokens: int = 0


@dataclass(frozen=True)
class RawResult:
    """What a provider returns: unvalidated JSON text plus usage."""

    text: str
    model: str
    usage: Usage


class ProviderError(Exception):
    """A provider call failed. `retryable` says whether retrying may help."""

    def __init__(self, message: str, *, retryable: bool, status: int | None = None):
        super().__init__(message)
        self.retryable = retryable
        self.status = status


class LLMProvider(Protocol):
    name: str

    async def generate_json(self, req: LLMRequest) -> RawResult:
        """Return JSON text that should match `req.schema`."""
        ...
