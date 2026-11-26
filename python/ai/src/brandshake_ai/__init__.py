"""Brandshake AI service."""


def main() -> None:
    import os

    import uvicorn

    uvicorn.run(
        "brandshake_ai.api:app",
        host=os.environ.get("BRANDSHAKE_AI_HOST", "127.0.0.1"),
        port=int(os.environ.get("BRANDSHAKE_AI_PORT", "8090")),
    )
