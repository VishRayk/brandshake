"""Deal drafts: pasted email or chat -> a deal the user can review and save."""

from typing import Literal

from pydantic import BaseModel, Field

from brandshake_ai.features.terms import RiskFlag, assess_risks
from brandshake_ai.gateway import LLMGateway, LLMRequest
from brandshake_ai.prompts import draft_deal as prompt
from brandshake_ai.prompts.draft_deal import DealDraft
from brandshake_ai.prompts.extract_terms import Deliverable, PactTerms


class DraftRequest(BaseModel):
    text: str = Field(min_length=1, max_length=20_000)
    owner_role: Literal["creator", "brand"]


class DraftResult(BaseModel):
    draft: DealDraft
    risk_flags: list[RiskFlag]
    prompt_version: str
    provider: str
    model: str
    cached: bool


def _as_terms(d: DealDraft) -> PactTerms:
    """Reuse the terms risk rules on a draft."""
    return PactTerms(
        deliverables=[
            Deliverable(
                platform=x.platform if x.platform in ("youtube", "instagram") else "other",
                format=x.format,
                quantity=x.quantity,
                requirements=[],
                due_date=x.due_date,
            )
            for x in d.deliverables
        ],
        price_amount=d.price_amount,
        currency=d.currency,
        revisions=d.revisions,
        usage_rights=d.usage_rights,
        exclusivity_days=d.exclusivity_days,
        open_questions=d.open_questions,
    )


async def draft_deal(gateway: LLMGateway, body: DraftRequest) -> DraftResult:
    req = LLMRequest(
        feature="draft_deal",
        prompt_version=prompt.VERSION,
        system=prompt.system_for(body.owner_role),
        prompt=prompt.render(body.text),
        schema=DealDraft,
        tier="smart",
    )
    res = await gateway.run(req, DealDraft)
    return DraftResult(
        draft=res.output,
        risk_flags=assess_risks(_as_terms(res.output)),
        prompt_version=res.prompt_version,
        provider=res.provider,
        model=res.model,
        cached=res.cached,
    )
