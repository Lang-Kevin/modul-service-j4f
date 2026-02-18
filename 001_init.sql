-- migrations/001_init.sql
-- Run this once against your Postgres database before starting the service.

CREATE TABLE IF NOT EXISTS time_slices (
    id              BIGSERIAL PRIMARY KEY,
    contract_id     TEXT        NOT NULL,
    top_article_id  BIGINT      NOT NULL,
    validity_tag    TEXT        NOT NULL,
    invoice_date    DATE        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookups by contract
CREATE INDEX IF NOT EXISTS idx_time_slices_contract_id
    ON time_slices (contract_id);

-- Index for the invoice-date dedup query
CREATE INDEX IF NOT EXISTS idx_time_slices_contract_created
    ON time_slices (contract_id, created_at DESC);
