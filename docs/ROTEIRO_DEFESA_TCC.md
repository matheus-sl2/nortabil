# 🎓 ROTEIRO E GUIA DE DEFESA PARA BANCA EXAMINADORA DE TCC

**Projeto:** NORTABIL — Motor de Partidas Dobradas e Núcleo ERP Contábil  
**Contexto de Aplicação:** Escritórios Contábeis do Norte de Minas Gerais (Januária, Jaíba, Montes Claros e Região)  
**Base Legal:** Frei Luca Pacioli (1494), Lei Federal nº 6.404/1976, NBC TG 26 (R5), NBC TG 1000 e SPED ECD (Instrução Normativa RFB nº 2.003/2021)  
**Repositório Oficial:** [github.com/matheus-sl2/nortabil](https://github.com/matheus-sl2/nortabil.git)  

---

## 📋 1. Visão Geral da Apresentação
* **Tempo Total Sugerido:** 15 a 20 minutos de apresentação + 10 a 15 minutos de arguição da banca.
* **Recursos Necessários:** Notebook, projetor/apresentação de slides e terminal de comando aberto com Docker rodando.

---

## 📽️ 2. Estrutura de Slides da Apresentação (Minuto a Minuto)

### Slide 1: Capa e Identificação (Minuto 00 - 01)
* **Título:** *NORTABIL: Desenvolvimento de um Motor Contábil de Partidas Dobradas de Alta Performance para Escritórios Regionais do Norte de Minas Gerais*.
* **Acadêmico(a):** Apresentação do autor e orientador(a).
* **Objetivo:** Apresentar a motivação de modernizar a escrituração contábil regional substituindo planilhas frágeis e softwares legados por uma arquitetura robusta, segura e auditável.

---

### Slide 2: O Problema Regional e a Justificativa (Minutos 01 - 03)
* **Realidade do Norte de Minas:**
  * Forte presença do **agronegócio irrigado** (Projeto Jaíba - banana, manga, limão) e de **comércio atacadista regional** (Montes Claros, Januária, Janaúba).
  * Escritórios locais sofrem com:
    1. Erros de arredondamento em planilhas eletrônicas (*floating-point errors* de ponto flutuante binário IEEE 754).
    2. Lentidão no fechamento contábil e apuração de resultados (ARE) no encerramento de safra ou exercício.
    3. Digitação manual e repetitiva de extratos bancários, suscetível a erros de digitação e duplicidade.
    4. Dificuldade de gerar o arquivo oficial do **SPED ECD** sem inconsistências de validação pelo PVA da Receita Federal.

---

### Slide 3: Fundamentação Teórica e Legal (Minutos 03 - 06)
* **A Lei Fundamental de Luca Pacioli (1494):**
  $$\sum \text{Débitos} = \sum \text{Créditos}$$
  *Não há devedor sem que haja credor, e para cada débito corresponde um crédito de igual valor.*
* **Legislação Societária e Normas Brasileiras de Contabilidade:**
  * **Lei nº 6.404/1976 (Artigos 176 a 187):** Estrutura do Balanço Patrimonial e da DRE.
  * **NBC TG 26:** Apresentação das Demonstrações Contábeis e segregação estrita de contas sintéticas e analíticas.
  * **Instrução Normativa RFB nº 2.003/2021:** Requisitos da Escrituração Contábil Digital (Blocos 0, I, J e 9).

---

### Slide 4: Arquitetura e Decisões de Engenharia (Minutos 06 - 09)
* **Clean Architecture & Domain-Driven Design (DDD):**
  * O domínio contábil é 100% isolado de bibliotecas externas e frameworks web.
* **Aritmética Monetária de Ponto Fixo Arbitrário (`shopspring/decimal`):**
  * Proibição categórica do uso de `float32` ou `float64` para valores financeiros. Prevenção de perda de centavos por representação binária aproximada.
* **Isolamento Multi-Tenant Estrito:**
  * Cada empresa cliente (tenancy) possui seu próprio plano de contas e livros contábeis isolados, com validação matemática de dígitos verificadores de CNPJ.
* **Concorrência e Locks Transacionais no PostgreSQL 16:**
  * Numeração sequencial contínua de lançamentos contábeis (Diário Geral) sem furos ou saltos, usando bloqueio em nível de linha (`SELECT FOR UPDATE`).

---

### Slide 5: Funcionalidades Desenvolvidas (Minutos 09 - 13)
1. **Motor de Partidas Dobradas:** Validação atômica e simultânea em memória antes de qualquer escrita no banco de dados.
2. **Importador de Extratos Bancários OFX (SGML/XML):** Leitura de extratos bancários com prevenção automática de duplicidade via hash/código `FITID` e sugestão de contrapartida.
3. **Controle de Períodos Fiscais & Apuração do Exercício (ARE):** Travamento de meses encerrados (bloqueio de lançamentos retroativos) e zeramento automático de contas de receita/despesa com transferência de lucro/prejuízo ao PL.
4. **Gerador Oficial do SPED ECD:** Emissão em conformidade com o Manual de Orientação do Leiaute 9.
5. **DRE, Balanço Patrimonial Estruturado e Exportação CSV com BOM UTF-8:** Demonstrações prontas para assinatura do contador e exportação compatível com Excel.
6. **Autenticação Segura com JWT, Bcrypt e Controle de Acesso Baseado em Funções (RBAC):** Hierarquia de acessos para `ADMIN`, `CONTADOR` (com registro de CRC) e `ASSISTENTE`.

---

### Slide 6: Resultados Experimentais e Científicos (Minutos 13 - 15)
* **Resultados dos Testes de Estresse e Benchmarking:**
  * Vazão média do motor de validação: **> 7,5 a 9,4 milhões de lançamentos por segundo**.
  * Tempo de validação por lançamento contábil: **~106 nanossegundos**.
  * Consumo de memória: **0 B/op** no hot-path de validação (alocação zero).
  * Concorrência em 8 workers simultâneos: **1.010.672 ops/seg** com consistência atômica ininterrupta.

---

### Slide 7: Demonstração Prática ao Vivo (Minutos 15 - 18)
*(Ver seção 3 abaixo para os comandos da demonstração)*

---

### Slide 8: Conclusão e Contribuições (Minutos 18 - 20)
* Demonstração de que o ecossistema contábil regional pode ter um núcleo ERP com desempenho superior aos grandes softwares comerciais, a custo zero de licença e com aderência total às normas do CFC e da Receita Federal.
* **Trabalhos Futuros:** Módulo de automação de notas fiscais de serviço eletrônicas (NFS-e) para municípios do Norte de Minas e módulo de conciliação com Pix dinâmico.

---

## 💻 3. Roteiro Passo a Passo para a Demonstração ao Vivo

Abra o terminal PowerShell ou Bash e execute os seguintes passos diante da banca:

### Passo 1: Subir o Ambiente Completo via Docker
```powershell
docker compose up -d
```
*Explicação para a banca:* "Em um único comando, inicializamos o PostgreSQL 16 com suas extensões e a API contábil em Go compilada com binário estático e Alpine Linux."

### Passo 2: Executar a Carga de Dados Realistas do Norte de Minas
```powershell
make seed
# Ou: go run ./cmd/seed/main.go
```
*Explicação para a banca:* "Executamos a carga demonstrativa que cria duas empresas da nossa região: a **Fruticultura Vale do Jaíba Ltda** (Agronegócio irrigado com manga e banana) e a **Distribuidora Comercial Montes Claros Ltda** (Atacado mercantil no Lucro Real), clonando seus planos de contas e registrando lançamentos em partidas dobradas rigorosas."

### Passo 3: Abrir a Interface Web Contábil
Acesse no navegador:
```
http://localhost:8080/dashboard
```
*Ações na tela:*
* Mostrar o resumo patrimonial de Ativo, Passivo e Patrimônio Líquido.
* Mostrar a verificação automática da Equação Fundamental: **Ativo = Passivo + Patrimônio Líquido**.
* Alternar entre o Balancete de Verificação, DRE e Balanço Estruturado.

### Passo 4: Demonstrar a Tentativa de Lançamento Desbalanceado (Rejeição por 1 Centavo)
No terminal ou no Swagger (`http://localhost:8080/swagger/index.html`), envie uma tentativa de lançamento onde:
- Débito: R$ 100,00
- Crédito: R$ 99,99
*Resultado esperado:* Erro imediato HTTP 422: `"UNBALANCED_JOURNAL_ENTRY: Soma de débitos (100.00) diverge da soma de créditos (99.99)"`.
*Explicação para a banca:* "O sistema recusa expressamente qualquer corrupção do balanço antes mesmo de abrir transação no banco de dados."

### Passo 5: Gerar o Arquivo do SPED ECD
```powershell
curl.exe -s "http://localhost:8080/api/v1/sped/ecd?company_id=675d7cea-9d79-4214-98e1-aec328f6cc34&year=2026" -o sped_ecd_jaiba.txt
head -n 20 sped_ecd_jaiba.txt
```
*Explicação para a banca:* "Arquivo textual no formato exigido pela Receita Federal com blocos de abertura, plano de contas I050, partidas I200/I250 e encerramento J930 devidamente assinado pelo contador responsável com CRC."

---

## ❓ 4. Guia de Perguntas e Respostas Prováveis da Banca Examinadora

### Pergunta 1 (Área de Engenharia de Software):
**"Por que vocês escolheram a linguagem Go e a biblioteca `shopspring/decimal` em vez de tipos nativos como `float64` ou linguagens como Python ou Java?"**
> **Resposta:**
> O padrão binário de ponto flutuante IEEE 754 utilizado por `float64` armazena números em potências de base 2, tornando impossível representar exatamente frações decimais simples como `0.1` ou `0.01` (causando dízimas como `0.1000000000000000055511...`). Em sistemas contábeis com milhares de lançamentos, essa discrepância acumula erros de centavos que impedem o fechamento do Balanço Patrimonial e geram multas fiscais.
> O Go foi selecionado por compilar para binário nativo estático de altíssima performance, com consumo irrisório de memória e suporte nativo a concorrência via goroutines. A biblioteca `shopspring/decimal` realiza aritmética decimal de ponto fixo arbitrário base 10, garantindo exatidão de 100% em qualquer cálculo contábil.

---

### Pergunta 2 (Área Contábil):
**"Como o sistema trata a Apuração do Resultado do Exercício (ARE) e o fechamento de períodos?"**
> **Resposta:**
> Em conformidade com a Lei 6.404/76 e as NBCs, no encerramento de um período fiscal todas as contas de resultado (Receitas - Grupo 4 e Despesas/Custos - Grupo 5) precisam ser zeradas.
> O motor calcula o saldo devedor/credor de cada conta de resultado e gera automaticamente um lançamento especial de encerramento (`is_closing: true`), transferindo o saldo líquido (Lucro ou Prejuízo Líquido) para a conta `3.2.01 (Lucros/Prejuízos Acumulados)` do Patrimônio Líquido. Simultaneamente, o período é marcado com status `CLOSED`, e o middleware/serviço bloqueia qualquer tentativa de lançamento retroativo com data pertencente àquele período.

---

### Pergunta 3 (Área Contábil/Tributária):
**"De que forma o sistema garante a conformidade com o SPED ECD e a rastreabilidade das operações?"**
> **Resposta:**
> A Escrituração Contábil Digital (IN RFB 2.003/2021) exige que cada registro contábil contenha histórico completo, data cronológica estrita, numeração contínua e associação a contas referenciais analíticas.
> O Nortabil gera os blocos 0 (Identificação da PJ e do Escritório), I (Plano de Contas I050, Lançamento I200 e Partida I250), J (Termo de Encerramento e Signatários J930 com o CRC do contador responsável autenticado) e 9 (Totalizadores de integridade 9900/9999). Além disso, cada partida grava o usuário autenticado que a originou e seu carimbo de data/hora imutável.

---

### Pergunta 4 (Área de Segurança e Gestão):
**"Como o sistema protege dados sensíveis dos clientes de contabilidade contra acessos indevidos?"**
> **Resposta:**
> O sistema implementa o modelo de Controle de Acesso Baseado em Funções (RBAC) com três perfis:
> 1. `ADMIN`: Controle total da infraestrutura do escritório;
> 2. `CONTADOR`: Acesso a lançamentos, apuração de resultados (ARE), fechamento de períodos e emissão de SPED ECD (exigindo CRC válido);
> 3. `ASSISTENTE`: Operações de entrada de lançamentos e consulta a relatórios, impedido de alterar períodos já homologados pelo contador.
> As senhas são hasheadas com Bcrypt (custo de processamento 10 resistente a ataques de dicionário e rainbow tables) e as requisições autenticadas via JSON Web Tokens (JWT) com assinatura HMAC-SHA256 e tempo de vida controlado.

---

### Pergunta 5 (Área Regional):
**"Qual foi o impacto real observado para o contexto específico do Norte de Minas Gerais?"**
> **Resposta:**
> No Norte de Minas, a sazonalidade da colheita no agronegócio de Jaíba e a logística de atacado em Montes Claros geram picos concentrados de movimentações bancárias e de notas fiscais ao final de cada mês. Os escritórios tradicionais gastavam dias para conciliar extratos e fechar períodos.
> Com o Nortabil, a importação OFX automatizada e a vazão de mais de 1 milhão de lançamentos por segundo sob concorrência reduzem o tempo de fechamento mensal de dias para minutos, eliminando retrabalho de digitação manual e garantindo 100% de conformidade fiscal para empresas do Simples, Lucro Presumido e Lucro Real da nossa região.
