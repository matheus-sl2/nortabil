# Relatório de Conformidade Jurídica, LGPD, Direitos do Consumidor e Acessibilidade — NORTABIL

Este documento consolida a governança jurídica, proteção de dados pessoais, revisão de terceiros, direitos autorais e auditoria de acessibilidade do sistema **NORTABIL — Motor de Partidas Dobradas e Núcleo ERP Contábil**.

---

## 1. Identificação Legal do Projeto e Controlador de Dados

- **Denominação do Sistema:** NORTABIL — Sistema ERP Contábil e Motor de Partidas Dobradas
- **Natureza Jurídica:** Projeto Acadêmico e Científico de Engenharia de Software / Trabalho de Conclusão de Curso (TCC) sob Licença Aberta MIT.
- **Sede de Referência Regional:** Januária – MG, CEP 39480-000 (Região Norte de Minas Gerais).
- **CNPJ de Homologação Técnica / Demonstração:** `12.345.678/0001-95` (Fictício e matematicamente válido nos termos da IN RFB nº 2.119/2022, exclusivo para homologação contábil e testes do validador SPED ECD).
- **Responsável Técnico Contábil (Ambiente Demo):** Contador Responsável — Registro `CRC MG-085421/O`.
- **Encarregado pelo Tratamento de Dados Pessoais (DPO — Art. 41 da LGPD):** Matheus S. L. (`dpo.privacidade@nortabil.com.br`).

---

## 2. Política de Privacidade e Adequação à LGPD (Lei nº 13.709/2018)

### 2.1. Princípio da Necessidade e Minimização de Dados (Art. 6º, III da LGPD)
Todos os formulários da aplicação foram revisados para coletar **estritamente o mínimo necessário**:
1. **Autenticação e Cadastro de Usuários (`users`):**
   - Nome completo (identificação na trilha de auditoria contábil);
   - E-mail corporativo (credencial única de login);
   - Senha armazenada exclusivamente sob *hash* criptográfico irreversível (`Bcrypt`);
   - Registro Profissional no CRC (solicitado apenas quando o perfil é `CONTADOR`, em cumprimento ao Decreto-Lei nº 9.295/1946).
   - *Não coletamos CPF pessoal de usuários, endereço residencial, telefone particular, dados biométricos ou dados pessoais sensíveis (Art. 5º, II da LGPD).*
2. **Cadastro de Empresas Escrituradas (`companies`):**
   - Coleta restrita a dados públicos de Pessoa Jurídica (CNPJ, Razão Social, Nome Fantasia, Município/UF e Regime Tributário) exigidos no Bloco 0 da Escrituração Contábil Digital (SPED ECD).

### 2.2. Bases Legais (Art. 7º da LGPD)
- **Art. 7º, II (Cumprimento de obrigação legal ou regulatória):** Escrituração contábil e conservação de livros mercantis conforme Art. 1.179 do Código Civil e NBC ITG 2000.
- **Art. 7º, V (Execução de contrato):** Operação das rotinas contábeis pelos profissionais habilitados.
- **Art. 7º, IX (Legítimo interesse técnico e segurança):** Controle de acesso baseado em papéis (RBAC) e prevenção à fraude.

### 2.3. Direitos do Titular e Autoatendimento para Exclusão de Dados (Art. 18 da LGPD)
O NORTABIL disponibiliza diretamente na interface (**botão "LGPD & Termos" -> aba "Meus Dados & Exclusão"**) e na API REST dois mecanismos imediatos:
- **Confirmação e Portabilidade (`GET /api/v1/auth/lgpd/export`):** Permite ao usuário autenticado baixar instantaneamente um arquivo `.json` com todos os seus dados pessoais armazenados.
- **Eliminação / Anonimização de Dados Pessoais (`DELETE /api/v1/auth/lgpd/delete-account`):** Permite ao titular solicitar a exclusão da sua conta. O sistema anonimiza imediatamente o nome, o e-mail e o CRC no banco de dados e desativa o acesso, preservando apenas a integridade matemática dos lançamentos contábeis históricos exigida pelo Art. 1.179 do Código Civil e Art. 16, I da LGPD.

---

## 3. Consentimento de Cookies e Auditoria de Rastreamento / Terceiros

### 3.1. Revisão de Google Analytics e Scripts de Rastreamento
- **Google Analytics (GA4 / Tag Manager):** **Ausente (Não utilizado).**
- **Meta Pixel / TikTok Pixel / Hotjar / Rastreadores Publicitários:** **Ausentes (Não utilizados).**
- **Cookies de Terceiros:** **Nenhum.**

### 3.2. Armazenamento Local Estritamente Necessário e Banner de Consentimento
O sistema exibe na entrada o **Banner de Transparência de Cookies e Armazenamento Local (`#cookieConsentBanner`)**, informando de forma clara que utiliza apenas chaves técnicas no `localStorage` do próprio navegador:
- `nortabil_auth_token` e `nortabil_current_user`: manutenção da sessão autenticada JWT;
- `nortabil_company_id`: seleção da empresa ativa (Multi-Tenant);
- `nortabil_cookie_consent_v1`: registro do aceite do aviso de privacidade.

### 3.3. Revisão das Integrações de Terceiros
As únicas conexões externas do cliente web ocorrem com redes de distribuição de conteúdo (CDNs) estritamente visuais e sem envio de dados contábeis:
1. **Tailwind CSS CDN (`cdn.tailwindcss.com`):** Estilização de layout responsivo;
2. **Font Awesome Free 6.4.0 (`cdnjs.cloudflare.com`):** Ícones vetoriais sob licença aberta (*CC BY 4.0* para ícones e *SIL OFL* para fontes);
3. **Google Fonts (`fonts.googleapis.com` - Família Inter):** Tipografia aberta sob *SIL Open Font License 1.1*;
4. **Swagger UI (`unpkg.com/swagger-ui-dist`):** Carregado exclusivamente na rota de documentação técnica `/swagger`.

---

## 4. Termos de Uso, Política de Reembolso e Direitos Autorais

### 4.1. Termos de Uso e Ausência de Promessas Sem Comprovação ou Depoimentos Falsos
- Toda a comunicação do sistema e da documentação técnica foi revisada conforme o **Código de Defesa do Consumidor (Art. 37 da Lei nº 8.078/1990)** e o **Código Brasileiro de Autorregulamentação Publicitária (CONAR)**:
  - **Zero depoimentos falsos ou avaliações fictícias;**
  - **Zero promessas financeiras ou tributárias sem comprovação científica;**
  - Todas as garantias técnicas referem-se exclusivamente a propriedades matemáticas comprováveis (Invariante de Luca Pacioli $\sum \text{Débitos} = \sum \text{Créditos}$, precisão decimal exata de ponto fixo e atomicidade ACID no PostgreSQL 16).

### 4.2. Política de Reembolso e Direito de Arrependimento (CDC Art. 49)
- Embora esta instância opere em caráter **100% Gratuito / Acadêmico (TCC)** sem cobrança de valores, a **Política de Reembolso** integrada ao sistema assegura expressamente que, em qualquer eventual contratação comercial de planos SaaS ou implantação remota, o contratante goza do **Direito de Arrependimento de 7 (sete) dias corridos** com **reembolso integral (100%)** (Art. 49 da Lei nº 8.078/1990 e Decreto nº 7.962/2013), além de cancelamento livre sem multa de fidelidade e portabilidade total dos dados em CSV e SPED ECD.

### 4.3. Direitos Autorais de Imagens e Ativos Visuais (Lei nº 9.610/1998)
- O projeto não utiliza nenhuma imagem ou fotografia protegida por *copyright* de terceiros. Todos os elementos gráficos são construídos em CSS puro e ícones vetoriais abertos (*Font Awesome Free — CC BY 4.0*).

---

## 5. Auditoria de Acessibilidade (WCAG 2.1 Nível AA / eMAG / LBI)

Em cumprimento à **Lei Brasileira de Inclusão (Lei nº 13.146/2015, Art. 63)** e ao **Modelo de Acessibilidade em Governo Eletrônico (eMAG)**:
1. **Navegação por Teclado:**
   - Adicionado link de salto rápido (*"Pular para o conteúdo principal"*) visível ao pressionar `Tab`;
   - Contorno de foco de alto contraste (`:focus-visible` em `#4f46e5`) para todos os botões, links, seletores e campos;
   - Fechamento imediato de qualquer modal aberto ao pressionar a tecla `Esc`.
2. **Textos Alternativos e Semântica ARIA:**
   - Símbolos gráficos e logotipos possuem `role="img"` e `aria-label` descritivo;
   - Ícones decorativos possuem `aria-hidden="true"` para não poluir leitores de tela (NVDA / VoiceOver);
   - Todos os campos de formulário possuem `<label for="...">` explicitamente vinculado;
   - Regiões principais demarcadas com `role="banner"`, `role="navigation"`, `role="main"`, `role="dialog"`, `role="alert"` (`aria-live="polite"`) e `role="contentinfo"`.
3. **Correção de Contraste e Clareza de Botões:**
   - Ajustados textos secundários e rótulos para tons de maior contraste (`text-slate-700`, `text-slate-600`, `text-indigo-100`, `bg-emerald-700`, `bg-rose-700`, `bg-amber-700`), superando a razão mínima de 4.5:1 exigida pela WCAG 2.1 AA;
   - Todos os botões possuem rótulos textuais autoexplicativos e `aria-label` claro.

---

## 6. Quadro de Leis e Normas Brasileiras Verificadas

| Diploma Legal / Norma Técnica | Escopo de Aplicação no NORTABIL |
| :--- | :--- |
| **Lei nº 13.709/2018 (LGPD)** | Minimização de dados, consentimento transparente, portabilidade JSON e exclusão/anonimização de conta (Art. 6º, 7º e 18). |
| **Lei nº 12.965/2014 (Marco Civil da Internet)** | Sigilo de credenciais, transparência de termos e guarda de registros de acesso à aplicação (Art. 7º e 15). |
| **Lei nº 8.078/1990 (CDC) e Decreto nº 7.962/2013** | Identificação clara no rodapé, ausência de publicidade enganosa (Art. 37) e Direito de Arrependimento de 7 dias (Art. 49). |
| **Lei nº 13.146/2015 (Lei Brasileira de Inclusão - LBI)** | Acessibilidade digital (Art. 63), foco por teclado, contraste WCAG AA e suporte a leitores de tela. |
| **Lei nº 9.609/1998 e Lei nº 9.610/1998** | Proteção da propriedade intelectual de programa de computador e direitos autorais (licenciamento MIT e ativos abertos). |
| **Lei nº 10.406/2002 (Código Civil, Art. 1.179 a 1.195)** | Obrigatoriedade da escrituração contábil uniforme em partidas dobradas, Livro Diário e Balanço Patrimonial. |
| **Lei nº 6.404/1976 (Lei das S/A)** | Estrutura legal da DRE, Balanço Patrimonial e Demonstração dos Fluxos de Caixa (DFC). |
| **NBC TG 26, NBC TG 03, NBC TG 16 e NBC ITG 2000 (CFC)** | Formalidades da escrituração contábil digital, estorno por partida inversa, estoques/CMV e responsabilidade técnica CRC. |
