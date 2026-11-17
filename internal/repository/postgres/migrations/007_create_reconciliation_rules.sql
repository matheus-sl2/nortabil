-- ==============================================================================
-- NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
-- Migração 007: Regras de Aprendizado e Classificação na Conciliação Bancária
-- ==============================================================================

ALTER TABLE bank_statement_item
    ADD COLUMN IF NOT EXISTS checknum VARCHAR(100),
    ADD COLUMN IF NOT EXISTS counterpart_account_id UUID REFERENCES account(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS bank_reconciliation_rule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES company(id) ON DELETE CASCADE,
    pattern VARCHAR(255) NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('INFLOW', 'OUTFLOW')),
    counterpart_account_id UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    hit_count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reconciliation_rule UNIQUE (company_id, pattern, direction)
);

CREATE INDEX IF NOT EXISTS idx_rec_rule_company_dir ON bank_reconciliation_rule(company_id, direction);
