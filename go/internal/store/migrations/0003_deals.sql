-- Deals are owned by the user who tracks them. The other side may be on
-- Brandshake (counterparty_user_id) or not (name/email only).
-- Brandshake never moves money: amount_minor is the agreed fee, for tracking.
CREATE TABLE deals (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id             uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    counterparty_user_id uuid        REFERENCES users (id) ON DELETE SET NULL,
    counterparty_name    text        NOT NULL CHECK (length(counterparty_name) BETWEEN 1 AND 120),
    counterparty_email   text        NOT NULL DEFAULT '' CHECK (length(counterparty_email) <= 254),
    title                text        NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    status               text        NOT NULL DEFAULT 'lead'
        CHECK (status IN ('lead', 'negotiating', 'confirmed', 'in_progress', 'posted', 'paid', 'lost')),
    amount_minor         bigint      CHECK (amount_minor >= 0),
    currency             text        NOT NULL DEFAULT 'INR' CHECK (currency = 'INR'),
    revisions            smallint    CHECK (revisions >= 0),
    usage_rights_days    integer     CHECK (usage_rights_days >= 0),
    usage_perpetual      boolean     NOT NULL DEFAULT false,
    exclusivity_days     integer     CHECK (exclusivity_days >= 0),
    notes                text        NOT NULL DEFAULT '' CHECK (length(notes) <= 5000),
    source               text        NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'ai_draft')),
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    status_changed_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT deals_not_self CHECK (counterparty_user_id IS NULL OR counterparty_user_id <> owner_id)
);

CREATE INDEX deals_owner_updated_idx ON deals (owner_id, updated_at DESC);

CREATE TABLE deal_deliverables (
    deal_id  uuid     NOT NULL REFERENCES deals (id) ON DELETE CASCADE,
    position smallint NOT NULL,
    title    text     NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    platform text     NOT NULL DEFAULT 'other'
        CHECK (platform IN ('instagram', 'youtube', 'x', 'linkedin', 'facebook', 'other')),
    due_date date,
    done     boolean  NOT NULL DEFAULT false,
    done_at  timestamptz,
    PRIMARY KEY (deal_id, position)
);

-- Supports "due soon" lookups across a user's open deliverables.
CREATE INDEX deal_deliverables_open_due_idx ON deal_deliverables (due_date) WHERE NOT done;

-- Append-only history shown on the deal page.
CREATE TABLE deal_events (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    deal_id     uuid        NOT NULL REFERENCES deals (id) ON DELETE CASCADE,
    actor_id    uuid        REFERENCES users (id) ON DELETE SET NULL,
    kind        text        NOT NULL CHECK (kind IN ('created', 'updated', 'status_changed', 'deliverable_done', 'deliverable_reopened')),
    from_status text,
    to_status   text,
    detail      text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX deal_events_deal_idx ON deal_events (deal_id, id);
