// NORTABIL - Lógica do Cliente Web / Dashboard Contábil

let currentCompanyId = localStorage.getItem('nortabil_company_id') || '';
let authToken = localStorage.getItem('nortabil_auth_token') || '';
let currentUser = JSON.parse(localStorage.getItem('nortabil_current_user') || 'null');
let cachedAccounts = [];

// Interceptador global de fetch para injetar o cabeçalho Authorization: Bearer <token>
const nativeFetch = window.fetch.bind(window);
window.fetch = async (input, init = {}) => {
  const url = typeof input === 'string' ? input : (input?.url || '');
  if (authToken && url.startsWith('/api/v1')) {
    const headers = new Headers(init.headers || {});
    if (!headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${authToken}`);
    }
    init = { ...init, headers };
  }
  return nativeFetch(input, init);
};

document.addEventListener('DOMContentLoaded', async () => {
  // Define data padrão de hoje no formulário de lançamento
  const today = new Date().toISOString().split('T')[0];
  const dateInput = document.getElementById('entryDate');
  if (dateInput) dateInput.value = today;

  const currentYear = new Date().getFullYear();
  const startInput = document.getElementById('statementStartDate');
  const endInput = document.getElementById('statementEndDate');
  if (startInput) startInput.value = `${currentYear}-01-01`;
  if (endInput) endInput.value = `${currentYear}-12-31`;

  const ledgerStart = document.getElementById('ledgerStartDate');
  const ledgerEnd = document.getElementById('ledgerEndDate');
  if (ledgerStart) ledgerStart.value = `${currentYear}-01-01`;
  if (ledgerEnd) ledgerEnd.value = `${currentYear}-12-31`;

  await initAuthSession();
  await loadCompanies();
  if (currentCompanyId) {
    await initCompanyContext();
  }

  // Inicializa 2 linhas padrão no formulário de lançamento
  initJournalFormRows();
});

// ==========================================
// AUTENTICAÇÃO JWT & PERFIS RBAC
// ==========================================
async function initAuthSession() {
  if (authToken) {
    try {
      const res = await fetch('/api/v1/auth/me');
      if (res.ok) {
        const json = await res.json();
        if (json.success && json.data) {
          currentUser = json.data;
          localStorage.setItem('nortabil_current_user', JSON.stringify(currentUser));
          updateAuthUI();
          return;
        }
      }
    } catch (_) {
      // Token expirado ou inválido; realiza login padrão de demonstração abaixo
    }
  }
  // Autentica automaticamente como Contador padrão para pronta demonstração do Dashboard
  await performLogin('contador@nortabil.com.br', 'Nortabil@2026', true);
}

async function performLogin(email, password, silent = false) {
  try {
    const res = await nativeFetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || json.message || 'Credenciais inválidas');
    }

    authToken = json.data.token;
    currentUser = json.data.user;
    localStorage.setItem('nortabil_auth_token', authToken);
    localStorage.setItem('nortabil_current_user', JSON.stringify(currentUser));
    updateAuthUI();

    if (!silent) {
      showAlert(`Autenticado como ${currentUser.name} [${currentUser.role}]`);
      closeModal('modalLogin');
    }
    return true;
  } catch (err) {
    if (!silent) {
      showAlert('Falha na autenticação: ' + err.message, 'error');
    }
    updateAuthUI();
    return false;
  }
}

async function handleLogin(e) {
  e.preventDefault();
  const email = document.getElementById('loginEmail').value.trim();
  const password = document.getElementById('loginPassword').value;
  await performLogin(email, password, false);
}

async function quickLogin(email, password) {
  document.getElementById('loginEmail').value = email;
  document.getElementById('loginPassword').value = password;
  await performLogin(email, password, false);
}

function handleLogout() {
  authToken = '';
  currentUser = null;
  localStorage.removeItem('nortabil_auth_token');
  localStorage.removeItem('nortabil_current_user');
  updateAuthUI();
  closeModal('modalLogin');
  showAlert('Sessão encerrada. Você está no modo visitante (sem permissões de Contador/Admin).', 'error');
}

function updateAuthUI() {
  const label = document.getElementById('authUserLabel');
  const btnUsers = document.getElementById('btnManageUsers');
  const sName = document.getElementById('sessionUserName');
  const sEmail = document.getElementById('sessionUserEmail');
  const sRole = document.getElementById('sessionUserRole');

  if (currentUser && authToken) {
    if (label) label.innerText = `${currentUser.name.split(' ')[0]} (${currentUser.role})`;
    if (sName) sName.innerText = currentUser.name;
    if (sEmail) sEmail.innerText = currentUser.email;
    if (sRole) sRole.innerText = currentUser.role + (currentUser.crc_number ? ` - CRC ${currentUser.crc_number}` : '');

    if (btnUsers) {
      if (currentUser.role === 'ADMIN') {
        btnUsers.classList.remove('hidden');
        btnUsers.classList.add('flex');
      } else {
        btnUsers.classList.add('hidden');
        btnUsers.classList.remove('flex');
      }
    }
  } else {
    if (label) label.innerText = 'Entrar (Login)';
    if (sName) sName.innerText = 'Não autenticado';
    if (sEmail) sEmail.innerText = '-';
    if (sRole) sRole.innerText = 'VISITANTE';
    if (btnUsers) {
      btnUsers.classList.add('hidden');
      btnUsers.classList.remove('flex');
    }
  }
}

async function openUsersModal() {
  openModal('modalUsers');
  await loadUsersList();
}

async function loadUsersList() {
  const tbody = document.getElementById('usersTableBody');
  if (!tbody) return;
  try {
    const res = await fetch('/api/v1/auth/users');
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.message || json.error?.message || 'Acesso negado (requer ADMIN)');
    }
    const users = json.data || [];
    tbody.innerHTML = users.map(u => `
      <tr class="hover:bg-slate-50">
        <td class="px-3 py-2 font-semibold text-slate-800">${u.name}</td>
        <td class="px-3 py-2 text-slate-600 font-mono">${u.email}</td>
        <td class="px-3 py-2"><span class="px-2 py-0.5 rounded text-[10px] font-bold ${u.role === 'ADMIN' ? 'bg-indigo-100 text-indigo-800' : (u.role === 'CONTADOR' ? 'bg-emerald-100 text-emerald-800' : 'bg-amber-100 text-amber-800')}">${u.role}</span></td>
        <td class="px-3 py-2 font-mono text-slate-500">${u.crc_number || '-'}</td>
      </tr>
    `).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="4" class="px-3 py-4 text-center text-rose-600 font-semibold">${err.message}</td></tr>`;
  }
}

async function handleRegisterUser(e) {
  e.preventDefault();
  const payload = {
    name: document.getElementById('newUserName').value.trim(),
    email: document.getElementById('newUserEmail').value.trim(),
    password: document.getElementById('newUserPassword').value,
    role: document.getElementById('newUserRole').value,
    crc_number: document.getElementById('newUserCRC').value.trim()
  };

  try {
    const res = await fetch('/api/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || json.message || 'Falha ao cadastrar usuário');
    }
    showAlert(`Usuário ${payload.name} (${payload.role}) cadastrado com sucesso!`);
    document.getElementById('newUserName').value = '';
    document.getElementById('newUserEmail').value = '';
    document.getElementById('newUserPassword').value = '';
    document.getElementById('newUserCRC').value = '';
    await loadUsersList();
  } catch (err) {
    showAlert('Erro ao cadastrar usuário: ' + err.message, 'error');
  }
}

// ==========================================
// NAVEGAÇÃO E MODAIS
// ==========================================
function switchTab(tabId) {
  document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
  document.querySelectorAll('.nav-btn').forEach(btn => {
    btn.classList.remove('active', 'border-indigo-400', 'text-white');
    btn.classList.add('border-transparent', 'text-indigo-300');
  });

  const selectedTab = document.getElementById(tabId);
  if (selectedTab) selectedTab.classList.remove('hidden');

  const activeBtn = Array.from(document.querySelectorAll('.nav-btn')).find(b => b.getAttribute('onclick')?.includes(tabId));
  if (activeBtn) {
    activeBtn.classList.add('active', 'border-indigo-400', 'text-white');
    activeBtn.classList.remove('border-transparent', 'text-indigo-300');
  }

  // Recarrega dados específicos da aba
  if (tabId === 'tabDashboard') loadDashboard();
  if (tabId === 'tabJournal') loadJournalEntries();
  if (tabId === 'tabLedger') {
    populateLedgerAccountSelect();
    loadLedgerReport();
  }
  if (tabId === 'tabAccounts') renderAccountsTable();
  if (tabId === 'tabOFX') populateOFXSelects();
  if (tabId === 'tabNFe') populateNFeSelects();
  if (tabId === 'tabStatements') loadFinancialStatements();
}

function openModal(modalId) {
  const m = document.getElementById(modalId);
  if (m) {
    m.classList.remove('hidden');
    m.classList.add('flex');
  }
}

function closeModal(modalId) {
  const m = document.getElementById(modalId);
  if (m) {
    m.classList.add('hidden');
    m.classList.remove('flex');
  }
}

function showAlert(message, type = 'success') {
  const box = document.getElementById('alertBox');
  const icon = document.getElementById('alertIcon');
  const text = document.getElementById('alertMessage');

  box.className = 'mb-4 p-4 rounded-lg flex items-center justify-between text-sm shadow-sm transition ';
  if (type === 'success') {
    box.classList.add('bg-emerald-50', 'text-emerald-800', 'border', 'border-emerald-200');
    icon.className = 'fa-solid fa-circle-check text-emerald-600';
  } else {
    box.classList.add('bg-rose-50', 'text-rose-800', 'border', 'border-rose-200');
    icon.className = 'fa-solid fa-triangle-exclamation text-rose-600';
  }

  text.innerText = message;
  box.classList.remove('hidden');

  setTimeout(() => {
    box.classList.add('hidden');
  }, 6000);
}

function hideAlert() {
  document.getElementById('alertBox').classList.add('hidden');
}

// ==========================================
// EMPRESAS (MULTI-TENANT)
// ==========================================
async function loadCompanies() {
  try {
    const res = await fetch('/api/v1/companies');
    const json = await res.json();
    if (!json.success) throw new Error(json.message);

    const select = document.getElementById('companySelector');
    select.innerHTML = '';

    const companies = json.data || [];
    if (companies.length === 0) {
      select.innerHTML = '<option value="">Nenhuma empresa cadastrada</option>';
      return;
    }

    companies.forEach(c => {
      const opt = document.createElement('option');
      opt.value = c.id;
      opt.innerText = `${c.legal_name} (${c.cnpj})`;
      if (c.id === currentCompanyId) opt.selected = true;
      select.appendChild(opt);
    });

    if (!currentCompanyId && companies.length > 0) {
      currentCompanyId = companies[0].id;
      localStorage.setItem('nortabil_company_id', currentCompanyId);
      select.value = currentCompanyId;
    }

    select.onchange = async (e) => {
      currentCompanyId = e.target.value;
      localStorage.setItem('nortabil_company_id', currentCompanyId);
      await initCompanyContext();
    };
  } catch (err) {
    showAlert('Erro ao carregar lista de empresas: ' + err.message, 'error');
  }
}

async function handleCreateCompany(e) {
  e.preventDefault();
  const payload = {
    cnpj: document.getElementById('newCompanyCNPJ').value.trim(),
    legal_name: document.getElementById('newCompanyLegalName').value.trim(),
    trade_name: document.getElementById('newCompanyTradeName').value.trim(),
    city: document.getElementById('newCompanyCity').value.trim(),
    state: document.getElementById('newCompanyState').value.trim(),
    tax_regime: document.getElementById('newCompanyRegime').value,
    clone_default_accounts: document.getElementById('newCompanyCloneChart').checked,
  };

  try {
    const res = await fetch('/api/v1/companies', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    showAlert('Empresa cadastrada e plano de contas criado com sucesso!');
    closeModal('modalNewCompany');
    currentCompanyId = json.data.id;
    localStorage.setItem('nortabil_company_id', currentCompanyId);
    await loadCompanies();
    await initCompanyContext();
  } catch (err) {
    showAlert('Falha ao cadastrar empresa: ' + err.message, 'error');
  }
}

async function initCompanyContext() {
  await loadAccounts();
  await loadDashboard();
  loadJournalEntries();
  populateOFXSelects();
}

// ==========================================
// CONTAS CONTÁBEIS
// ==========================================
async function loadAccounts() {
  if (!currentCompanyId) return;
  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/accounts`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message);
    cachedAccounts = json.data || [];
    renderAccountsTable();
    updateJournalAccountSelects();
  } catch (err) {
    console.error('Erro ao carregar contas:', err);
  }
}

function renderAccountsTable(filtered = null) {
  const list = filtered || cachedAccounts;
  const tbody = document.getElementById('accountsTableBody');
  if (!tbody) return;

  if (list.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5" class="px-4 py-8 text-center text-slate-400">Nenhuma conta cadastrada.</td></tr>';
    return;
  }

  tbody.innerHTML = list.map(a => {
    const isSynthetic = a.is_synthetic;
    const fontStyle = isSynthetic ? 'font-bold text-slate-800' : 'text-slate-600 font-normal';
    const natureBadge = a.nature === 'DEBIT' ? '<span class="badge-debit px-2 py-0.5 rounded text-[10px] font-bold">DÉBITO</span>' : '<span class="badge-credit px-2 py-0.5 rounded text-[10px] font-bold">CRÉDITO</span>';
    const typeLabel = { 'ASSET': 'Ativo', 'LIABILITY': 'Passivo', 'EQUITY': 'Patrimônio Líquido', 'REVENUE': 'Receita', 'EXPENSE': 'Despesa' }[a.type] || a.type;

    return `
      <tr class="hover:bg-slate-50 transition">
        <td class="px-4 py-2.5 ${fontStyle}">${a.code}</td>
        <td class="px-4 py-2.5 ${fontStyle}">${a.name}</td>
        <td class="px-4 py-2.5 text-slate-500">${typeLabel}</td>
        <td class="px-4 py-2.5">${natureBadge}</td>
        <td class="px-4 py-2.5"><span class="px-2 py-0.5 rounded text-[10px] ${isSynthetic ? 'bg-amber-100 text-amber-800' : 'bg-slate-100 text-slate-700'}">${isSynthetic ? 'Sintética' : 'Analítica'}</span></td>
      </tr>
    `;
  }).join('');
}

function filterAccounts() {
  const query = document.getElementById('accountSearch').value.toLowerCase();
  const filtered = cachedAccounts.filter(a => a.code.toLowerCase().includes(query) || a.name.toLowerCase().includes(query));
  renderAccountsTable(filtered);
}

async function handleCreateAccount(e) {
  e.preventDefault();
  if (!currentCompanyId) return;

  const payload = {
    code: document.getElementById('newAccountCode').value.trim(),
    name: document.getElementById('newAccountName').value.trim(),
    type: document.getElementById('newAccountType').value,
    is_synthetic: document.getElementById('newAccountIsSynthetic').checked
  };

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/accounts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    showAlert('Conta criada com sucesso!');
    closeModal('modalNewAccount');
    await loadAccounts();
  } catch (err) {
    showAlert('Falha ao criar conta: ' + err.message, 'error');
  }
}

// ==========================================
// DASHBOARD & BALANCETE
// ==========================================
async function loadDashboard() {
  if (!currentCompanyId) return;
  try {
    const currentYear = new Date().getFullYear();
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/reports/trial-balance?start_date=${currentYear}-01-01&end_date=${currentYear}-12-31`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message);

    const data = json.data || {};
    const items = data.items || [];

    let asset = 0, liability = 0, equity = 0, revenue = 0, expense = 0;

    items.forEach(t => {
      const bal = parseFloat(t.final_balance) || 0;
      if (t.account_code === '1') asset = bal;
      if (t.account_code === '2') liability = bal;
      if (t.account_code === '3') equity = bal;
      if (t.account_code === '4') revenue = bal;
      if (t.account_code === '5') expense = bal;
    });

    const netResult = revenue - expense;

    document.getElementById('dashTotalAsset').innerText = formatBRL(asset);
    document.getElementById('dashTotalLiability').innerText = formatBRL(liability);
    document.getElementById('dashTotalEquity').innerText = formatBRL(equity);
    document.getElementById('dashTotalRevenue').innerText = formatBRL(revenue);
    document.getElementById('dashTotalExpense').innerText = formatBRL(expense);
    document.getElementById('dashNetResult').innerText = formatBRL(netResult);

    const statusBadge = document.getElementById('dashResultStatus');
    if (netResult > 0) {
      statusBadge.innerText = 'Lucro Líquido';
      statusBadge.className = 'text-[11px] font-semibold text-emerald-600';
    } else if (netResult < 0) {
      statusBadge.innerText = 'Prejuízo do Exercício';
      statusBadge.className = 'text-[11px] font-semibold text-rose-600';
    } else {
      statusBadge.innerText = 'Equilibrado';
      statusBadge.className = 'text-[11px] font-semibold text-slate-500';
    }

    // Tabela do Balancete (exibe contas com saldo ou sintéticas principais)
    const tbody = document.getElementById('trialBalanceTableBody');
    const visibleItems = items.filter(t => t.is_synthetic || parseFloat(t.final_balance) > 0 || parseFloat(t.period_debit) > 0 || parseFloat(t.period_credit) > 0);
    if (visibleItems.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="px-6 py-6 text-center text-slate-400">Nenhuma movimentação para o período.</td></tr>';
      return;
    }

    tbody.innerHTML = visibleItems.map(t => {
      const rowWeight = t.is_synthetic ? 'font-bold text-slate-900 bg-slate-50/70' : 'text-slate-600';
      return `
        <tr class="hover:bg-slate-50 transition ${rowWeight}">
          <td class="px-6 py-2.5 font-mono">${t.account_code}</td>
          <td class="px-6 py-2.5">${t.account_name}</td>
          <td class="px-6 py-2.5">${t.final_nature === 'DEBIT' ? '<span class="badge-debit px-2 py-0.5 rounded text-[10px] font-bold">DÉBITO</span>' : '<span class="badge-credit px-2 py-0.5 rounded text-[10px] font-bold">CRÉDITO</span>'}</td>
          <td class="px-6 py-2.5 text-right font-mono">${formatBRL(t.period_debit)}</td>
          <td class="px-6 py-2.5 text-right font-mono">${formatBRL(t.period_credit)}</td>
          <td class="px-6 py-2.5 text-right font-mono font-bold text-slate-900">${formatBRL(t.final_balance)}</td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    console.error('Erro ao carregar balancete:', err);
  }
}

// ==========================================
// LANÇAMENTOS CONTÁBEIS (DIÁRIO)
// ==========================================
function initJournalFormRows() {
  const container = document.getElementById('journalLinesContainer');
  if (!container) return;
  container.innerHTML = '';
  addJournalLineRow('DEBIT');
  addJournalLineRow('CREDIT');
  recalculateLiveBalance();
}

function addJournalLineRow(defaultType = 'DEBIT') {
  const container = document.getElementById('journalLinesContainer');
  const rowId = 'line_' + Math.random().toString(36).substr(2, 9);

  const analyticalAccounts = cachedAccounts.filter(a => !a.is_synthetic);

  const row = document.createElement('div');
  row.id = rowId;
  row.className = 'grid grid-cols-12 gap-2 items-center journal-line-row bg-white p-2 rounded border border-slate-200';
  row.innerHTML = `
    <div class="col-span-5">
      <select class="line-account w-full text-xs border border-slate-300 rounded px-2 py-1.5 bg-white focus:outline-none" required>
        <option value="" disabled selected>Selecione a conta analítica...</option>
        ${analyticalAccounts.map(a => `<option value="${a.id}">${a.code} - ${a.name}</option>`).join('')}
      </select>
    </div>
    <div class="col-span-2">
      <select class="line-type w-full text-xs font-bold border border-slate-300 rounded px-2 py-1.5 bg-white focus:outline-none" onchange="recalculateLiveBalance()">
        <option value="DEBIT" ${defaultType === 'DEBIT' ? 'selected' : ''}>DÉBITO (D)</option>
        <option value="CREDIT" ${defaultType === 'CREDIT' ? 'selected' : ''}>CRÉDITO (C)</option>
      </select>
    </div>
    <div class="col-span-4">
      <input type="number" step="0.01" min="0.01" placeholder="0.00" class="line-amount w-full text-xs font-mono border border-slate-300 rounded px-2 py-1.5 focus:outline-none" required oninput="recalculateLiveBalance()" />
    </div>
    <div class="col-span-1 text-center">
      <button type="button" onclick="removeJournalLineRow('${rowId}')" class="text-slate-400 hover:text-rose-600 text-xs transition">
        <i class="fa-solid fa-trash-can"></i>
      </button>
    </div>
  `;

  container.appendChild(row);
  recalculateLiveBalance();
}

function removeJournalLineRow(rowId) {
  const rows = document.querySelectorAll('.journal-line-row');
  if (rows.length <= 2) {
    showAlert('Um lançamento contábil precisa de no mínimo duas partidas dobradas!', 'error');
    return;
  }
  const el = document.getElementById(rowId);
  if (el) el.remove();
  recalculateLiveBalance();
}

function recalculateLiveBalance() {
  let totalD = 0;
  let totalC = 0;

  document.querySelectorAll('.journal-line-row').forEach(row => {
    const type = row.querySelector('.line-type')?.value;
    const amt = parseFloat(row.querySelector('.line-amount')?.value) || 0;
    if (type === 'DEBIT') totalD += amt;
    if (type === 'CREDIT') totalC += amt;
  });

  const diff = Math.abs(totalD - totalC);
  const ind = document.getElementById('liveBalanceIndicator');
  const btn = document.getElementById('btnSubmitJournal');

  if (diff < 0.001 && totalD > 0) {
    ind.className = 'px-3 py-1.5 rounded-lg text-xs font-mono font-bold flex items-center space-x-1.5 bg-emerald-100 text-emerald-800 border border-emerald-300';
    ind.innerHTML = `<i class="fa-solid fa-check"></i><span>Balanceado: D ${formatBRL(totalD)} == C ${formatBRL(totalC)}</span>`;
    if (btn) btn.disabled = false;
  } else {
    ind.className = 'px-3 py-1.5 rounded-lg text-xs font-mono font-bold flex items-center space-x-1.5 bg-rose-100 text-rose-800 border border-rose-300';
    ind.innerHTML = `<i class="fa-solid fa-scale-unbalanced"></i><span>D: ${formatBRL(totalD)} | C: ${formatBRL(totalC)} (Diferença: ${formatBRL(diff)})</span>`;
  }
}

async function handleCreateJournalEntry(e) {
  e.preventDefault();
  if (!currentCompanyId) return;

  const date = document.getElementById('entryDate').value;
  const description = document.getElementById('entryDescription').value.trim();

  const lines = [];
  document.querySelectorAll('.journal-line-row').forEach(row => {
    const account_id = row.querySelector('.line-account').value;
    const type = row.querySelector('.line-type').value;
    const amount = parseFloat(row.querySelector('.line-amount').value);

    if (account_id && amount > 0) {
      lines.push({ account_id, type, amount: amount.toFixed(2), description });
    }
  });

  const payload = {
    date: new Date(date).toISOString(),
    description,
    lines
  };

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/journal-entries`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    showAlert('Lançamento contábil registrado no Livro Diário com sucesso!');
    document.getElementById('entryDescription').value = '';
    initJournalFormRows();
    await loadJournalEntries();
    await loadDashboard();
  } catch (err) {
    showAlert('Falha ao registrar lançamento: ' + err.message, 'error');
  }
}

async function loadJournalEntries() {
  if (!currentCompanyId) return;
  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/journal-entries?limit=30`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message);

    const entries = Array.isArray(json.data) ? json.data : (json.data?.entries || []);
    const list = document.getElementById('journalEntriesList');
    if (!list) return;

    if (entries.length === 0) {
      list.innerHTML = '<div class="p-8 text-center text-slate-400">Nenhum lançamento registrado no Livro Diário.</div>';
      return;
    }

    list.innerHTML = entries.map(e => {
      const isReversed = e.status === 'REVERSED';
      const statusBadge = isReversed
        ? `<span class="text-[10px] font-bold bg-rose-100 text-rose-800 border border-rose-300 px-2 py-0.5 rounded ml-2">ESTORNADO</span>`
        : `<button onclick="handleReverseJournalEntry('${e.id}', ${e.entry_number})" class="text-[10px] font-semibold bg-amber-50 hover:bg-amber-100 text-amber-800 border border-amber-300 px-2 py-0.5 rounded ml-2 transition" title="Gerar Lançamento de Estorno (Partida Inversa - NBC ITG 2000)"><i class="fa-solid fa-rotate-left mr-1"></i>Estornar</button>`;

      return `
      <div class="p-4 hover:bg-slate-50 transition ${isReversed ? 'opacity-75 bg-slate-50/50' : ''}">
        <div class="flex justify-between items-start mb-2">
          <div>
            <span class="font-mono text-xs font-bold text-indigo-700 bg-indigo-50 px-2 py-0.5 rounded border border-indigo-200">#${e.entry_number}</span>
            <span class="text-xs font-semibold text-slate-500 ml-2">${new Date(e.date).toLocaleDateString('pt-BR')}</span>
            ${e.created_by ? `<span class="text-[11px] text-slate-400 ml-2"><i class="fa-solid fa-user-check mr-1"></i>${e.created_by}</span>` : ''}
            ${statusBadge}
            <h4 class="font-semibold text-slate-800 text-sm mt-1 ${isReversed ? 'line-through text-slate-500' : ''}">${e.description}</h4>
          </div>
          <span class="font-mono text-xs font-bold text-slate-900 bg-slate-100 px-3 py-1 rounded">R$ ${parseFloat(e.total_debit).toLocaleString('pt-BR', { minimumFractionDigits: 2 })}</span>
        </div>
        <div class="bg-slate-50 rounded p-2 border border-slate-100 text-xs font-mono space-y-1">
          ${(e.lines || []).map(l => `
            <div class="flex justify-between">
              <span class="text-slate-600">${l.type === 'DEBIT' ? '<b class="text-rose-700">[D]</b>' : '<b class="text-emerald-700">[C]</b>'} ${l.account_code || ''} - ${l.account_name || l.account_id}</span>
              <span class="font-bold text-slate-700">${formatBRL(l.amount)}</span>
            </div>
          `).join('')}
        </div>
      </div>
    `;
    }).join('');
  } catch (err) {
    console.error('Erro ao carregar diário:', err);
  }
}

async function handleReverseJournalEntry(entryId, entryNumber) {
  if (!currentCompanyId) return;
  const reason = prompt(`Confirma o Estorno Contábil (Partida Inversa - NBC ITG 2000) do lançamento #${entryNumber}?\n\nInforme o motivo da retificação:`, 'Retificação de lançamento contábil');
  if (reason === null) return;

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/journal-entries/${entryId}/reverse`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason })
    });
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || json.message || 'Falha ao estornar lançamento');
    }
    showAlert(`Lançamento #${entryNumber} estornado com sucesso! Partida inversa #${json.data.entry_number} registrada no Diário.`);
    await loadJournalEntries();
    await loadDashboard();
  } catch (err) {
    showAlert('Erro ao estornar lançamento: ' + err.message, 'error');
  }
}

function updateJournalAccountSelects() {
  const analyticalAccounts = cachedAccounts.filter(a => !a.is_synthetic);
  document.querySelectorAll('.line-account').forEach(select => {
    const curr = select.value;
    select.innerHTML = '<option value="" disabled selected>Selecione a conta analítica...</option>' +
      analyticalAccounts.map(a => `<option value="${a.id}" ${a.id === curr ? 'selected' : ''}>${a.code} - ${a.name}</option>`).join('');
  });
  populateLedgerAccountSelect();
}

// ==========================================
// LIVRO RAZÃO ANALÍTICO (GENERAL LEDGER)
// ==========================================
function populateLedgerAccountSelect() {
  const select = document.getElementById('ledgerAccountSelect');
  if (!select) return;
  const curr = select.value;
  const analyticalAccounts = cachedAccounts.filter(a => !a.is_synthetic);
  select.innerHTML = '<option value="">Todas as Contas Movimentadas</option>' +
    analyticalAccounts.map(a => `<option value="${a.id}" ${a.id === curr ? 'selected' : ''}>${a.code} - ${a.name}</option>`).join('');
}

async function loadLedgerReport() {
  if (!currentCompanyId) return;
  const container = document.getElementById('ledgerAccountsContainer');
  if (!container) return;

  const currentYear = new Date().getFullYear();
  const startDate = document.getElementById('ledgerStartDate')?.value || `${currentYear}-01-01`;
  const endDate = document.getElementById('ledgerEndDate')?.value || `${currentYear}-12-31`;
  const accountId = document.getElementById('ledgerAccountSelect')?.value || '';

  let url = `/api/v1/companies/${currentCompanyId}/reports/ledger?start_date=${startDate}&end_date=${endDate}`;
  if (accountId) {
    url += `&account_id=${encodeURIComponent(accountId)}`;
  }

  try {
    const res = await fetch(url);
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || json.message || 'Falha ao consultar Livro Razão');
    }

    const report = json.data || {};
    const accounts = (report.accounts || []).filter(acc => {
      if (accountId) return true;
      const hasMovements = Array.isArray(acc.movements) && acc.movements.length > 0;
      const hasInit = parseFloat(acc.initial_balance) > 0;
      const hasFin = parseFloat(acc.final_balance) > 0;
      return hasMovements || hasInit || hasFin;
    });

    if (accounts.length === 0) {
      container.innerHTML = `
        <div class="bg-white p-8 rounded-xl border border-slate-200 text-center text-slate-400 text-sm">
          Nenhuma movimentação encontrada para os filtros selecionados no Livro Razão.
        </div>
      `;
      return;
    }

    container.innerHTML = accounts.map(acc => {
      const movements = acc.movements || [];
      const initNat = acc.initial_nature === 'DEBIT' ? 'D' : 'C';
      const finNat = acc.final_nature === 'DEBIT' ? 'D' : 'C';

      const rowsHtml = movements.length === 0
        ? `<tr><td colspan="6" class="px-4 py-4 text-center text-slate-400">Sem partidas no período selecionado.</td></tr>`
        : movements.map(m => {
            const debVal = parseFloat(m.debit) || 0;
            const credVal = parseFloat(m.credit) || 0;
            const runNat = m.balance_nature === 'DEBIT' ? 'D' : 'C';
            return `
              <tr class="hover:bg-slate-50 transition">
                <td class="px-4 py-2 whitespace-nowrap">${new Date(m.date).toLocaleDateString('pt-BR')}</td>
                <td class="px-4 py-2 font-bold text-indigo-700">#${m.entry_number}</td>
                <td class="px-4 py-2 text-slate-700 font-sans">${m.description}</td>
                <td class="px-4 py-2 text-right ${debVal > 0 ? 'text-rose-700 font-semibold' : 'text-slate-300'}">${debVal > 0 ? formatBRL(debVal) : '-'}</td>
                <td class="px-4 py-2 text-right ${credVal > 0 ? 'text-emerald-700 font-semibold' : 'text-slate-300'}">${credVal > 0 ? formatBRL(credVal) : '-'}</td>
                <td class="px-4 py-2 text-right font-bold text-slate-900">${formatBRL(m.running_balance)} <span class="text-[10px] px-1.5 py-0.5 rounded ${runNat === 'D' ? 'badge-debit' : 'badge-credit'}">${runNat}</span></td>
              </tr>
            `;
          }).join('');

      return `
        <div class="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <div class="px-5 py-3 bg-slate-50 border-b border-slate-200 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2">
            <div>
              <span class="font-mono text-xs font-bold text-indigo-700 bg-indigo-50 px-2 py-0.5 rounded border border-indigo-200">${acc.account_code}</span>
              <span class="font-bold text-slate-800 text-sm ml-2">${acc.account_name}</span>
              <span class="text-[11px] text-slate-400 ml-2">(${acc.account_type})</span>
            </div>
            <div class="flex items-center space-x-4 text-xs font-mono">
              <span class="text-slate-600">Saldo Inicial: <b>${formatBRL(acc.initial_balance)} (${initNat})</b></span>
              <span class="text-slate-900 bg-slate-200/70 px-2.5 py-1 rounded font-bold">Saldo Final: ${formatBRL(acc.final_balance)} (${finNat})</span>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs font-mono text-slate-600">
              <thead class="bg-slate-100/80 text-[11px] uppercase font-semibold text-slate-500 border-b border-slate-200">
                <tr>
                  <th class="px-4 py-2.5">Data</th>
                  <th class="px-4 py-2.5">Lanç. #</th>
                  <th class="px-4 py-2.5">Histórico da Partida</th>
                  <th class="px-4 py-2.5 text-right">Débito</th>
                  <th class="px-4 py-2.5 text-right">Crédito</th>
                  <th class="px-4 py-2.5 text-right">Saldo Acumulado</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                ${rowsHtml}
              </tbody>
              <tfoot class="bg-slate-50 font-bold text-slate-800 border-t border-slate-200">
                <tr>
                  <td colspan="3" class="px-4 py-2 text-right uppercase text-[11px] text-slate-500">Totais da Conta no Período:</td>
                  <td class="px-4 py-2 text-right text-rose-700">${formatBRL(acc.total_debit)}</td>
                  <td class="px-4 py-2 text-right text-emerald-700">${formatBRL(acc.total_credit)}</td>
                  <td class="px-4 py-2 text-right">${formatBRL(acc.final_balance)} (${finNat})</td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      `;
    }).join('');
  } catch (err) {
    container.innerHTML = `<div class="bg-rose-50 p-4 rounded-xl border border-rose-200 text-rose-700 text-xs font-semibold">Erro ao carregar Livro Razão: ${err.message}</div>`;
  }
}

// ==========================================
// CONCILIAÇÃO BANCÁRIA OFX
// ==========================================
function populateOFXSelects() {
  const analyticalAccounts = cachedAccounts.filter(a => !a.is_synthetic);
  const bankSelect = document.getElementById('ofxBankAccount');
  const counterpartSelect = document.getElementById('ofxCounterpartAccount');

  if (!bankSelect || !counterpartSelect) return;

  const banks = analyticalAccounts.filter(a => a.code.startsWith('1.1.01'));
  bankSelect.innerHTML = '<option value="" disabled selected>Selecione a conta de banco...</option>' +
    banks.map(a => `<option value="${a.id}">${a.code} - ${a.name}</option>`).join('');

  counterpartSelect.innerHTML = '<option value="" disabled selected>Selecione a contrapartida padrão...</option>' +
    analyticalAccounts.map(a => `<option value="${a.id}">${a.code} - ${a.name}</option>`).join('');

  bankSelect.onchange = () => loadReconciliationItems();
}

async function handleImportOFX(e) {
  e.preventDefault();
  if (!currentCompanyId) return;

  const bankId = document.getElementById('ofxBankAccount').value;
  const counterpartId = document.getElementById('ofxCounterpartAccount').value;
  const fileInput = document.getElementById('ofxFile');

  if (!fileInput.files || fileInput.files.length === 0) {
    showAlert('Selecione um arquivo .OFX', 'error');
    return;
  }

  const formData = new FormData();
  formData.append('bank_account_id', bankId);
  formData.append('counterpart_account_id', counterpartId);
  formData.append('file', fileInput.files[0]);

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/reconciliation/import-ofx`, {
      method: 'POST',
      body: formData
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    const s = json.data;
    showAlert(`Extrato processado! Lidas: ${s.total_read} | Importadas: ${s.total_imported} | Duplicadas ignoradas: ${s.total_duplicates}`);
    fileInput.value = '';
    await loadReconciliationItems();
    await loadJournalEntries();
    await loadDashboard();
  } catch (err) {
    showAlert('Falha ao importar extrato OFX: ' + err.message, 'error');
  }
}

async function loadReconciliationItems() {
  const bankId = document.getElementById('ofxBankAccount')?.value;
  const tbody = document.getElementById('reconciliationTableBody');
  if (!bankId || !tbody) return;

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/reconciliation/items?bank_account_id=${bankId}`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message);

    const items = json.data || [];
    if (items.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-slate-400">Nenhuma transação importada para esta conta.</td></tr>';
      return;
    }

    tbody.innerHTML = items.map(item => `
      <tr class="hover:bg-slate-50 transition">
        <td class="px-4 py-2.5 font-mono text-indigo-700">${item.fitid}</td>
        <td class="px-4 py-2.5">${new Date(item.date).toLocaleDateString('pt-BR')}</td>
        <td class="px-4 py-2.5 text-slate-700">${item.memo}</td>
        <td class="px-4 py-2.5 text-right font-bold">${formatBRL(item.amount)}</td>
        <td class="px-4 py-2.5">${item.type === 'DEBIT' ? '<span class="badge-debit px-2 py-0.5 rounded text-[10px] font-bold">DÉBITO</span>' : '<span class="badge-credit px-2 py-0.5 rounded text-[10px] font-bold">CRÉDITO</span>'}</td>
        <td class="px-4 py-2.5 text-center"><span class="bg-emerald-100 text-emerald-800 text-[10px] font-bold px-2 py-0.5 rounded">CONCILIADO</span></td>
      </tr>
    `).join('');
  } catch (err) {
    console.error('Erro ao listar reconciliação:', err);
  }
}

// ==========================================
// IMPORTADOR AUTOMÁTICO DE NF-e (XML SEFAZ)
// ==========================================
function populateNFeSelects() {
  const analyticalAccounts = cachedAccounts.filter(a => !a.is_synthetic);
  const debSel = document.getElementById('nfeDebitAccount');
  const credSel = document.getElementById('nfeCreditAccount');
  if (!debSel || !credSel) return;

  const options = '<option value="">Classificação Contábil Automática (Padrão)</option>' +
    analyticalAccounts.map(a => `<option value="${a.id}">${a.code} - ${a.name}</option>`).join('');
  debSel.innerHTML = options;
  credSel.innerHTML = options;
}

function fillSampleNFeXML(type) {
  const textarea = document.getElementById('nfeXmlContent');
  if (!textarea) return;

  const today = new Date().toISOString().split('T')[0];

  if (type === 'ENTRADA') {
    textarea.value = `<?xml version="1.0" encoding="UTF-8"?>
<nfeProc versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">
  <NFe>
    <infNFe Id="NFe31260211222333000181550010000894121000894125" versao="4.00">
      <ide>
        <cUF>31</cUF>
        <natOp>Compra de Insumos Agricolas e Embalagens</natOp>
        <mod>55</mod>
        <serie>1</serie>
        <nNF>89412</nNF>
        <dhEmi>${today}T09:30:00-03:00</dhEmi>
        <tpNF>0</tpNF>
      </ide>
      <emit>
        <CNPJ>11222333000181</CNPJ>
        <xNome>AGROINSUMOS NORTE DE MINAS LTDA - JANUARIA MG</xNome>
      </emit>
      <dest>
        <CNPJ>19131243000197</CNPJ>
        <xNome>FRUTICULTURA VALE DO JAIBA LTDA</xNome>
      </dest>
      <det nItem="1">
        <prod>
          <cProd>INS-001</cProd>
          <xProd>Fertilizante NPK Organomineral e Caixas Plasticas para Colheita</xProd>
          <CFOP>1101</CFOP>
          <vProd>8450.00</vProd>
        </prod>
      </det>
      <total>
        <ICMSTot>
          <vProd>8450.00</vProd>
          <vICMS>1014.00</vICMS>
          <vDesc>0.00</vDesc>
          <vNF>8450.00</vNF>
        </ICMSTot>
      </total>
    </infNFe>
  </NFe>
  <protNFe versao="4.00">
    <infProt>
      <chNFe>31260211222333000181550010000894121000894125</chNFe>
    </infProt>
  </protNFe>
</nfeProc>`;
  } else {
    textarea.value = `<?xml version="1.0" encoding="UTF-8"?>
<nfeProc versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">
  <NFe>
    <infNFe Id="NFe31260219131243000197550010001254801001254809" versao="4.00">
      <ide>
        <cUF>31</cUF>
        <natOp>Venda de Producao do Estabelecimento</natOp>
        <mod>55</mod>
        <serie>1</serie>
        <nNF>125480</nNF>
        <dhEmi>${today}T14:15:00-03:00</dhEmi>
        <tpNF>1</tpNF>
      </ide>
      <emit>
        <CNPJ>19131243000197</CNPJ>
        <xNome>FRUTICULTURA VALE DO JAIBA LTDA</xNome>
      </emit>
      <dest>
        <CNPJ>60701190000104</CNPJ>
        <xNome>DISTRIBUIDORA COMERCIAL MONTES CLAROS LTDA</xNome>
      </dest>
      <det nItem="1">
        <prod>
          <cProd>FRU-101</cProd>
          <xProd>Banana Prata Selecionada Exportacao e Manga Palmer (Lote Irrigado Jaiba)</xProd>
          <CFOP>5101</CFOP>
          <vProd>22500.00</vProd>
        </prod>
      </det>
      <total>
        <ICMSTot>
          <vProd>22500.00</vProd>
          <vICMS>2700.00</vICMS>
          <vDesc>0.00</vDesc>
          <vNF>22500.00</vNF>
        </ICMSTot>
      </total>
    </infNFe>
  </NFe>
  <protNFe versao="4.00">
    <infProt>
      <chNFe>31260219131243000197550010001254801001254809</chNFe>
    </infProt>
  </protNFe>
</nfeProc>`;
  }
}

async function handleImportNFeXML(e) {
  e.preventDefault();
  if (!currentCompanyId) return;

  const xmlContent = document.getElementById('nfeXmlContent').value.trim();
  const debitAccountId = document.getElementById('nfeDebitAccount')?.value || '';
  const creditAccountId = document.getElementById('nfeCreditAccount')?.value || '';

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/nfe/import-xml`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        xml_content: xmlContent,
        debit_account_id: debitAccountId,
        credit_account_id: creditAccountId
      })
    });
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || json.message || 'Falha ao processar XML da NF-e');
    }

    const r = json.data;
    const doc = r.document;
    const je = r.journal_entry;
    const box = document.getElementById('nfeResultBox');
    if (box) {
      box.classList.remove('hidden');
      box.innerHTML = `
        <div class="flex justify-between items-center border-b border-emerald-200 pb-2">
          <span class="font-bold text-emerald-900 text-sm"><i class="fa-solid fa-circle-check mr-1"></i>NF-e nº ${doc.number} (Série ${doc.series}) Contabilizada com Sucesso!</span>
          <span class="font-mono font-bold px-2.5 py-0.5 rounded bg-emerald-200 text-emerald-900">${r.direction === 'ENTRADA_COMPRA' ? 'ENTRADA / COMPRA' : 'SAÍDA / FATURAMENTO'}</span>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-2 font-mono text-slate-700 pt-1">
          <div><b>Chave SEFAZ:</b> ${doc.access_key || '-'}</div>
          <div><b>Natureza:</b> ${doc.operation_nature}</div>
          <div><b>Emitente:</b> ${doc.emitter_name} (${doc.emitter_cnpj})</div>
          <div><b>Destinatário:</b> ${doc.recipient_name} (${doc.recipient_cnpj})</div>
          <div><b>Valor Total (vNF):</b> <span class="font-bold text-emerald-800">${formatBRL(doc.total_invoice)}</span></div>
          <div><b>Lançamento Diário:</b> <span class="font-bold text-indigo-800">#${je.entry_number} (${je.created_by || 'Sistema'})</span></div>
        </div>
      `;
    }

    showAlert(`NF-e nº ${doc.number} importada! Lançamento #${je.entry_number} (${formatBRL(doc.total_invoice)}) registrado no Livro Diário.`);
    await loadJournalEntries();
    await loadDashboard();
  } catch (err) {
    showAlert('Erro ao importar XML de NF-e: ' + err.message, 'error');
  }
}

// ==========================================
// PERÍODOS E ARE
// ==========================================
async function handleClosePeriod(e) {
  e.preventDefault();
  if (!currentCompanyId) return;

  const year = parseInt(document.getElementById('periodYear').value);
  const month = parseInt(document.getElementById('periodMonth').value);

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/periods/close`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ year, month })
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    showAlert(`Período ${month}/${year} fechado com sucesso! Lançamentos retroativos bloqueados.`);
  } catch (err) {
    showAlert('Falha ao fechar período: ' + err.message, 'error');
  }
}

async function handleReopenPeriod() {
  if (!currentCompanyId) return;
  const year = parseInt(document.getElementById('periodYear').value);
  const month = parseInt(document.getElementById('periodMonth').value);

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/periods/reopen`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ year, month })
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    showAlert(`Período ${month}/${year} reaberto com sucesso.`);
  } catch (err) {
    showAlert('Falha ao reabrir período: ' + err.message, 'error');
  }
}

async function handleClosingYear() {
  if (!currentCompanyId) return;
  const year = parseInt(document.getElementById('areYear').value);

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/periods/closing-year`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ year })
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || json.error);

    const r = json.data;
    const box = document.getElementById('areResultBox');
    box.classList.remove('hidden');
    const isProfit = r.is_profit || parseFloat(r.net_result) >= 0;
    box.innerHTML = `
      <div class="font-bold text-slate-800 text-sm mb-1">Apuração do Exercício ${r.year} Concluída</div>
      <div>Total de Receitas Encerradas : ${formatBRL(r.total_revenues)}</div>
      <div>Total de Despesas Encerradas: ${formatBRL(r.total_expenses)}</div>
      <div class="font-bold ${isProfit ? 'text-emerald-700' : 'text-rose-700'}">Resultado Líquido Apurado    : ${formatBRL(r.net_result)} (${isProfit ? 'LUCRO' : 'PREJUÍZO'})</div>
    `;

    showAlert('ARE Anual processada e transferida para o Patrimônio Líquido com sucesso!');
    await loadJournalEntries();
    await loadDashboard();
  } catch (err) {
    showAlert('Falha na apuração do resultado: ' + err.message, 'error');
  }
}

// ==========================================
// EXPORTADOR SPED ECD
// ==========================================
function handleExportSPED(e) {
  e.preventDefault();
  if (!currentCompanyId) return;

  const year = document.getElementById('spedYear').value;
  const crc = encodeURIComponent(document.getElementById('spedCRC').value);
  const auditorName = encodeURIComponent(document.getElementById('spedAuditorName').value);
  const auditorCPF = encodeURIComponent(document.getElementById('spedAuditorCPF').value);
  const signatoryName = encodeURIComponent(document.getElementById('spedSignatoryName').value);
  const signatoryCPF = encodeURIComponent(document.getElementById('spedSignatoryCPF').value);

  const tokenParam = authToken ? `&token=${encodeURIComponent(authToken)}` : '';
  const url = `/api/v1/companies/${currentCompanyId}/export/sped-ecd?year=${year}&auditor_crc=${crc}&auditor_name=${auditorName}&auditor_cpf=${auditorCPF}&signatory_name=${signatoryName}&signatory_cpf=${signatoryCPF}${tokenParam}`;

  window.open(url, '_blank');
}

async function handleValidateSPED() {
  if (!currentCompanyId) return;

  const year = document.getElementById('spedYear').value;
  const crc = encodeURIComponent(document.getElementById('spedCRC').value);
  const auditorName = encodeURIComponent(document.getElementById('spedAuditorName').value);
  const auditorCPF = encodeURIComponent(document.getElementById('spedAuditorCPF').value);
  const signatoryName = encodeURIComponent(document.getElementById('spedSignatoryName').value);
  const signatoryCPF = encodeURIComponent(document.getElementById('spedSignatoryCPF').value);

  try {
    const res = await fetch(`/api/v1/companies/${currentCompanyId}/export/sped-ecd/validate?year=${year}&auditor_crc=${crc}&auditor_name=${auditorName}&auditor_cpf=${auditorCPF}&signatory_name=${signatoryName}&signatory_cpf=${signatoryCPF}`);
    const json = await res.json();
    if (!res.ok || !json.success) {
      throw new Error(json.error?.message || json.message || 'Falha ao validar SPED ECD');
    }

    const v = json.data;
    const box = document.getElementById('spedValidationBox');
    if (box) {
      box.classList.remove('hidden');
      box.innerHTML = `
        <div class="flex justify-between items-center border-b border-emerald-200 pb-2">
          <span class="font-bold text-emerald-900 text-sm"><i class="fa-solid fa-circle-check mr-1"></i>Laudo de Auditoria do Pré-Validador PVA (SPED ECD Leiaute 9)</span>
          <span class="font-mono font-bold px-2.5 py-0.5 rounded ${v.is_valid ? 'bg-emerald-200 text-emerald-900' : 'bg-rose-200 text-rose-900'}">${v.is_valid ? '100% APROVADO SEM ERROS' : 'INCONSISTÊNCIAS DETECTADAS'}</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 font-mono text-slate-700 pt-1">
          <div><b>Linhas Físicas:</b> ${v.total_lines}</div>
          <div><b>Registro 9999:</b> ${v.reported_lines_9999}</div>
          <div><b>Contas (I050):</b> ${v.total_accounts_i050}</div>
          <div><b>Lançamentos (I200):</b> ${v.total_entries_i200} (${v.total_partidas_i250} partidas I250)</div>
          <div class="col-span-2"><b>Soma Débitos I250:</b> <span class="text-rose-700 font-bold">${formatBRL(v.total_debit_i250)}</span></div>
          <div class="col-span-2"><b>Soma Créditos I250:</b> <span class="text-emerald-700 font-bold">${formatBRL(v.total_credit_i250)}</span></div>
        </div>
        <div class="pt-1 text-[11px] text-slate-600">
          <b>Blocos Auditados:</b> ${(v.blocks_verified || []).join(' • ')}
        </div>
      `;
    }
    showAlert('Auditoria do SPED ECD concluída: arquivo 100% íntegro e balanceado para o PVA da Receita Federal!');
  } catch (err) {
    showAlert('Erro na validação do SPED ECD: ' + err.message, 'error');
  }
}

// ==========================================
// DEMONSTRAÇÕES CONTÁBEIS (DRE & BALANÇO)
// ==========================================
async function loadFinancialStatements() {
  if (!currentCompanyId) return;

  const startDate = document.getElementById('statementStartDate')?.value || `${new Date().getFullYear()}-01-01`;
  const endDate = document.getElementById('statementEndDate')?.value || `${new Date().getFullYear()}-12-31`;

  // 1. Carrega DRE
  try {
    const resDRE = await fetch(`/api/v1/companies/${currentCompanyId}/reports/dre?start_date=${startDate}&end_date=${endDate}`);
    const jsonDRE = await resDRE.json();
    if (jsonDRE.success) {
      renderDRE(jsonDRE.data);
    }
  } catch (err) {
    console.error('Erro ao carregar DRE:', err);
  }

  // 2. Carrega Balanço Patrimonial
  try {
    const resBS = await fetch(`/api/v1/companies/${currentCompanyId}/reports/balance-sheet?date=${endDate}`);
    const jsonBS = await resBS.json();
    if (jsonBS.success) {
      renderBalanceSheet(jsonBS.data);
    }
  } catch (err) {
    console.error('Erro ao carregar Balanço Patrimonial:', err);
  }

  // 3. Carrega Indicadores Econômico-Financeiros
  try {
    const resInd = await fetch(`/api/v1/companies/${currentCompanyId}/reports/indicators?start_date=${startDate}&end_date=${endDate}`);
    const jsonInd = await resInd.json();
    if (jsonInd.success && jsonInd.data) {
      renderFinancialIndicators(jsonInd.data);
    }
  } catch (err) {
    console.error('Erro ao carregar Indicadores Financeiros:', err);
  }

  // 4. Carrega Demonstração dos Fluxos de Caixa (DFC - NBC TG 03)
  try {
    const resDFC = await fetch(`/api/v1/companies/${currentCompanyId}/reports/dfc?start_date=${startDate}&end_date=${endDate}`);
    const jsonDFC = await resDFC.json();
    if (jsonDFC.success && jsonDFC.data) {
      renderDFC(jsonDFC.data);
    }
  } catch (err) {
    console.error('Erro ao carregar DFC:', err);
  }
}

function renderDFC(dfc) {
  const tbody = document.getElementById('dfcTableBody');
  const badge = document.getElementById('dfcReconciledBadge');
  if (!tbody) return;

  if (badge) {
    if (dfc.is_reconciled) {
      badge.className = 'text-[11px] font-bold px-2 py-0.5 rounded bg-emerald-100 text-emerald-800';
      badge.innerText = `CAIXA FINAL: ${formatBRL(dfc.final_cash_balance)} (CONCILIADO)`;
    } else {
      badge.className = 'text-[11px] font-bold px-2 py-0.5 rounded bg-rose-100 text-rose-800';
      badge.innerText = 'DIVERGENTE';
    }
  }

  tbody.innerHTML = (dfc.lines || []).map(line => {
    const amt = parseFloat(line.amount) || 0;
    const isTotal = line.is_total;
    const rowClass = isTotal ? 'font-bold bg-slate-50 text-slate-900 border-t border-b border-slate-200' : 'text-slate-600 hover:bg-slate-50';
    const amountColor = isTotal
      ? (amt >= 0 ? 'text-indigo-800 font-bold' : 'text-rose-700 font-bold')
      : (amt >= 0 ? 'text-emerald-700' : 'text-rose-700');

    return `
      <tr class="${rowClass} transition">
        <td class="py-2 px-3">${line.description}</td>
        <td class="py-2 px-3 text-right ${amountColor}">${formatBRL(amt)}</td>
      </tr>
    `;
  }).join('');
}

function renderFinancialIndicators(ind) {
  const fmtNum = (v) => (parseFloat(v) || 0).toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  const elCurr = document.getElementById('indCurrentLiquidity');
  const elImm = document.getElementById('indImmediateLiquidity');
  const elDebt = document.getElementById('indIndebtedness');
  const elGross = document.getElementById('indGrossMargin');
  const elNet = document.getElementById('indNetMargin');
  const elRoeRoa = document.getElementById('indRoeRoa');

  if (elCurr) elCurr.innerText = `${fmtNum(ind.current_liquidity)}x`;
  if (elImm) elImm.innerText = `${fmtNum(ind.immediate_liquidity)}x`;
  if (elDebt) elDebt.innerText = `${fmtNum(ind.general_indebtedness_pct)}%`;
  if (elGross) elGross.innerText = `${fmtNum(ind.gross_margin_pct)}%`;
  if (elNet) elNet.innerText = `${fmtNum(ind.net_margin_pct)}%`;
  if (elRoeRoa) elRoeRoa.innerText = `${fmtNum(ind.roe_pct)}% / ${fmtNum(ind.roa_pct)}%`;
}

function renderDRE(dre) {
  const tbody = document.getElementById('dreTableBody');
  const badge = document.getElementById('dreNetBadge');
  if (!tbody) return;

  const net = parseFloat(dre.net_income) || 0;
  if (badge) {
    badge.innerText = formatBRL(net);
    badge.className = `text-xs font-mono font-bold px-2 py-0.5 rounded ${net >= 0 ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'}`;
  }

  tbody.innerHTML = (dre.lines || []).map(line => {
    const amt = parseFloat(line.amount) || 0;
    const isTotal = line.is_total;
    const rowClass = isTotal ? 'font-bold bg-slate-50 text-slate-900 border-t border-b border-slate-200' : 'text-slate-600 hover:bg-slate-50';
    const amountColor = isTotal
      ? (amt >= 0 ? 'text-emerald-700 font-bold' : 'text-rose-700 font-bold')
      : 'text-slate-700';

    return `
      <tr class="${rowClass} transition">
        <td class="py-2 px-3">${line.description}</td>
        <td class="py-2 px-3 text-right ${amountColor}">${formatBRL(amt)}</td>
      </tr>
    `;
  }).join('');
}

function renderBalanceSheet(bs) {
  document.getElementById('bsTotalAsset').innerText = formatBRL(bs.total_asset);
  document.getElementById('bsCurrentAsset').innerText = formatBRL(bs.current_asset);
  document.getElementById('bsNonCurrentAsset').innerText = formatBRL(bs.non_current_asset);

  document.getElementById('bsTotalLiabilityEquity').innerText = formatBRL(bs.total_liability_and_equity);
  document.getElementById('bsCurrentLiability').innerText = formatBRL(bs.current_liability);
  document.getElementById('bsNonCurrentLiability').innerText = formatBRL(bs.non_current_liability);
  document.getElementById('bsTotalEquity').innerText = formatBRL(bs.total_equity);

  const badge = document.getElementById('bsBalanceBadge');
  if (badge) {
    if (bs.is_balanced) {
      badge.className = 'text-[11px] font-bold px-2 py-0.5 rounded bg-emerald-100 text-emerald-800';
      badge.innerText = 'EQUILIBRADO';
    } else {
      badge.className = 'text-[11px] font-bold px-2 py-0.5 rounded bg-rose-100 text-rose-800';
      badge.innerText = 'DESEQUILIBRADO';
    }
  }

  // Linhas analíticas do Ativo
  const assetContainer = document.getElementById('bsAssetLines');
  if (assetContainer) {
    assetContainer.innerHTML = (bs.asset_lines || []).filter(l => !l.is_synthetic).map(l => `
      <div class="flex justify-between py-0.5">
        <span>${l.account_code} ${l.account_name}</span>
        <span class="font-mono">${formatBRL(l.amount)}</span>
      </div>
    `).join('') || '<div class="text-slate-400">Nenhuma conta com saldo.</div>';
  }

  // Linhas analíticas do Passivo e PL
  const liabilityContainer = document.getElementById('bsLiabilityLines');
  if (liabilityContainer) {
    liabilityContainer.innerHTML = (bs.liability_lines || []).filter(l => !l.is_synthetic).map(l => `
      <div class="flex justify-between py-0.5">
        <span>${l.account_code} ${l.account_name}</span>
        <span class="font-mono">${formatBRL(l.amount)}</span>
      </div>
    `).join('') || '<div class="text-slate-400">Nenhuma conta com saldo.</div>';
  }
}

// ==========================================
// EXPORTAÇÃO E IMPRESSÃO DE RELATÓRIOS
// ==========================================
function exportTrialBalanceCSV() {
  if (!currentCompanyId) return;
  const currentYear = new Date().getFullYear();
  const tokenParam = authToken ? `&token=${encodeURIComponent(authToken)}` : '';
  const url = `/api/v1/companies/${currentCompanyId}/reports/export/csv?type=trial-balance&start_date=${currentYear}-01-01&end_date=${currentYear}-12-31${tokenParam}`;
  window.open(url, '_blank');
}

function exportDRECSV() {
  if (!currentCompanyId) return;
  const startDate = document.getElementById('statementStartDate')?.value || `${new Date().getFullYear()}-01-01`;
  const endDate = document.getElementById('statementEndDate')?.value || `${new Date().getFullYear()}-12-31`;
  const tokenParam = authToken ? `&token=${encodeURIComponent(authToken)}` : '';
  const url = `/api/v1/companies/${currentCompanyId}/reports/export/csv?type=dre&start_date=${startDate}&end_date=${endDate}${tokenParam}`;
  window.open(url, '_blank');
}

function printReport(type) {
  if (!currentCompanyId) return;
  let startDate = `${new Date().getFullYear()}-01-01`;
  let endDate = `${new Date().getFullYear()}-12-31`;

  if (type === 'ledger') {
    const lStart = document.getElementById('ledgerStartDate');
    const lEnd = document.getElementById('ledgerEndDate');
    if (lStart && lStart.value) startDate = lStart.value;
    if (lEnd && lEnd.value) endDate = lEnd.value;
  } else {
    const sInput = document.getElementById('statementStartDate');
    const eInput = document.getElementById('statementEndDate');
    if (sInput && sInput.value) startDate = sInput.value;
    if (eInput && eInput.value) endDate = eInput.value;
  }

  const tokenParam = authToken ? `&token=${encodeURIComponent(authToken)}` : '';
  const url = `/api/v1/companies/${currentCompanyId}/reports/export/print?type=${type}&start_date=${startDate}&end_date=${endDate}${tokenParam}`;
  window.open(url, '_blank');
}

// ==========================================
// UTILITÁRIOS
// ==========================================
function formatBRL(val) {
  const n = parseFloat(val) || 0;
  return n.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

