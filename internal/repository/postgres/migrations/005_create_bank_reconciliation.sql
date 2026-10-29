-- ==============================================================================
-- NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
-- Migração 005: Conciliação Bancária e Ingestão de Extratos OFX
-- ==============================================================================

CREATE TABLE IF NOT EXISTS bank_statement_item (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES company(id) ON DELETE CASCADE,
    bank_account_id UUID NOT NULL REFERENCES account(id) ON DELETE RESTRICT,
    fitid VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    amount NUMERIC(18, 4) NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('DEBIT', 'CREDIT')),
    memo TEXT,
    journal_entry_id UUID REFERENCES journal_entry(id) ON DELETE SET NULL,
    reconciled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_bank_fitid UNIQUE (company_id, bank_account_id, fitid)
);

CREATE INDEX IF NOT EXISTS idx_bank_item_company ON bank_statement_item(company_id);
CREATE INDEX IF NOT EXISTS idx_bank_item_account ON bank_statement_item(bank_account_id);
CREATE INDEX IF NOT EXISTS idx_bank_item_reconciled ON bank_statement_item(reconciled);

