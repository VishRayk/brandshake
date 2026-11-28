"""Rule-based responders that make FakeProvider useful offline.

They are deliberately simple: good enough for demos, UI work and tests,
not a replacement for the model.
"""

import datetime as dt
import html
import re

from brandshake_ai.prompts.draft_deal import DealDraft, DraftDeliverable
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


_PASTED = re.compile(r"<pasted>\n(.*)\n</pasted>", re.S)
_EMAIL = re.compile(r"[\w.+-]+@([\w-]+)\.[\w.]+")
_FROM = re.compile(r"^from:\s*(?P<name>[^<\n]+?)\s*<", re.I | re.M)
_QTY = re.compile(
    r"\b(\d+|one|two|three|four|five)\s+(?:instagram\s+|youtube\s+)?(reel|story|stories|post|short|video)s?\b", re.I
)
_ISO_DATE = re.compile(r"\b(20\d\d-\d\d-\d\d)\b")
_DAY_MON = re.compile(r"\b(\d{1,2})(?:st|nd|rd|th)?\s+(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\b", re.I)
_WORDS = {"one": 1, "two": 2, "three": 3, "four": 4, "five": 5}
_MONTHS = ["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"]
_DRAFT_PLATFORM = {
    "reel": "instagram",
    "story": "instagram",
    "post": "instagram",
    "short": "youtube",
    "video": "youtube",
}


def _due_date(text: str) -> str | None:
    if m := _ISO_DATE.search(text):
        return m.group(1)
    if m := _DAY_MON.search(text):
        today = dt.date.today()
        month = _MONTHS.index(m.group(2).lower()[:3]) + 1
        try:
            d = dt.date(today.year, month, int(m.group(1)))
        except ValueError:
            return None
        if d < today:
            d = d.replace(year=today.year + 1)
        return d.isoformat()
    return None


def draft_deal(req: LLMRequest) -> DealDraft:
    m = _PASTED.search(req.prompt)
    text = html.unescape(m.group(1)) if m else ""
    lower = text.lower()

    email = _EMAIL.search(text)
    name = None
    if f := _FROM.search(text):
        name = f.group("name").strip().strip('"')
    elif email:
        name = email.group(1).replace("-", " ").title()

    deliverables = []
    for qty, fmt in _QTY.findall(text):
        fmt = "story" if fmt.lower().startswith("stor") else fmt.lower()
        n = int(qty) if qty.isdigit() else _WORDS[qty.lower()]
        deliverables.append(
            DraftDeliverable(platform=_DRAFT_PLATFORM.get(fmt, "other"), format=fmt, quantity=n, due_date=None)
        )
    if not deliverables:
        for fmt, platform in _DRAFT_PLATFORM.items():
            if re.search(rf"\b{fmt}s?\b", lower):
                deliverables.append(DraftDeliverable(platform=platform, format=fmt, quantity=1, due_date=None))
                break
    due = _due_date(text)
    if due and deliverables:
        deliverables[0] = deliverables[0].model_copy(update={"due_date": due})

    price = _price(text)
    revisions = int(r[-1]) if (r := _REVISIONS.findall(text)) else None
    perpetual = "perpetual" in lower or "forever" in lower
    duration = None
    if u := _USAGE.search(text):
        duration = int(u.group(1)) * _UNIT_DAYS[u.group(2).lower()]
    granted = perpetual or duration is not None or " ads" in f" {lower}"

    what = f"{deliverables[0].quantity} {deliverables[0].format}" if deliverables else "collaboration"
    title = f"{name or 'New'} {what}".strip()[:60]
    summary = f"{name or 'The other side'} proposes {what}"
    if price:
        summary += f" for {(price[1] or 'INR')} {price[0]:,.0f}"
    summary += "."

    open_questions = []
    if price is None:
        open_questions.append("Price has not been mentioned.")
    if not due:
        open_questions.append("No deadline has been mentioned.")

    return DealDraft(
        counterparty_name=name,
        counterparty_email=email.group(0) if email else None,
        title=title,
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
        open_questions=open_questions,
        summary=summary,
    )


def default_fake_provider() -> FakeProvider:
    return FakeProvider({TermsExtraction: extract_terms, DealDraft: draft_deal})
