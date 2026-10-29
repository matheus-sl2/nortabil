-- ==============================================================================
-- NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
-- Migração 004: Controle de Períodos Contábeis, Travas de Fechamento e ARE
-- ==============================================================================

CREATE TABLE IF NOT EXISTS accounting_period (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES company(id) ON DELETE CASCADE,
    year INT NOT NULL CHECK (year >= 1970 AND year <= 2100),
    month INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED', 'LOCKED')),
    closed_at TIMESTAMPTZ,
    closed_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_company_period UNIQUE (company_id, year, month)
);

CREATE INDEX IF NOT EXISTS idx_accounting_period_company ON accounting_period(company_id);
CREATE INDEX IF NOT EXISTS idx_accounting_period_status ON accounting_period(status);
CREATE INDEX IF NOT EXISTS idx_accounting_period_range ON accounting_period(start_date, end_date);
