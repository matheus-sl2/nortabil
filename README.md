# NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3.0-6BA539?style=flat&logo=swagger)](http://localhost:8080/swagger)
[![License](https://img.shields.io/badge/Status-TCC%20Concluído-success)](#)

> **Núcleo transacional contábil em nuvem de altíssimo desempenho, estritamente balanceado pela Lei de Luca Pacioli e aderente às normas contábeis brasileiras (Lei 6.404/76, Código Civil e CFC ITG 2000).**

---

## 1. Visão Geral e Justificativa

O **Nortabil** é o motor de partidas dobradas projetado como a espinha dorsal de um futuro ERP Contábil em Nuvem voltado para a realidade dos escritórios de contabilidade do **Norte de Minas Gerais** (como Januária, Montes Claros e cidades circunvizinhas).

### O Problema
A grande maioria dos escritórios contábeis da região ainda depende de sistemas contábeis legados (desktop, monolíticos ou virtualizados via Terminal Services). Durante períodos de pico — como o fechamento mensal ou a entrega de obrigações acessórias (SPED/ECD/ECF) —, esses sistemas sofrem lentidão severa ou travamentos ao processar lotes de milhares de notas fiscais, cupons e recibos bancários.

### A Solução
O Nortabil substitui a arquitetura legada por um **núcleo de microprecisão transacional** em Go (Golang) e PostgreSQL, combinando:
1. **Concorrência Nativa e Baixa Latência:** Processamento de lotes de milhares de lançamentos em milissegundos através de goroutines e pools de conexão assíncronos.
2. **Garantia Absoluta de Integridade (ACID):** A clássica equação patrimonial é aplicada tanto a nível de memória da aplicação quanto em transações PostgreSQL com gatilhos de restrição diferidos (*DEFERRABLE INITIALLY DEFERRED*).
3. **Precisão Decimal Rígida:** Eliminação absoluta de ponto flutuante (`float64`), adotando aritmética de precisão arbitrária (`shopspring/decimal`).
4. **Legislação Brasileira Embutida:** Conformidade estrutural com a Lei das S.A. (Lei 6.404/1976), o Código Civil Brasileiro (Lei 10.406/2002) e a Resolução CFC nº 1.330/2011 (ITG 2000 - Escrituração Contábil).

---

## 2. A Teoria Contábil e Fundamentação Legal

### 2.1 A Lei de Frei Luca Pacioli (1494)
O Teorema Fundamental da Contabilidade estabelece que todo evento contábil produz um efeito simétrico e simultâneo:
$$\sum \text{Débitos} = \sum \text{Créditos}$$

No Nortabil:
- Não é permitida a persistência de nenhum lançamento contábil em que $\sum \text{Débitos} \neq \sum \text{Créditos}$.
- Cada partida deve ter valor estritamente positivo ($> 0$).
- Devem existir no mínimo duas partidas em cada lançamento (1 Débito / 1 Crédito, 1 Débito / múltiplos Créditos, múltiplos Débitos / 1 Crédito ou múltiplos Débitos / múltiplos Créditos).

### 2.2 Legislação Contábil Brasileira (Repositório de Leis)
* **Lei Federal nº 6.404/1976 (Lei das Sociedades por Ações):**
  * *Artigos 176 a 188:* Estruturação de balanço patrimonial com separação entre Ativo (Circulante e Não Circulante), Passivo (Circulante e Não Circulante), Patrimônio Líquido, Receitas e Despesas.
* **Código Civil Brasileiro (Lei Federal nº 10.406/2002):**
  * *Artigos 1.179 a 1.184:* Obrigatoriedade da escrituração uniforme, cronológica e individualizada, em moeda corrente nacional, com emissão indispensável do Livro Diário e Livro Razão.
* **Resolução CFC nº 1.330/2011 (NBC ITG 2000 - Escrituração Contábil):**
  * **Contas Sintéticas vs. Analíticas:** Contas sintéticas funcionam apenas como totalizadoras de grupo. **É expressamente vedado efetuar lançamentos contábeis diretamente em contas sintéticas.** Os lançamentos são registrados exclusivamente nas contas analíticas (folhas da árvore hierárquica). O Nortabil bloqueia lançamentos em contas sintéticas em tempo de execução e a nível de banco de dados.

---

## 3. Arquitetura do Sistema

O projeto adota os princípios da **Clean Architecture** (Arquitetura Limpa), separando regras de negócio puras de detalhes de infraestrutura e persistência:

```
c:\Projeto NORTABIL\
├── cmd/
│   └── api/
│       └── main.go                 # Ponto de entrada, injeção de dependências e graceful shutdown
├── docs/
│   ├── swagger.yaml                # Especificação OpenAPI 3.0 completa
│   ├── docs.go                     # Embed estático da documentação no binário
│   └── ARQUITETURA.md              # Detalhamento arquitetural
├── internal/
│   ├── config/                     # Gestão de configurações e variáveis de ambiente
│   ├── domain/                     # Modelos de domínio, entidades e validações de Pacioli
│   │   ├── account.go              # Plano de contas, tipos (Ativo, Passivo, PL, etc.)
│   │   ├── journal.go              # Lançamentos contábeis e partidas dobradas
│   │   ├── report.go               # Modelos do Livro Razão, Diário e Balancete
│   │   └── errors.go               # Erros canônicos contábeis
│   ├── repository/
│   │   └── postgres/               # Persistência de alto desempenho com pgxpool
│   │       ├── db.go               # Pool e auto-migrações
│   │       ├── account_repo.go     # CRUD e queries hierárquicas de contas
│   │       ├── journal_repo.go     # Transações ACID com rollback automático
│   │       ├── report_repo.go      # Queries agregadas de Razão, Diário e Balancete
│   │       └── migrations/         # Scripts DDL e Carga Inicial (Seed SPED/ECD)
│   ├── service/                    # Camada de regras de negócio contábeis
│   │   ├── account_service.go      # Validação de hierarquia e árvore de contas
│   │   ├── journal_service.go      # Motor de partidas dobradas e lotes concorrentes
│   │   └── report_service.go       # Emissão de livros contábeis e balancete
│   ├── handler/                    # Handlers RESTful HTTP (chi router)
│   └── middleware/                 # CORS, Logger estruturado, RequestID e Recovery
├── scripts/
│   └── demo_data.go                # Carga de fatos contábeis reais para demonstração
├── tests/
│   ├── double_entry_test.go        # Testes de partidas dobradas e precisão decimal
│   └── concurrent_test.go          # Testes de concorrência com 5.000 lançamentos simultâneos
├── docker-compose.yml              # Orquestração da API Go + PostgreSQL 16
├── Dockerfile                      # Multi-stage build compilado com CGO=0
├── Makefile                        # Automação de tarefas
├── go.mod / go.sum                 # Módulos e dependências Go
└── README.md                       # Documentação principal
```

---

## 4. Endpoints Principais da API RESTful

A documentação interativa completa (Swagger UI) está disponível em `http://localhost:8080/swagger`.

### 4.1 Plano de Contas
| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/api/v1/accounts` | Cadastra nova conta contábil (sintética ou analítica) |
| `GET` | `/api/v1/accounts` | Lista contas (parâmetro `?tree=true` para formato em árvore) |
| `GET` | `/api/v1/accounts/{id}` | Busca conta contábil pelo UUID |
| `GET` | `/api/v1/accounts/code/{code}` | Busca conta pelo código estruturado (ex: `1.1.01.001`) |

### 4.2 Lançamentos Contábeis (Partidas Dobradas)
| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/api/v1/journal-entries` | Registra lançamento contábil individual com validação de Luca Pacioli |
| `POST` | `/api/v1/journal-entries/batch` | Ingestão em lote de alta performance com worker pool concorrente |
| `GET` | `/api/v1/journal-entries` | Lista lançamentos paginados com filtros por período e status |
| `GET` | `/api/v1/journal-entries/{id}` | Retorna detalhes completos do lançamento com suas partidas |

### 4.3 Relatórios e Livros Legais (Além do Mínimo Viável)
| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/api/v1/reports/ledger` | **Livro Razão:** Emite o razão analítico com saldos anteriores e movimentações |
| `GET` | `/api/v1/reports/journal` | **Livro Diário Geral:** Relação cronológica oficial de todos os lançamentos |
| `GET` | `/api/v1/reports/trial-balance` | **Balancete de Verificação:** Balancete hierárquico com conferência de débitos/créditos |

### 4.4 Operação e Monitoramento
| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/healthz` | Health check da API e conectividade com o banco de dados |
| `GET` | `/swagger` | Interface visual do Swagger UI para experimentação de rotas |

---

## 5. Como Executar o Projeto

### Opção A: Execução Conteinerizada via Docker Compose (Recomendado)

Suba todo o ecossistema (PostgreSQL 16 configurado + API Nortabil compilada):

```bash
docker compose up -d --build
```

Acesse:
* API e Health Check: `http://localhost:8080/healthz`
* Documentação Swagger: `http://localhost:8080/swagger`

Para parar:
```bash
docker compose down
```

---

### Opção B: Execução Local em Ambiente Go

1. **Inicie uma instância do PostgreSQL 16** (local ou via Docker):
   ```bash
   docker run --name nortabil-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=nortabil_db -p 5432:5432 -d postgres:16-alpine
   ```

2. **Execute as variáveis de ambiente e suba a API:**
   ```bash
   go run ./cmd/api
   ```

---

## 6. Execução de Testes Automatizados

O Nortabil conta com testes unitários, de precisão decimal e de concorrência massiva:

```bash
go test -v ./tests/...
```

### Resultados dos Testes de Concorrência:
```
=== RUN   TestConcurrentDoubleEntryValidation
    concurrent_test.go: Processados 5.000 lançamentos em 2.79ms (Vazão: 1.789.036 ops/seg)
--- PASS: TestConcurrentDoubleEntryValidation
=== RUN   TestTrialBalanceIntegrity
--- PASS: TestTrialBalanceIntegrity
=== RUN   TestDoubleEntryBalance
--- PASS: TestDoubleEntryBalance
=== RUN   TestAccountNatureDerivation
--- PASS: TestAccountNatureDerivation
=== RUN   TestPrecisionDecimalMath
--- PASS: TestPrecisionDecimalMath
PASS
```

---

## 7. Carga de Demonstração (Fatos Contábeis Reais)

Para popular a base de dados com cenários reais de uma empresa do Norte de Minas Gerais (integralização de capital, compras a prazo, vendas à vista/a prazo, apropriação de folha de pagamento, despesas operacionais e cálculo de CMV):

```bash
go run ./scripts/demo_data.go
```

Após rodar o script, você pode consultar o **Balancete de Verificação** e o **Livro Razão** diretamente no Swagger (`/swagger`) ou via curl:

```bash
curl -X GET "http://localhost:8080/api/v1/reports/trial-balance"
```

---

## 8. Repositório e Versionamento

* **Repositório do Projeto:** [github.com/matheus-sl2/nortabil](https://github.com/matheus-sl2/nortabil.git)
* **Repositório de Legislação de Consulta:** [github.com/ctresb/leis](https://github.com/ctresb/leis.git)
* **Idioma oficial de commits e comentários:** Português do Brasil (pt-BR).
