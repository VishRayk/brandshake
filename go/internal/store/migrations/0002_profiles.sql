-- One public profile per user. Creator-only and brand-only columns are
-- nullable or default-empty; the API enforces which role may set them.
CREATE TABLE profiles (
    user_id         uuid        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    handle          text        NOT NULL,
    headline        text        NOT NULL DEFAULT '' CHECK (length(headline) <= 120),
    bio             text        NOT NULL DEFAULT '' CHECK (length(bio) <= 1000),
    city            text        NOT NULL DEFAULT '' CHECK (length(city) <= 60),
    state           text        NOT NULL DEFAULT '' CHECK (length(state) <= 60),
    website         text        NOT NULL DEFAULT '' CHECK (length(website) <= 200),
    -- creators
    niches          text[]      NOT NULL DEFAULT '{}',
    languages       text[]      NOT NULL DEFAULT '{}',
    open_to_collabs boolean     NOT NULL DEFAULT true,
    -- brands
    industry        text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT profiles_handle_key UNIQUE (handle),
    CONSTRAINT profiles_handle_format CHECK (handle ~ '^[a-z0-9][a-z0-9_.]{1,28}[a-z0-9]$')
);

CREATE INDEX profiles_niches_idx ON profiles USING gin (niches);

CREATE TABLE social_accounts (
    user_id   uuid    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    platform  text    NOT NULL CHECK (platform IN ('instagram', 'youtube', 'x', 'linkedin', 'facebook')),
    handle    text    NOT NULL CHECK (length(handle) BETWEEN 1 AND 60),
    url       text    NOT NULL CHECK (length(url) <= 200),
    followers integer NOT NULL DEFAULT 0 CHECK (followers >= 0),
    verified  boolean NOT NULL DEFAULT false,
    position  smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, platform)
);

-- Creator rate card. Amounts are in paise (minor units); never floats.
CREATE TABLE rate_cards (
    user_id     uuid     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    position    smallint NOT NULL,
    deliverable text     NOT NULL CHECK (length(deliverable) BETWEEN 1 AND 80),
    price_minor bigint   NOT NULL CHECK (price_minor >= 0),
    note        text     NOT NULL DEFAULT '' CHECK (length(note) <= 120),
    PRIMARY KEY (user_id, position)
);

CREATE TABLE work_samples (
    user_id    uuid     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    position   smallint NOT NULL,
    title      text     NOT NULL CHECK (length(title) BETWEEN 1 AND 100),
    url        text     NOT NULL CHECK (length(url) <= 300),
    platform   text     NOT NULL DEFAULT '',
    brand_name text     NOT NULL DEFAULT '' CHECK (length(brand_name) <= 80),
    PRIMARY KEY (user_id, position)
);
