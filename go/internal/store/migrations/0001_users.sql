CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('creator', 'brand')),
    display_name  TEXT NOT NULL,
    created_at    TEXT NOT NULL
);
