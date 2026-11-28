-- Creator invoicing. Brandshake generates and tracks invoices; the brand pays
-- the creator directly (UPI or bank transfer). No money passes through us.

CREATE TABLE billing_profiles (
    user_id             uuid        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    legal_name          text        NOT NULL CHECK (length(legal_name) BETWEEN 1 AND 120),
    address             text        NOT NULL DEFAULT '' CHECK (length(address) <= 300),
    state               text        NOT NULL,
    pan                 text        NOT NULL DEFAULT '',
    gst_registered      boolean     NOT NULL DEFAULT false,
    gstin               text        NOT NULL DEFAULT '',
    gst_rate            smallint    NOT NULL DEFAULT 18 CHECK (gst_rate IN (0, 5, 18, 40)),
    upi_id              text        NOT NULL DEFAULT '',
    bank_account_name   text        NOT NULL DEFAULT '' CHECK (length(bank_account_name) <= 120),
    bank_account_number text        NOT NULL DEFAULT '',
    bank_ifsc           text        NOT NULL DEFAULT '',
    bank_name           text        NOT NULL DEFAULT '' CHECK (length(bank_name) <= 80),
    invoice_prefix      text        NOT NULL DEFAULT 'INV' CHECK (invoice_prefix ~ '^[A-Z0-9]{1,5}$'),
    payment_terms_days  smallint    NOT NULL DEFAULT 15 CHECK (payment_terms_days BETWEEN 0 AND 120),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT billing_gstin_when_registered CHECK (NOT gst_registered OR gstin <> ''),
    CONSTRAINT billing_has_payment_method CHECK (upi_id <> '' OR bank_account_number <> '')
);

-- Gapless per-user, per-financial-year sequence (incremented inside the
-- invoice transaction, so a rolled-back invoice doesn't burn a number).
CREATE TABLE invoice_counters (
    user_id     uuid    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    fy          text    NOT NULL CHECK (fy ~ '^[0-9]{4}-[0-9]{2}$'),
    last_number integer NOT NULL CHECK (last_number >= 0),
    PRIMARY KEY (user_id, fy)
);

CREATE TABLE invoices (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id              uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    deal_id               uuid        REFERENCES deals (id) ON DELETE SET NULL,
    number                text        NOT NULL CHECK (length(number) <= 16),
    fy                    text        NOT NULL,
    status                text        NOT NULL DEFAULT 'issued' CHECK (status IN ('issued', 'paid', 'void')),
    issue_date            date        NOT NULL,
    due_date              date        NOT NULL CHECK (due_date >= issue_date),
    -- Seller details are copied at issue time so the invoice never changes later.
    seller                jsonb       NOT NULL,
    buyer_name            text        NOT NULL CHECK (length(buyer_name) BETWEEN 1 AND 120),
    buyer_email           text        NOT NULL DEFAULT '' CHECK (length(buyer_email) <= 254),
    buyer_address         text        NOT NULL DEFAULT '' CHECK (length(buyer_address) <= 300),
    buyer_state           text        NOT NULL DEFAULT '',
    buyer_gstin           text        NOT NULL DEFAULT '',
    gst_mode              text        NOT NULL CHECK (gst_mode IN ('none', 'intra', 'inter')),
    gst_rate              smallint    NOT NULL,
    subtotal_minor        bigint      NOT NULL CHECK (subtotal_minor >= 0),
    cgst_minor            bigint      NOT NULL DEFAULT 0,
    sgst_minor            bigint      NOT NULL DEFAULT 0,
    igst_minor            bigint      NOT NULL DEFAULT 0,
    total_minor           bigint      NOT NULL CHECK (total_minor >= subtotal_minor),
    notes                 text        NOT NULL DEFAULT '' CHECK (length(notes) <= 1000),
    -- Unguessable token for the brand's no-login link.
    public_token          text        NOT NULL,
    sent_at               timestamptz,
    payer_claimed_paid_at timestamptz,
    payer_reference       text        NOT NULL DEFAULT '' CHECK (length(payer_reference) <= 100),
    paid_at               timestamptz,
    paid_reference        text        NOT NULL DEFAULT '' CHECK (length(paid_reference) <= 100),
    voided_at             timestamptz,
    reminders_sent        smallint    NOT NULL DEFAULT 0,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT invoices_owner_number_key UNIQUE (owner_id, number),
    CONSTRAINT invoices_public_token_key UNIQUE (public_token),
    CONSTRAINT invoices_paid_has_date CHECK (status <> 'paid' OR paid_at IS NOT NULL)
);

CREATE INDEX invoices_owner_idx ON invoices (owner_id, issue_date DESC, number DESC);
CREATE INDEX invoices_open_due_idx ON invoices (due_date) WHERE status = 'issued';

CREATE TABLE invoice_items (
    invoice_id       uuid     NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    position         smallint NOT NULL,
    description      text     NOT NULL CHECK (length(description) BETWEEN 1 AND 200),
    sac_code         text     NOT NULL DEFAULT '998361',
    quantity         integer  NOT NULL CHECK (quantity BETWEEN 1 AND 1000),
    unit_price_minor bigint   NOT NULL CHECK (unit_price_minor >= 0),
    amount_minor     bigint   NOT NULL CHECK (amount_minor = quantity * unit_price_minor),
    PRIMARY KEY (invoice_id, position)
);

CREATE TABLE invoice_events (
    id         bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_id uuid        NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    kind       text        NOT NULL CHECK (kind IN ('issued', 'sent', 'viewed', 'payer_claimed_paid', 'paid', 'voided', 'reminder_sent')),
    detail     text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX invoice_events_invoice_idx ON invoice_events (invoice_id, id);
