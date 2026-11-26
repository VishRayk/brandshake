"""Terms Extractor: negotiation chat -> structured terms + risk flags.

The model only extracts. Risk flags are deterministic rules in code, so
they are consistent, testable and cannot be talked out of by chat content.
"""

from typing import Literal

from pydantic import BaseModel

from brandshake_ai.gateway import LLMGateway, LLMRequest
from brandshake_ai.prompts import extract_terms as prompt
from brandshake_ai.prompts.extract_terms import ChatMessage, PactTerms, TermsExtraction


class RiskFlag(BaseModel):
    severity: Literal["low", "medium", "high"]
    term: str
    message: str


class TermsResult(BaseModel):
    terms: PactTerms
    summary: str
    risk_flags: list[RiskFlag]
    prompt_version: str
    provider: str
    model: str
    cached: bool


def assess_risks(terms: PactTerms) -> list[RiskFlag]:
    flags: list[RiskFlag] = []
    ur = terms.usage_rights

    if ur.granted and ur.perpetual:
        flags.append(
            RiskFlag(
                severity="high",
                term="usage_rights",
                message="Usage rights are perpetual. Most deals limit reuse to 30–180 days; "
                "consider a time limit or a higher fee.",
            )
        )
    elif ur.granted and ur.duration_days is None:
        flags.append(
            RiskFlag(
                severity="medium",
                term="usage_rights",
                message="Usage rights are granted with no end date. Agree on how long the brand may reuse the content.",
            )
        )
    elif ur.granted and ur.duration_days and ur.duration_days > 365:
        flags.append(
            RiskFlag(
                severity="medium",
                term="usage_rights",
                message=f"Usage rights last {ur.duration_days} days, which is longer than a year.",
            )
        )

    if terms.price_amount is None:
        flags.append(RiskFlag(severity="high", term="price", message="No price has been agreed yet."))
    if not terms.deliverables:
        flags.append(RiskFlag(severity="high", term="deliverables", message="No deliverables have been agreed yet."))
    if terms.revisions is None:
        flags.append(
            RiskFlag(
                severity="low",
                term="revisions",
                message="The number of revision rounds is not set. Agree on one to avoid open-ended edits.",
            )
        )
    if terms.deliverables and all(d.due_date is None for d in terms.deliverables):
        flags.append(RiskFlag(severity="medium", term="due_date", message="No deadline has been agreed."))
    if terms.exclusivity_days and terms.exclusivity_days > 90:
        flags.append(
            RiskFlag(
                severity="medium",
                term="exclusivity",
                message=f"Exclusivity lasts {terms.exclusivity_days} days. Long exclusivity usually costs extra.",
            )
        )
    for q in terms.open_questions:
        flags.append(RiskFlag(severity="low", term="open_question", message=f"Still open: {q}"))
    return flags


async def extract_terms(gateway: LLMGateway, messages: list[ChatMessage]) -> TermsResult:
    req = LLMRequest(
        feature="extract_terms",
        prompt_version=prompt.VERSION,
        system=prompt.SYSTEM,
        prompt=prompt.render(messages),
        schema=TermsExtraction,
        tier="smart",
    )
    res = await gateway.run(req, TermsExtraction)
    return TermsResult(
        terms=res.output.terms,
        summary=res.output.summary,
        risk_flags=assess_risks(res.output.terms),
        prompt_version=res.prompt_version,
        provider=res.provider,
        model=res.model,
        cached=res.cached,
    )
