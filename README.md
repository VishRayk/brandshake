<p align="center">
  <img src="web/public/favicon.svg" width="56" alt="Brandshake logo" />
</p>

<h1 align="center">Brandshake</h1>

<p align="center"><strong>Where creators and brands do business.</strong><br/>
The workspace and network for creator–brand partnerships in India.</p>

<p align="center">
  <a href="https://github.com/VishRayk/brandshake/actions/workflows/ci.yml"><img src="https://github.com/VishRayk/brandshake/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Python-3.12-3776AB?logo=python&logoColor=white" alt="Python" />
  <img src="https://img.shields.io/badge/React-19-149ECA?logo=react&logoColor=white" alt="React" />
</p>

---

## Why Brandshake

Creator–brand partnerships still run on Instagram DMs, WhatsApp threads and spreadsheets. Creators chase late payments and rebuild invoices by hand; brands juggle creator lists and struggle to find the right people. Brandshake gives both sides one place to **run** their partnerships (deals, deliverables, invoices) and one place to **find** each other.

## Architecture

```
┌──────────────────────┐   REST/JSON   ┌───────────────────────┐
│  Web app             │ ────────────▶ │  Core API (Go)        │──▶ PostgreSQL
│  React · Vite · TS   │               │  auth · accounts      │
└──────────────────────┘               └───────────────────────┘

                                        ┌───────────────────────┐
                                        │  AI service (Python)  │──▶ Gemini
                                        │  FastAPI · LLM gateway│
                                        └───────────────────────┘
```

| Service | Path | Stack |
|---|---|---|
| **Core API** | [`go/`](go) | Go standard library HTTP server, PostgreSQL via pgx, embedded migrations, bcrypt, JWT |
| **AI service** | [`python/ai/`](python/ai) | FastAPI, Pydantic, `google-genai`, uv |
| **Web app** | [`web/`](web) | React 19, Vite, TypeScript, Tailwind CSS v4, TanStack Router & Query, React Hook Form, Zod |

## Highlights

**Core API (Go)**
- Public creator and brand profiles: unique handles with reserved words blocked, niches, languages, social accounts (links checked against each platform's domain), rate cards in paise, and work samples
- Account data export and permanent deletion, in line with India's DPDP Act
- Deal tracker: deals with brands or creators on or off Brandshake, deliverables with due dates, fee in paise, revisions, usage rights and exclusivity, a status pipeline, and an append-only history written in the same transaction as each change
- Every deal query is scoped to its owner; other users get 404, never a hint that the deal exists
- "Paste an email" drafting: the Go API calls the AI service, maps the draft onto validated deal fields, and saves nothing until the user confirms
- Dashboard summary: active deals, payments awaited, paid this month, overdue and upcoming deliverables
- Email sign-up and sign-in for creators and brands with bcrypt hashing and HS256 access tokens
- Login runs in the same time whether or not the account exists, so emails can't be discovered
- Strict JSON decoding (size limit, unknown fields rejected) and field-level validation errors in one error envelope
- Request IDs, structured logging, panic recovery, CORS, security headers, timeouts and graceful shutdown
- PostgreSQL with transactional, embedded migrations guarded by an advisory lock, so concurrent deploys never race
- Integration tests run against real PostgreSQL, each in its own throwaway schema

**AI service (Python)**
- **One LLM gateway** for every model call: token-bucket rate limiting, result caching, retries with exponential backoff and jitter, and Pydantic validation of every response
- **Provider-agnostic**: Gemini in production, plus a deterministic offline provider for development and tests
- **Terms extractor**: turns a brand–creator negotiation into structured deal terms (deliverables, price, revisions, usage rights, exclusivity) and a plain-language summary
- **Risk flags come from code, not the model**: perpetual usage rights, missing deadlines and unpriced deals are flagged by deterministic rules
- **Prompt-injection safe**: chat content is escaped and passed to the model only as untrusted data

**Web app**
- A design system built on tokens, with light and dark themes, accessible form fields and tabular money formatting
- Responsive app shell: sidebar on desktop, bottom tabs on mobile
- Profile editor with live handle availability, niche and language pickers, and editable social accounts, rate card (₹, stored in paise) and work samples
- Public profile pages at `/@handle`: a shareable media kit for creators and a brand page for brands
- Home with a profile-setup checklist and live deal numbers; settings with data export and account deletion
- Deals board (a column per stage) and list views with search; new deal from a pasted email or by hand; deal page with a deliverable checklist, terms and activity history
- Every page is code-split; the main bundle is about 145 KB gzipped
- Forms validated on the client with Zod, with server errors mapped back onto each field

## Getting started

Requirements: Go 1.27+, PostgreSQL 16+, Python 3.12+ with [uv](https://docs.astral.sh/uv/), Node.js 26+.

```bash
git clone https://github.com/VishRayk/brandshake.git
cd brandshake
cp .env.example .env
make setup
make dev
```

Create the local databases once (PostgreSQL 16 on Ubuntu or WSL):

```bash
sudo -u postgres psql -c "CREATE ROLE brandshake WITH LOGIN CREATEDB PASSWORD 'brandshake';"
sudo -u postgres createdb -O brandshake brandshake
sudo -u postgres createdb -O brandshake brandshake_test
```

`make db-start` and `make db-stop` start and stop PostgreSQL.

| | URL |
|---|---|
| Web app | http://localhost:5173 |
| Core API | http://localhost:8080 |
| AI service | http://localhost:8090 |

The AI service runs offline by default. To use Gemini, set `BRANDSHAKE_AI_PROVIDER=gemini`, `GEMINI_API_KEY` and the model names in `.env` (see [`python/ai/README.md`](python/ai/README.md)).

## API

```http
POST   /v1/auth/signup        { email, password, role: "creator" | "brand", display_name }
POST   /v1/auth/login         { email, password }
GET    /v1/me                 Authorization: Bearer <token>
DELETE /v1/me                 { password }         delete account and all data
GET    /v1/me/export          download all your data as JSON

GET    /v1/me/profile         profile, socials, rate card, work samples
PUT    /v1/me/profile         { handle, headline, bio, city, state, website, niches, languages, open_to_collabs, industry }
PUT    /v1/me/socials         { socials: [{ platform, handle, url, followers }] }
PUT    /v1/me/rates           { rates: [{ deliverable, price_minor, note }] }          creators
PUT    /v1/me/samples         { samples: [{ title, url, brand_name }] }               creators
GET    /v1/handles/{handle}   is a handle available?
GET    /v1/profiles/{handle}  public profile (no auth)

GET    /v1/deals?status=&q=   your deals, with deliverables
POST   /v1/deals              { title, counterparty_name | counterparty_handle, amount_minor, deliverables, ... }
POST   /v1/deals/draft        { text }   draft a deal from a pasted email or chat (nothing saved)
GET    /v1/deals/summary      dashboard numbers
GET    /v1/deals/{id}         · PUT replaces details · DELETE
PUT    /v1/deals/{id}/status  { status }
PUT    /v1/deals/{id}/deliverables/{position}   { done }
GET    /v1/deals/{id}/events  history

GET    /healthz · /readyz
```

AI service:

```http
POST /v1/terms/extract   { messages: [{ author: "brand" | "creator", text }] }
POST /v1/deals/draft     { text, owner_role }
GET  /v1/metrics
```

Errors always use the same envelope:

```json
{ "error": { "code": "validation_failed", "message": "Some fields are invalid.", "fields": { "email": "Enter a valid email address." } } }
```

## Development

```bash
make test    # Go (with race detector), pytest, Vitest
make lint    # gofmt + go vet, ruff, oxlint + tsc
make build   # API binary and production web bundle
```

CI runs all of these on every push and pull request.
