# brandshake-ai

Python service for Brandshake's AI features. Every model call goes through one
gateway (`brandshake_ai.gateway.LLMGateway`) that handles rate limiting, caching,
retries and schema validation, so features never talk to a provider directly.

| Provider | Use |
|---|---|
| `fake` (default) | Rule-based, offline and deterministic. Used for development and tests |
| `gemini` | Google Gemini via the `google-genai` SDK |

```bash
uv sync
uv run brandshake-ai          # http://127.0.0.1:8090
uv run pytest
uv run ruff check . && uv run ruff format --check .
```

Gemini configuration:

```bash
BRANDSHAKE_AI_PROVIDER=gemini
GEMINI_API_KEY=...
BRANDSHAKE_AI_GEMINI_MODEL_FAST=...    # model names from ai.google.dev/models
BRANDSHAKE_AI_GEMINI_MODEL_SMART=...
BRANDSHAKE_AI_REQUESTS_PER_MINUTE=10   # keep at or below your quota
```

## Endpoints

- `GET /healthz`
- `POST /v1/terms/extract`: negotiation messages in, structured pact terms, plain-language summary and risk flags out
- `GET /v1/metrics`: per-feature request, cache and error counters
