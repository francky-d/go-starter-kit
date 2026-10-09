CREATE TABLE users (
    id            uuid PRIMARY KEY,
    name          text NOT NULL,
    email         text NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    -- Emails are stored lowercase, so the unique index below is enough to
    -- forbid two accounts differing only by case.
    CONSTRAINT users_email_lowercase CHECK (email = lower(email))
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
