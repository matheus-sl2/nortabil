-- ==============================================================================
-- NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
-- Migração 002: Carga Inicial de Plano de Contas Referencial Brasileiro (SPED/ECD)
-- ==============================================================================

-- 1. ATIVO (Sintética)
INSERT INTO account (id, code, name, type, nature, is_synthetic, active)
VALUES ('00000000-0000-0000-0001-000000000000', '1', 'ATIVO', 'ASSET', 'DEBIT', TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

  -- 1.1 ATIVO CIRCULANTE (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0001-000000000010', '1.1', 'ATIVO CIRCULANTE', 'ASSET', 'DEBIT', TRUE, '00000000-0000-0000-0001-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 1.1.01 Caixa e Equivalentes de Caixa (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0001-000000000100', '1.1.01', 'Caixa e Equivalentes de Caixa', 'ASSET', 'DEBIT', TRUE, '00000000-0000-0000-0001-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 1.1.01.001 Caixa Geral (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000001001', '1.1.01.001', 'Caixa Geral', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000100', TRUE)
      ON CONFLICT (code) DO NOTHING;

      -- 1.1.01.002 Bancos Conta Movimento (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000001002', '1.1.01.002', 'Bancos Conta Movimento', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000100', TRUE)
      ON CONFLICT (code) DO NOTHING;

      -- 1.1.01.003 Aplicações de Liquidez Imediata (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000001003', '1.1.01.003', 'Aplicações de Liquidez Imediata', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000100', TRUE)
      ON CONFLICT (code) DO NOTHING;

    -- 1.1.02 Clientes e Contas a Receber (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0001-000000000200', '1.1.02', 'Clientes e Contas a Receber', 'ASSET', 'DEBIT', TRUE, '00000000-0000-0000-0001-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 1.1.02.001 Clientes Nacionais (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000002001', '1.1.02.001', 'Clientes Nacionais', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000200', TRUE)
      ON CONFLICT (code) DO NOTHING;

    -- 1.1.03 Estoques (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0001-000000000300', '1.1.03', 'Estoques', 'ASSET', 'DEBIT', TRUE, '00000000-0000-0000-0001-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 1.1.03.001 Mercadorias para Revenda (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000003001', '1.1.03.001', 'Mercadorias para Revenda', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000300', TRUE)
      ON CONFLICT (code) DO NOTHING;

  -- 1.2 ATIVO NÃO CIRCULANTE (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0001-000000000020', '1.2', 'ATIVO NÃO CIRCULANTE', 'ASSET', 'DEBIT', TRUE, '00000000-0000-0000-0001-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 1.2.01 Imobilizado (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0001-000000000400', '1.2.01', 'Imobilizado', 'ASSET', 'DEBIT', TRUE, '00000000-0000-0000-0001-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 1.2.01.001 Máquinas e Equipamentos (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000004001', '1.2.01.001', 'Máquinas e Equipamentos', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000400', TRUE)
      ON CONFLICT (code) DO NOTHING;

      -- 1.2.01.002 Veículos (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000004002', '1.2.01.002', 'Veículos', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000400', TRUE)
      ON CONFLICT (code) DO NOTHING;

      -- 1.2.01.003 Computadores e Periféricos (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0001-000000004003', '1.2.01.003', 'Computadores e Periféricos', 'ASSET', 'DEBIT', FALSE, '00000000-0000-0000-0001-000000000400', TRUE)
      ON CONFLICT (code) DO NOTHING;

-- 2. PASSIVO (Sintética)
INSERT INTO account (id, code, name, type, nature, is_synthetic, active)
VALUES ('00000000-0000-0000-0002-000000000000', '2', 'PASSIVO', 'LIABILITY', 'CREDIT', TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

  -- 2.1 PASSIVO CIRCULANTE (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0002-000000000010', '2.1', 'PASSIVO CIRCULANTE', 'LIABILITY', 'CREDIT', TRUE, '00000000-0000-0000-0002-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 2.1.01 Fornecedores (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0002-000000000100', '2.1.01', 'Fornecedores', 'LIABILITY', 'CREDIT', TRUE, '00000000-0000-0000-0002-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 2.1.01.001 Fornecedores Nacionais (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0002-000000001001', '2.1.01.001', 'Fornecedores Nacionais', 'LIABILITY', 'CREDIT', FALSE, '00000000-0000-0000-0002-000000000100', TRUE)
      ON CONFLICT (code) DO NOTHING;

    -- 2.1.02 Obrigações Fiscais e Tributárias (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0002-000000000200', '2.1.02', 'Obrigações Fiscais e Tributárias', 'LIABILITY', 'CREDIT', TRUE, '00000000-0000-0000-0002-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 2.1.02.001 Simples Nacional a Recolher (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0002-000000002001', '2.1.02.001', 'Simples Nacional a Recolher', 'LIABILITY', 'CREDIT', FALSE, '00000000-0000-0000-0002-000000000200', TRUE)
      ON CONFLICT (code) DO NOTHING;

    -- 2.1.03 Obrigações Trabalhistas e Sociais (Sintética)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0002-000000000300', '2.1.03', 'Obrigações Trabalhistas e Sociais', 'LIABILITY', 'CREDIT', TRUE, '00000000-0000-0000-0002-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

      -- 2.1.03.001 Salários a Pagar (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0002-000000003001', '2.1.03.001', 'Salários a Pagar', 'LIABILITY', 'CREDIT', FALSE, '00000000-0000-0000-0002-000000000300', TRUE)
      ON CONFLICT (code) DO NOTHING;

      -- 2.1.03.002 FGTS a Recolher (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0002-000000003002', '2.1.03.002', 'FGTS a Recolher', 'LIABILITY', 'CREDIT', FALSE, '00000000-0000-0000-0002-000000000300', TRUE)
      ON CONFLICT (code) DO NOTHING;

      -- 2.1.03.003 INSS a Recolher (Analítica)
      INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
      VALUES ('00000000-0000-0000-0002-000000003003', '2.1.03.003', 'INSS a Recolher', 'LIABILITY', 'CREDIT', FALSE, '00000000-0000-0000-0002-000000000300', TRUE)
      ON CONFLICT (code) DO NOTHING;

-- 3. PATRIMÔNIO LÍQUIDO (Sintética)
INSERT INTO account (id, code, name, type, nature, is_synthetic, active)
VALUES ('00000000-0000-0000-0003-000000000000', '3', 'PATRIMÔNIO LÍQUIDO', 'EQUITY', 'CREDIT', TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

  -- 3.1 Capital Social (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0003-000000000010', '3.1', 'Capital Social', 'EQUITY', 'CREDIT', TRUE, '00000000-0000-0000-0003-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 3.1.01 Capital Subscrito (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0003-000000001001', '3.1.01', 'Capital Subscrito e Integralizado', 'EQUITY', 'CREDIT', FALSE, '00000000-0000-0000-0003-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

  -- 3.2 Lucros ou Prejuízos Acumulados (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0003-000000000020', '3.2', 'Lucros ou Prejuízos Acumulados', 'EQUITY', 'CREDIT', TRUE, '00000000-0000-0000-0003-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 3.2.01 Lucros Acumulados (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0003-000000002001', '3.2.01', 'Lucros Acumulados', 'EQUITY', 'CREDIT', FALSE, '00000000-0000-0000-0003-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;

-- 4. RECEITAS (Sintética)
INSERT INTO account (id, code, name, type, nature, is_synthetic, active)
VALUES ('00000000-0000-0000-0004-000000000000', '4', 'RECEITAS', 'REVENUE', 'CREDIT', TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

  -- 4.1 Receita Bruta das Vendas e Serviços (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0004-000000000010', '4.1', 'Receita Bruta das Vendas e Serviços', 'REVENUE', 'CREDIT', TRUE, '00000000-0000-0000-0004-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 4.1.01.001 Receita de Prestação de Serviços (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0004-000000001001', '4.1.01.001', 'Receita de Prestação de Serviços', 'REVENUE', 'CREDIT', FALSE, '00000000-0000-0000-0004-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

    -- 4.1.01.002 Receita de Venda de Mercadorias (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0004-000000001002', '4.1.01.002', 'Receita de Venda de Mercadorias', 'REVENUE', 'CREDIT', FALSE, '00000000-0000-0000-0004-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

-- 5. DESPESAS E CUSTOS (Sintética)
INSERT INTO account (id, code, name, type, nature, is_synthetic, active)
VALUES ('00000000-0000-0000-0005-000000000000', '5', 'DESPESAS E CUSTOS', 'EXPENSE', 'DEBIT', TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

  -- 5.1 Custos das Mercadorias e Serviços (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0005-000000000010', '5.1', 'Custos das Mercadorias e Serviços', 'EXPENSE', 'DEBIT', TRUE, '00000000-0000-0000-0005-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 5.1.01.001 Custo das Mercadorias Vendidas - CMV (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0005-000000001001', '5.1.01.001', 'Custo das Mercadorias Vendidas - CMV', 'EXPENSE', 'DEBIT', FALSE, '00000000-0000-0000-0005-000000000010', TRUE)
    ON CONFLICT (code) DO NOTHING;

  -- 5.2 Despesas Operacionais e Administrativas (Sintética)
  INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
  VALUES ('00000000-0000-0000-0005-000000000020', '5.2', 'Despesas Operacionais e Administrativas', 'EXPENSE', 'DEBIT', TRUE, '00000000-0000-0000-0005-000000000000', TRUE)
  ON CONFLICT (code) DO NOTHING;

    -- 5.2.01.001 Despesas com Salários e Ordenados (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0005-000000002001', '5.2.01.001', 'Despesas com Salários e Ordenados', 'EXPENSE', 'DEBIT', FALSE, '00000000-0000-0000-0005-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;

    -- 5.2.01.002 Despesas com Aluguel (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0005-000000002002', '5.2.01.002', 'Despesas com Aluguel', 'EXPENSE', 'DEBIT', FALSE, '00000000-0000-0000-0005-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;

    -- 5.2.01.003 Despesas com Energia Elétrica e Água (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0005-000000002003', '5.2.01.003', 'Despesas com Energia Elétrica e Água', 'EXPENSE', 'DEBIT', FALSE, '00000000-0000-0000-0005-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;

    -- 5.2.01.004 Despesas com Internet e Softwares (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0005-000000002004', '5.2.01.004', 'Despesas com Internet e Softwares', 'EXPENSE', 'DEBIT', FALSE, '00000000-0000-0000-0005-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;

    -- 5.2.01.005 Despesas Financeiras e Tarifas Bancárias (Analítica)
    INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active)
    VALUES ('00000000-0000-0000-0005-000000002005', '5.2.01.005', 'Despesas Financeiras e Tarifas Bancárias', 'EXPENSE', 'DEBIT', FALSE, '00000000-0000-0000-0005-000000000020', TRUE)
    ON CONFLICT (code) DO NOTHING;
