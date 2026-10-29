# Documento de Arquitetura: Motor de Partidas Dobradas (Nortabil)

Este documento detalha as decisões técnicas, modelo de dados, fluxos de concorrência e conformidade com as normas contábeis brasileiras adotadas no **Nortabil**.

---

## 1. Visão Arquitetural

O Nortabil foi projetado com base no padrão **Clean Architecture / Arquitetura Hexagonal**, garantindo que as regras de negócio contábeis sejam completamente desacopladas de frameworks de transporte (HTTP) e mecanismos de persistência (PostgreSQL).

```mermaid
flowchart TD
    Client["Clientes Externos / ERP Frontend / Fechamento Mensal"]
    
    subgraph HTTP_Layer["Camada de Apresentação (HTTP / REST)"]
        Router["Chi Router & Middlewares (CORS, Logger, RequestID)"]
        AccH["AccountHandler"]
        JournH["JournalHandler"]
        RepH["ReportHandler"]
    end
    
    subgraph Service_Layer["Camada de Serviços (Regras Contábeis)"]
        AccS["AccountService"]
        JournS["JournalService (Lei de Luca Pacioli & Lotes Concorrentes)"]
        RepS["ReportService"]
    end
    
    subgraph Domain_Layer["Camada de Domínio (Entidades Puras)"]
        Account["Account (Árvore Hierárquica)"]
        JournalEntry["JournalEntry & JournalLine (Partidas Dobradas)"]
        Reports["Ledger / DailyBook / TrialBalance"]
    end
    
    subgraph Infra_Layer["Camada de Persistência (PostgreSQL 16)"]
        DB["pgxpool Connection Pool"]
        AccRepo["AccountRepository"]
        JournRepo["JournalRepository (Transações ACID)"]
        RepRepo["ReportRepository (Queries Agregadas)"]
        Trig["Constraint Trigger Diferido (Balanceamento)"]
    end

    Client --> Router
    Router --> AccH & JournH & RepH
    AccH --> AccS
    JournH --> JournS
    RepH --> RepS
    AccS & JournS & RepS --> Domain_Layer
    AccS --> AccRepo
    JournS --> JournRepo & AccRepo
    RepS --> RepRepo
    AccRepo & JournRepo & RepRepo --> DB
    DB --> Trig
```

---

## 2. Modelagem do Banco de Dados Relacional (PostgreSQL)

```mermaid
erDiagram
    account ||--o{ account : "parent_of"
    account ||--o{ journal_line : "records"
    journal_entry ||--|{ journal_line : "contains"

    account {
        uuid id PK
        varchar code UK "Código estruturado (ex: 1.1.01.001)"
        varchar name "Nome da conta"
        varchar type "ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE"
        varchar nature "DEBIT ou CREDIT"
        boolean is_synthetic "Sintética ou Analítica"
        uuid parent_id FK "Conta Pai na hierarquia"
        boolean active "Ativa para lançamentos"
        timestamptz created_at
        timestamptz updated_at
    }

    journal_entry {
        uuid id PK
        bigint entry_number UK "Sequência cronológica do Diário"
        date date "Data de competência"
        text description "Histórico do fato contábil"
        varchar status "POSTED, DRAFT, REVERSED"
        numeric total_debit "Totalizador de débitos"
        numeric total_credit "Totalizador de créditos"
        timestamptz created_at
        timestamptz updated_at
    }

    journal_line {
        uuid id PK
        uuid journal_entry_id FK "Lançamento pai"
        uuid account_id FK "Conta contábil analítica"
        varchar type "DEBIT ou CREDIT"
        numeric amount "Valor da partida (maior que zero)"
        text description "Histórico individual da partida"
        timestamptz created_at
    }
```

---

## 3. Dupla Validação da Lei de Luca Pacioli

A regra imutável $\sum \text{Débitos} = \sum \text{Créditos}$ é garantida em dois níveis complementares:

### Nível 1: Validação Prévia na Aplicação (Go)
Antes de abrir qualquer transação no banco de dados, o método `ValidateDoubleEntry()` da entidade `JournalEntry` confere em memória com precisão arbitrária:
1. Pelo menos duas partidas presentes.
2. Cada valor estritamente positivo ($> 0$).
3. Presença de partidas a débito e a crédito.
4. Igualdade matemática exata entre a soma de débitos e créditos.
5. Se todas as contas informadas são do tipo **analítico** e estão **ativas**.

### Nível 2: Garantia Transacional no Banco de Dados (PostgreSQL)
Todas as operações de inserção são envolvidas em um bloco `BEGIN ... COMMIT`. O PostgreSQL executa o gatilho:
```sql
CREATE CONSTRAINT TRIGGER trg_enforce_double_entry_balance
AFTER INSERT OR UPDATE OR DELETE ON journal_line
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION check_journal_entry_balance();
```
* **DEFERRABLE INITIALLY DEFERRED:** O trigger só é executado no momento exato do `COMMIT`. Isso permite que a aplicação insira a linha de débito e em seguida a de crédito dentro da mesma transação. Ao final, o banco soma todas as partidas e valida a igualdade e o tipo analítico. Se houver divergência, o banco cancela a transação com `ROLLBACK` total.

---

## 4. Concorrência e Alto Desempenho

Para atender aos escritórios de contabilidade durante os fechamentos mensais:
* **Worker Pool com Goroutines:** O endpoint `/api/v1/journal-entries/batch` recebe milhares de transações e distribui a validação e processamento entre múltiplos workers controlados por semáforos, evitando exaustão de conexões.
* **Driver pgx v5 com pgxpool:** Utiliza protocolo binário nativo do PostgreSQL e pool de conexões reutilizáveis, reduzindo o overhead de conexões TCP.
* **Evitação de Travamentos (Deadlocks):** Contas são referenciadas exclusivamente para leitura de integridade analítica e as partidas são inseridas de forma independente, garantindo que múltiplos lotes possam ser processados concorrentemente.

