# Documento de Arquitetura: Motor de Partidas Dobradas e Núcleo ERP Contábil (Nortabil)

Este documento detalha as decisões técnicas, modelo de dados relacional, fluxos de concorrência, segurança RBAC e conformidade com as normas contábeis brasileiras adotadas no **Nortabil**.

---

## 1. Visão Arquitetural Completa (Clean Architecture)

O Nortabil foi projetado com base no padrão **Clean Architecture / Arquitetura Hexagonal**, garantindo que as regras de negócio contábeis sejam completamente desacopladas de frameworks de transporte (HTTP) e mecanismos de persistência (PostgreSQL 16).

```mermaid
flowchart TD
    Client["Dashboard Web Contábil / Clientes REST / Importadores XML & OFX"]

    subgraph HTTP_Layer["Camada de Apresentação e Segurança (HTTP / Chi Router)"]
        Router["Chi Router + Middlewares (CORS, Logger, RequestID, AuthJWT, RequireRole, TenantMiddleware)"]
        AuthH["AuthHandler (JWT & RBAC)"]
        CompH["CompanyHandler (Multi-Tenant)"]
        AccH["AccountHandler (Plano de Contas)"]
        JournH["JournalHandler (Diário & Estorno)"]
        NFeH["NFeHandler (XML SEFAZ Mod. 55)"]
        RecH["ReconciliationHandler (Extrato OFX)"]
        PerH["PeriodHandler (Trava Mensal & ARE)"]
        RepH["ReportHandler (Razão, Balancete, DRE, BP, DFC, Indicadores)"]
        SpedH["SPEDHandler (ECD & Pré-Validador PVA)"]
    end

    subgraph Service_Layer["Camada de Serviços (Orquestração de Regras Contábeis)"]
        AuthS["AuthService (Bcrypt & HMAC-SHA256)"]
        CompS["CompanyService (Validação CNPJ & Clone SPED)"]
        AccS["AccountService (Árvore Sintética/Analítica)"]
        JournS["JournalService (Lei de Luca Pacioli, Lotes Concorrentes & Estorno)"]
        NFeS["NFeService (Classificação Automática Entrada/Saída)"]
        OFXS["OFXService (Deduplicação FITID & Partidas)"]
        PerS["PeriodService (Fechamento & Zeramento ARE)"]
        RepS["ReportService & ExportService (CSV BOM & Impressão A4/PDF)"]
        SpedS["SPEDService (Blocos 0, I, J, 9 & PVA Interno)"]
    end

    subgraph Domain_Layer["Camada de Domínio Contábil Puro (Zero Dependências Externas)"]
        Entities["Company, User, Account, JournalEntry, JournalLine, AccountingPeriod, NFeDocument, BankStatementItem"]
        Reports["LedgerReport, TrialBalance, DREReport, BalanceSheet, CashFlowReport (DFC), FinancialIndicators"]
    end

    subgraph Infra_Layer["Camada de Persistência Transacional (PostgreSQL 16 + pgxpool)"]
        DB["pgxpool Connection Pool (Protocolo Binário)"]
        Trig["Constraint Trigger Diferido (DEFERRABLE INITIALLY DEFERRED)"]
    end

    Client --> Router
    Router --> AuthH & CompH & AccH & JournH & NFeH & RecH & PerH & RepH & SpedH
    AuthH --> AuthS
    CompH --> CompS
    AccH --> AccS
    JournH --> JournS
    NFeH --> NFeS
    RecH --> OFXS
    PerH --> PerS
    RepH --> RepS
    SpedH --> SpedS
    AuthS & CompS & AccS & JournS & NFeS & OFXS & PerS & RepS & SpedS --> Domain_Layer
    AuthS & CompS & AccS & JournS & OFXS & PerS & RepS & SpedS --> DB
    DB --> Trig
```

---

## 2. Modelagem do Banco de Dados Relacional Multi-Tenant (PostgreSQL 16)

```mermaid
erDiagram
    company ||--o{ account : "owns"
    company ||--o{ journal_entry : "records"
    company ||--o{ accounting_period : "controls"
    company ||--o{ bank_statement_item : "imports"
    account ||--o{ account : "parent_of"
    account ||--o{ journal_line : "receives"
    journal_entry ||--|{ journal_line : "contains"

    company {
        uuid id PK
        varchar cnpj UK "CNPJ validado matematicamente"
        varchar legal_name "Razão Social"
        varchar trade_name "Nome Fantasia"
        varchar tax_regime "SIMPLES_NACIONAL, LUCRO_PRESUMIDO, LUCRO_REAL, MEI"
        varchar city "Município (ex: Januária, Jaíba, Montes Claros)"
        varchar state "UF (MG)"
        boolean active
        timestamptz created_at
    }

    app_user {
        uuid id PK
        varchar name "Nome completo do profissional"
        varchar email UK "E-mail corporativo"
        varchar password_hash "Hash Bcrypt (custo 10)"
        varchar role "ADMIN, CONTADOR, ASSISTENTE"
        varchar crc_number "Registro no CRC/MG"
        boolean active
    }

    account {
        uuid id PK
        uuid company_id FK "Isolamento Multi-Tenant"
        varchar code "Código hierárquico (ex: 1.1.01.001)"
        varchar name "Nome da conta"
        varchar type "ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE"
        varchar nature "DEBIT ou CREDIT"
        boolean is_synthetic "true = Sintética, false = Analítica"
        uuid parent_id FK "Conta Pai"
        boolean active
    }

    journal_entry {
        uuid id PK
        uuid company_id FK "Isolamento Multi-Tenant"
        bigint entry_number "Sequência cronológica no Diário"
        date date "Data de competência"
        text description "Histórico completo (NBC ITG 2000)"
        varchar status "POSTED, DRAFT, REVERSED"
        boolean is_closing "Flag de lançamento de encerramento ARE"
        varchar created_by "Trilha de auditoria do usuário autenticado"
        numeric total_debit "Soma exata dos débitos"
        numeric total_credit "Soma exata dos créditos"
    }

    journal_line {
        uuid id PK
        uuid journal_entry_id FK "Lançamento contábil pai"
        uuid account_id FK "Conta analítica movimentada"
        varchar type "DEBIT ou CREDIT"
        numeric amount "Valor decimal exato (> 0)"
        text description "Histórico complementar da partida"
    }

    accounting_period {
        uuid id PK
        uuid company_id FK "Empresa"
        int year "Exercício fiscal"
        int month "Mês de competência (1..12)"
        varchar status "OPEN, CLOSED, LOCKED"
        varchar closed_by "Contador responsável (CRC)"
        timestamptz closed_at
    }

    bank_statement_item {
        uuid id PK
        uuid company_id FK "Empresa"
        uuid bank_account_id FK "Conta bancária analítica"
        varchar fitid "Identificador único OFX (anti-duplicidade)"
        date date "Data da transação bancária"
        numeric amount "Valor absoluto"
        varchar type "DEBIT ou CREDIT"
        boolean reconciled "Status de conciliação"
        uuid journal_entry_id FK "Lançamento gerado no Diário"
    }
```

---

## 3. Dupla Barreira de Validação da Lei de Luca Pacioli e Imutabilidade Contábil

A regra fundamental $\sum \text{Débitos} = \sum \text{Créditos}$ e a proibição de alteração direta de lançamentos (NBC ITG 2000) são garantidas em dois níveis complementares:

### Nível 1: Validação Prévia na Aplicação (Go + `shopspring/decimal`)
Antes de abrir qualquer transação no banco de dados, o método `ValidateDoubleEntry()` da entidade `JournalEntry` confere em memória com precisão decimal de ponto fixo:
1. Pelo menos duas partidas presentes no lançamento.
2. Cada valor estritamente positivo ($> 0$).
3. Presença obrigatória de partidas a débito e a crédito.
4. Igualdade matemática exata entre a soma de débitos e créditos ($\sum D - \sum C = 0,00$).
5. Verificação de que o período contábil (`accounting_period`) está aberto (`OPEN`) e de que todas as contas pertencem à mesma empresa (`company_id`), estão **ativas** e são **analíticas** (`is_synthetic = FALSE`).
6. Para retificações, o método `BuildReversal()` gera um lançamento inverso espelhado (`DEBIT <-> CREDIT`), preservando a trilha histórica exigida pela NBC ITG 2000 (item 31).

### Nível 2: Garantia Transacional no Banco de Dados (PostgreSQL 16)
Todas as operações de escrita ocorrem dentro de transações ACID (`BEGIN ... COMMIT`). O PostgreSQL executa o gatilho de restrição diferido:
```sql
CREATE CONSTRAINT TRIGGER trg_enforce_double_entry_balance
AFTER INSERT OR UPDATE OR DELETE ON journal_line
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION check_journal_entry_balance();
```
* **DEFERRABLE INITIALLY DEFERRED:** O gatilho é disparado no instante exato do `COMMIT`, somando todas as linhas do lançamento no banco e rejeitando qualquer violação com `ROLLBACK` imediato.

---

## 4. Concorrência, Segurança RBAC e Alto Desempenho

* **Worker Pool com Goroutines:** O endpoint `/api/v1/journal-entries/batch` processa lotes massivos de lançamentos utilizando goroutines coordenadas por semáforo, atingindo mais de **1 milhão de validações por segundo** sob concorrência.
* **Autenticação JWT e RBAC:** Senhas protegidas com Bcrypt (custo 10) e tokens JWT assinados via HMAC-SHA256, segregando permissões entre `ADMIN` (gestão de usuários), `CONTADOR` (fechamento de períodos, ARE e assinatura SPED ECD com CRC) e `ASSISTENTE` (escrituração operacional).
* **Automação Fiscal Regional (OFX + NF-e SEFAZ Modelo 55):** Importadores nativos para extratos bancários (`.OFX` SGML/XML) e Notas Fiscais Eletrônicas (`.XML` `<nfeProc>`), eliminando erros de digitação manual nos escritórios contábeis do Norte de Minas Gerais.
