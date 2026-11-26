"""Rule-based responders that make FakeProvider useful offline.

They are deliberately simple: good enough for demos, UI work and tests,
not a replacement for the model.
"""

import html
import re

from brandshake_ai.prompts.extract_terms import Deliverable, PactTerms, TermsExtraction, UsageRights

from .base import LLMRequest
from .fake import FakeProvider

_MSG = re.compile(r'<message n="\d+" author="(brand|creator)">(.*?)</message>', re.S)
_PRICE = re.compile(r"(?P<cur>₹|\$|rs\.?|inr|usd)?\s*(?P<num>\d[\d,]*(?:\.\d+)?)\s*(?P<k>k)?\b", re.I)
_REVISIONS = re.compile(r"(\d+)\s+revisions?", re.I)
_USAGE = re.compile(r"(?:ads?|usage|reuse)[^.]*?(\d+)\s*(day|week|month|year)s?", re.I)
_FORMATS = {
    "reel": "instagram",
    "story": "instagram",
    "post": "instagram",
    "short": "youtube",
    "video": "youtube",
    "tiktok": "tiktok",
}
_UNIT_DAYS = {"day": 1, "week": 7, "month": 30, "year": 365}


def _messages(prompt: str) -> list[tuple[str, str]]:
    return [(a, html.unescape(t)) for a, t in _MSG.findall(prompt)]


def _price(text: str) -> tuple[float, str | None] | None:
    best = None
    for m in _PRICE.finditer(text):
        if not (m.group("cur") or m.group("k")):
            continue
        value = float(m.group("num").replace(",", "")) * (1000 if m.group("k") else 1)
        cur = (m.group("cur") or "").lower()
        currency = "USD" if cur in ("$", "usd") else "INR" if cur else None
        best = (value, currency)
    return best


def extract_terms(req: LLMRequest) -> TermsExtraction:
    msgs = _messages(req.prompt)
    text = " ".join(t for _, t in msgs)
    lower = text.lower()

    price = None
    for _, t in msgs:  # the latest mentioned price wins
        price = _price(t) or price

    deliverables = [
        Deliverable(platform=platform, format=fmt, quantity=1, requirements=[], due_date=None)
        for fmt, platform in _FORMATS.items()
        if re.search(rf"\b{fmt}s?\b", lower)
    ][:1]

    revisions = None
    if m := _REVISIONS.findall(text):
        revisions = int(m[-1])

    perpetual = "perpetual" in lower or "forever" in lower
    duration = None
    if m := _USAGE.search(text):
        duration = int(m.group(1)) * _UNIT_DAYS[m.group(2).lower()]
    granted = perpetual or duration is not None or " ads" in f" {lower}"

    terms = PactTerms(
        deliverables=deliverables,
        price_amount=price[0] if price else None,
        currency=(price[1] or "INR") if price else None,
        revisions=revisions,
        usage_rights=UsageRights(
            granted=granted,
            channels=["paid ads"] if granted else [],
            duration_days=None if perpetual else duration,
            perpetual=perpetual,
        ),
        exclusivity_days=None,
        open_questions=[],
    )

    parts = []
    if deliverables:
        parts.append(f"The creator delivers 1 {deliverables[0].platform} {deliverables[0].format}")
    if terms.price_amount is not None:
        parts.append(f"for {terms.currency} {terms.price_amount:,.0f}")
    summary = (" ".join(parts) + ".") if parts else "No terms have been agreed yet."
    if revisions is not None:
        summary += f" Includes {revisions} revision round{'s' if revisions != 1 else ''}."
    if granted:
        summary += " The brand may reuse the content in ads " + (
            "indefinitely." if perpetual else f"for {duration} days." if duration else "(duration not set)."
        )
    return TermsExtraction(terms=terms, summary=summary)


def default_fake_provider() -> FakeProvider:
    return FakeProvider({TermsExtraction: extract_terms})
