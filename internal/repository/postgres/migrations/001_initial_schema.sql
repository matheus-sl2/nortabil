-- ==============================================================================
-- NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
-- Migração 001: Esquema Inicial de Tabelas, Índices e Integridade Transacional
-- ==============================================================================

-- Extensão para geração de identificadores únicos universais (UUID v4)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Sequência global para numeração cronológica do Livro Diário Geral
CREATE SEQUENCE IF NOT EXISTS journal_entry_number_seq START WITH 1 INCREMENT BY 1;

-- ------------------------------------------------------------------------------
-- 1. TABELA DE PLANO DE CONTAS (account)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS account (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    nature VARCHAR(10) NOT NULL CHECK (nature IN ('DEBIT', 'CREDIT')),
    is_synthetic BOOLEAN NOT NULL DEFAULT FALSE,
    parent_id UUID REFERENCES account(id) ON DELETE RESTRICT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_code ON account(code);
CREATE INDEX IF NOT EXISTS idx_account_parent_id ON account(parent_id);
CREATE INDEX IF NOT EXISTS idx_account_type ON account(type);
CREATE INDEX IF NOT EXISTS idx_account_is_synthetic ON account(is_synthetic);

-- ------------------------------------------------------------------------------
-- 2. TABELA DE CABEÇALHO DO LANÇAMENTO CONTÁBIL (journal_entry)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS journal_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_number BIGINT NOT NULL UNIQUE DEFAULT nextval('journal_entry_number_seq'),
    date DATE NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'POSTED' CHECK (status IN ('DRAFT', 'POSTED', 'REVERSED')),
    total_debit NUMERIC(18, 4) NOT NULL DEFAULT 0,
    total_credit NUMERIC(18, 4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_journal_entry_date ON journal_entry(date);
CREATE INDEX IF NOT EXISTS idx_journal_entry_entry_number ON journal_entry(entry_number);
CREATE INDEX IF NOT EXISTS idx_journal_entry_status ON journal_entry(status);

-- ------------------------------------------------------------------------------
-- 3. TABELA DE PARTIDAS CONTÁBEIS (journal_line)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS journal_line (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_entry_id UUID NOT NULL REFERENCES journal_entry(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES account(id) ON DELETE RESTRICT,
    type VARCHAR(10) NOT NULL CHECK (type IN ('DEBIT', 'CREDIT')),
    amount NUMERIC(18, 4) NOT NULL CHECK (amount > 0),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_journal_line_entry_id ON journal_line(journal_entry_id);
CREATE INDEX IF NOT EXISTS idx_journal_line_account_id ON journal_line(account_id);
CREATE INDEX IF NOT EXISTS idx_journal_line_type ON journal_line(type);

-- ------------------------------------------------------------------------------
-- 4. INTEGRIDADE TRANSACIONAL: LEI DE LUCA PACIOLI A NÍVEL DE BANCO DE DADOS
-- Função e Trigger diferidos (DEFERRABLE INITIALLY DEFERRED) que conferem o
-- balanceamento das partidas no momento do COMMIT da transação.
-- ------------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION check_journal_entry_balance()
RETURNS TRIGGER AS $$
DECLARE
    sum_debit NUMERIC(18, 4);
    sum_credit NUMERIC(18, 4);
    has_synthetic BOOLEAN;
BEGIN
    -- 1. Verifica se alguma conta utilizada no lançamento é sintética
    SELECT EXISTS (
        SELECT 1 
        FROM journal_line jl
        JOIN account a ON a.id = jl.account_id
        WHERE jl.journal_entry_id = NEW.journal_entry_id AND a.is_synthetic = TRUE
    ) INTO has_synthetic;

    IF has_synthetic THEN
        RAISE EXCEPTION 'Lançamento contábil rejeitado: proibido lançar em contas sintéticas (Resolução CFC ITG 2000)';
    END IF;

    -- 2. Soma débitos e créditos associados ao lançamento
    SELECT 
        COALESCE(SUM(CASE WHEN type = 'DEBIT' THEN amount ELSE 0 END), 0),
        COALESCE(SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE 0 END), 0)
    INTO sum_debit, sum_credit
    FROM journal_line
    WHERE journal_entry_id = NEW.journal_entry_id;

    -- Se não há linhas ainda, permite aguardar as próximas inserções da mesma transação
    IF sum_debit = 0 AND sum_credit = 0 THEN
        RETURN NEW;
    END IF;

    -- Validação do Teorema Fundamental de Luca Pacioli
    IF sum_debit <> sum_credit THEN
        RAISE EXCEPTION 'Violação da Lei de Partidas Dobradas: Débito (%) diferente de Crédito (%) no lançamento %', 
            sum_debit, sum_credit, NEW.journal_entry_id;
    END IF;

    -- Atualiza os totalizadores do cabeçalho
    UPDATE journal_entry 
    SET total_debit = sum_debit, total_credit = sum_credit, updated_at = NOW()
    WHERE id = NEW.journal_entry_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_enforce_double_entry_balance ON journal_line;

CREATE CONSTRAINT TRIGGER trg_enforce_double_entry_balance
AFTER INSERT OR UPDATE OR DELETE ON journal_line
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION check_journal_entry_balance();
