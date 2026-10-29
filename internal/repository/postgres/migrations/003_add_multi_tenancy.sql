-- ==============================================================================
-- NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
-- Migração 003: Suporte a Multi-Tenancy (Multi-Empresas e Escritórios)
-- ==============================================================================

-- 1. TABELA DE EMPRESAS / CLIENTES DO ESCRITÓRIO (company)
CREATE TABLE IF NOT EXISTS company (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cnpj VARCHAR(18) NOT NULL UNIQUE,
    legal_name VARCHAR(255) NOT NULL,
    trade_name VARCHAR(255),
    tax_regime VARCHAR(50) NOT NULL CHECK (tax_regime IN ('SIMPLES_NACIONAL', 'LUCRO_PRESUMIDO', 'LUCRO_REAL', 'MEI')),
    city VARCHAR(100) NOT NULL DEFAULT 'Januária',
    state VARCHAR(2) NOT NULL DEFAULT 'MG',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_company_cnpj ON company(cnpj);
CREATE INDEX IF NOT EXISTS idx_company_active ON company(active);

-- 2. INSERE EMPRESA PADRÃO (Para retrocompatibilidade e ambiente inicial)
INSERT INTO company (id, cnpj, legal_name, trade_name, tax_regime, city, state, active)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '12.345.678/0001-90',
    'Comércio e Serviços Norte de Minas Ltda',
    'Nortabil Matriz Januária',
    'SIMPLES_NACIONAL',
    'Januária',
    'MG',
    TRUE
) ON CONFLICT (cnpj) DO NOTHING;

-- 3. ADICIONA company_id NA TABELA account
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='account' AND column_name='company_id'
    ) THEN
        ALTER TABLE account ADD COLUMN company_id UUID REFERENCES company(id) ON DELETE CASCADE;
        UPDATE account SET company_id = '00000000-0000-0000-0000-000000000001' WHERE company_id IS NULL;
        ALTER TABLE account ALTER COLUMN company_id SET NOT NULL;

        -- Remove chave única antiga apenas por código e cria chave única composta (company_id, code)
        ALTER TABLE account DROP CONSTRAINT IF EXISTS account_code_key;
        ALTER TABLE account ADD CONSTRAINT uq_account_company_code UNIQUE (company_id, code);
        CREATE INDEX IF NOT EXISTS idx_account_company_id ON account(company_id);
    END IF;
END $$;

-- 4. ADICIONA company_id NA TABELA journal_entry
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='journal_entry' AND column_name='company_id'
    ) THEN
        ALTER TABLE journal_entry ADD COLUMN company_id UUID REFERENCES company(id) ON DELETE CASCADE;
        UPDATE journal_entry SET company_id = '00000000-0000-0000-0000-000000000001' WHERE company_id IS NULL;
        ALTER TABLE journal_entry ALTER COLUMN company_id SET NOT NULL;

        CREATE INDEX IF NOT EXISTS idx_journal_entry_company_id ON journal_entry(company_id);
        CREATE INDEX IF NOT EXISTS idx_journal_entry_company_date ON journal_entry(company_id, date);
    END IF;
END $$;

-- 5. ATUALIZAÇÃO DO TRIGGER DE INTEGRIDADE CONTÁBIL
-- Valida que todas as partidas do lançamento pertencem a contas da mesma empresa
CREATE OR REPLACE FUNCTION check_journal_entry_balance()
RETURNS TRIGGER AS $$
DECLARE
    sum_debit NUMERIC(18, 4);
    sum_credit NUMERIC(18, 4);
    has_synthetic BOOLEAN;
    entry_comp_id UUID;
    mismatched_company BOOLEAN;
BEGIN
    -- Busca a empresa do cabeçalho do lançamento
    SELECT company_id INTO entry_comp_id 
    FROM journal_entry 
    WHERE id = NEW.journal_entry_id;

    -- 1. Verifica se alguma conta utilizada pertence a outra empresa (quebra de isolamento)
    SELECT EXISTS (
        SELECT 1 
        FROM journal_line jl
        JOIN account a ON a.id = jl.account_id
        WHERE jl.journal_entry_id = NEW.journal_entry_id 
          AND a.company_id <> entry_comp_id
    ) INTO mismatched_company;

    IF mismatched_company THEN
        RAISE EXCEPTION 'Quebra de isolamento multi-tenant: contas pertencem a empresas distintas';
    END IF;

    -- 2. Verifica se alguma conta utilizada no lançamento é sintética
    SELECT EXISTS (
        SELECT 1 
        FROM journal_line jl
        JOIN account a ON a.id = jl.account_id
        WHERE jl.journal_entry_id = NEW.journal_entry_id AND a.is_synthetic = TRUE
    ) INTO has_synthetic;

    IF has_synthetic THEN
        RAISE EXCEPTION 'Lançamento contábil rejeitado: proibido lançar em contas sintéticas (Resolução CFC ITG 2000)';
    END IF;

    -- 3. Soma débitos e créditos associados ao lançamento
    SELECT 
        COALESCE(SUM(CASE WHEN type = 'DEBIT' THEN amount ELSE 0 END), 0),
        COALESCE(SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE 0 END), 0)
    INTO sum_debit, sum_credit
    FROM journal_line
    WHERE journal_entry_id = NEW.journal_entry_id;

    IF sum_debit = 0 AND sum_credit = 0 THEN
        RETURN NEW;
    END IF;

    -- 4. Validação do Teorema Fundamental de Luca Pacioli
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
