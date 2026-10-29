-- =========================================================================
-- MIGRAÇÃO 006: CONTROLE DE ACESSO BASEADO EM FUNÇÕES (RBAC) E AUTENTICAÇÃO
-- Escrituração Contábil Segura - Conformidade CFC e Normas de Auditoria
-- =========================================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    email VARCHAR(150) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(30) NOT NULL CHECK (role IN ('ADMIN', 'CONTADOR', 'ASSISTENTE')),
    crc_number VARCHAR(50),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Seed de usuários padrão caso a tabela esteja vazia (Senha inicial: Nortabil@2026)
-- Hash Bcrypt custo 10 de 'Nortabil@2026': $2a$10$wOAlcG7i8ulr/tuKMyYrPeRG8Ku9DioZfhuWk9eI5sAlwVr2llbVy
INSERT INTO users (id, name, email, password_hash, role, crc_number, active)
VALUES 
    (
        'a1b2c3d4-e5f6-4a1b-8c2d-000000000001',
        'Administrador Nortabil',
        'admin@nortabil.com.br',
        '$2a$10$wOAlcG7i8ulr/tuKMyYrPeRG8Ku9DioZfhuWk9eI5sAlwVr2llbVy',
        'ADMIN',
        NULL,
        TRUE
    ),
    (
        'a1b2c3d4-e5f6-4a1b-8c2d-000000000002',
        'Sebastião Ferreira da Silva (Contador)',
        'contador@nortabil.com.br',
        '$2a$10$wOAlcG7i8ulr/tuKMyYrPeRG8Ku9DioZfhuWk9eI5sAlwVr2llbVy',
        'CONTADOR',
        'MG-085421/O',
        TRUE
    ),
    (
        'a1b2c3d4-e5f6-4a1b-8c2d-000000000003',
        'Ana Paula Ribeiro (Assistente Contábil)',
        'assistente@nortabil.com.br',
        '$2a$10$wOAlcG7i8ulr/tuKMyYrPeRG8Ku9DioZfhuWk9eI5sAlwVr2llbVy',
        'ASSISTENTE',
        NULL,
        TRUE
    )
-- DO NOTHING: as migrações rodam a cada inicialização; não sobrescrever senhas alteradas pelos usuários
ON CONFLICT (email) DO NOTHING;

-- Rastreabilidade / Trilha de Auditoria do Usuário no Livro Diário
ALTER TABLE journal_entry
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(150) NOT NULL DEFAULT 'Sistema Contábil';

