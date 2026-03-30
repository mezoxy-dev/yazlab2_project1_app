
const API = 'http://localhost:8080';

const state = {
  token:           localStorage.getItem('token') || '',
  currentUser:     '',
  currentRole:     '',
  tokenExp:        0,
  allLogs:         [],
  trafficFilter:   '',
  refreshInterval: null,
  timerInterval:   null,
  durChart:        null,
  statusChart:     null,
  methodChart:     null,
  routeChart:      null,
};

/* ── Boot ── */
window.addEventListener('DOMContentLoaded', () => {
  buildEndpointList();

  const tok = state.token;
  if (tok && tok !== 'null' && tok !== 'undefined') {
    if (_isTokenExpired(tok)) {
      _clearSession();
    } else {
      _applyTokenPayload(tok);
      enterApp();
    }
  }

  document.addEventListener('keydown', e => {
    if (e.key !== 'Enter') return;
    const active = document.querySelector('.tab-btn.active');
    if (!active) return;
    const t = active.dataset.tab;
    if (t === 'login')               doLogin();
    else if (t === 'register')       doRegister();
    else if (t === 'admin-register') doAdminRegister();
  });
});

function setLoginTab(tab) {
  ['login', 'register', 'admin-register'].forEach(t => {
    const el = _el('tab-' + t);
    if (el) el.style.display = (t === tab) ? 'block' : 'none';
  });
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.tab === tab);
  });
  ['login-error', 'register-error', 'admin-register-error'].forEach(id => {
    const el = _el(id);
    if (el) { el.textContent = ''; el.style.color = 'var(--danger)'; }
  });
}

async function doLogin() {
  const username = _el('l-username')?.value.trim();
  const password = _el('l-password')?.value;
  if (!username || !password) { _authErr('login', 'Tüm alanlar zorunlu'); return; }
  await _authRequest('/login', { username, password }, 'login');
}

async function doRegister() {
  const username = _el('r-username')?.value.trim();
  const password = _el('r-password')?.value;
  if (!username || !password) { _authErr('register', 'Tüm alanlar zorunlu'); return; }
  await _authRequest('/register', { username, password }, 'register');
}

async function doAdminRegister() {
  const username    = _el('ar-username')?.value.trim();
  const password    = _el('ar-password')?.value;
  const adminSecret = _el('ar-secret')?.value;
  if (!username || !password || !adminSecret) { _authErr('admin-register', 'Tüm alanlar zorunlu'); return; }
  await _authRequest('/register', { username, password, admin_secret: adminSecret }, 'admin-register');
}

async function _authRequest(path, body, mode) {
  const btnLabels = { login: 'Giriş Yap →', register: 'Kayıt Ol →', 'admin-register': 'Admin Kayıt →' };
  const btn = document.querySelector(`#tab-${mode} .btn-primary`);
  if (btn) { btn.disabled = true; btn.textContent = '…'; }

  try {
    const res = await fetch(API + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    let data;
    const ct = res.headers.get('content-type') || '';
    if (ct.includes('json')) { try { data = await res.json(); } catch { data = {}; } }
    else { data = await res.text(); }

    if (!res.ok) {
      const msg = (typeof data === 'string') ? data : (data?.error || 'İşlem başarısız');
      _authErr(mode, msg);
      return;
    }

    if (mode === 'login') {
      localStorage.setItem('token', data.token);
      state.token = data.token;
      _applyTokenPayload(data.token);
      enterApp();
    } else {
      showToast('Kayıt başarılı! Giriş yapabilirsiniz.', 'success');
      setLoginTab('login');
    }
  } catch {
    _authErr(mode, 'Sunucuya ulaşılamadı. Dispatcher çalışıyor mu?');
  } finally {
    if (btn) { btn.disabled = false; btn.textContent = btnLabels[mode]; }
  }
}

function _authErr(mode, msg) {
  const map = { login: 'login-error', register: 'register-error', 'admin-register': 'admin-register-error' };
  const el  = _el(map[mode]);
  if (el) el.textContent = msg;
}

function _applyTokenPayload(tok) {
  try {
    const payload     = JSON.parse(atob(tok.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
    state.currentUser = payload.sub  || 'kullanıcı';
    state.currentRole = payload.role || 'user';
    state.tokenExp    = (payload.exp || 0) * 1000;
  } catch {
    state.currentUser = 'kullanıcı';
    state.currentRole = 'user';
    state.tokenExp    = Date.now() + 86_400_000;
  }
}

function _isTokenExpired(tok) {
  try {
    const p = JSON.parse(atob(tok.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
    return p.exp * 1000 < Date.now();
  } catch { return true; }
}

function enterApp() {
  _el('login-screen')?.classList.add('hidden');
  _el('topbar')?.classList.remove('hidden');
  _el('app')?.classList.remove('hidden');

  _setText('user-name-display', state.currentUser);
  const rb = _el('role-badge');
  if (rb) { rb.textContent = state.currentRole; rb.className = 'role-badge ' + state.currentRole; }

  const apiTok = _el('api-token');
  if (apiTok && state.token) apiTok.value = 'Bearer ' + state.token;

  initCharts();
  fetchAndRender();

  if (state.refreshInterval) clearInterval(state.refreshInterval);
  state.refreshInterval = setInterval(fetchAndRender, 5000);

  startTokenTimer();

  if (state.currentRole !== 'admin') {
    showToast('Admin yetkisi gerekli — istatistikler görüntülenemiyor', 'error');
  }
}

function logout() {
  _clearSession();
  location.reload();
}

function _clearSession() {
  localStorage.removeItem('token');
  state.token = '';
  state.allLogs = [];
  if (state.refreshInterval) clearInterval(state.refreshInterval);
  if (state.timerInterval)   clearInterval(state.timerInterval);
}

async function fetchAndRender() {
  if (!state.token || state.currentRole !== 'admin') return;

  try {
    const res = await fetch(API + '/admin/stats', {
      headers: { Authorization: 'Bearer ' + state.token },
    });

    if (res.status === 401 || res.status === 403) { logout(); return; }
    if (!res.ok) { showToast('Veri alınamadı: ' + res.status, 'error'); return; }

    const data    = await res.json();
    state.allLogs = Array.isArray(data) ? data : [];

    renderStats();
    renderTraffic();
    renderLogs();
    updateCharts();
    renderRouteMap();
    renderServiceCards();
  } catch {
    showToast("Dispatcher'a bağlanılamadı", 'error');
  }
}

function renderStats() {
  const logs   = state.allLogs;
  const total  = logs.length;
  const ok     = logs.filter(l => l.status >= 200 && l.status < 400).length;
  const errors = logs.filter(l => l.status >= 400).length;
  const avg    = total > 0 ? Math.round(logs.reduce((s, l) => s + l.duration_ms, 0) / total) : 0;
  const p95    = _percentile(logs.map(l => l.duration_ms), 95);

  _setText('stat-total',   total);
  _setText('stat-success', total > 0 ? Math.round(ok / total * 100) + '%' : '0%');
  _setText('stat-avg',     avg + 'ms');
  _setText('stat-errors',  errors);
  _setText('stat-p95',     p95 + 'ms');
}

function _createTrafficItem(log) {
  const sc   = _statusClass(log.status);
  const time = new Date(log.timestamp).toLocaleTimeString('tr-TR');
  const slow = log.duration_ms > 500;
  const div  = document.createElement('div');
  div.className = 'traffic-item' + (log.status >= 400 ? ' traffic-item--err' : '');
  div.innerHTML = `
    <span class="method-badge ${log.method}">${log.method}</span>
    <span class="traffic-path" title="${_esc(log.path)}">${_esc(log.path)}</span>
    <span class="traffic-status ${sc}">${log.status}</span>
    <span class="traffic-dur${slow ? ' traffic-dur--slow' : ''}">${log.duration_ms}ms</span>
    <span class="traffic-time">${time}</span>
  `;
  return div;
}

function renderTraffic() {
  const miniEl = _el('mini-traffic');
  const fullEl = _el('full-traffic');
  if (!miniEl || !fullEl) return;

  const filtered = _filteredLogs();
  miniEl.innerHTML = '';
  fullEl.innerHTML = '';

  if (filtered.length === 0) {
    const msg = '<div class="traffic-empty">Trafik bekleniyor…</div>';
    miniEl.innerHTML = msg;
    fullEl.innerHTML = msg;
    return;
  }

  filtered.slice(0, 8).forEach(log => miniEl.appendChild(_createTrafficItem(log)));
  filtered.forEach(log => fullEl.appendChild(_createTrafficItem(log)));
}

function setTrafficFilter(filter, btn) {
  state.trafficFilter = filter;
  document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active-filter'));
  if (btn) btn.classList.add('active-filter');
  renderTraffic();
}

function _filteredLogs() {
  const { allLogs, trafficFilter } = state;
  if (trafficFilter === 'err') return allLogs.filter(l => l.status >= 400);
  if (trafficFilter === '2xx') return allLogs.filter(l => l.status >= 200 && l.status < 300);
  if (trafficFilter)           return allLogs.filter(l => l.method === trafficFilter);
  return allLogs;
}

function renderLogs() {
  _renderLogsTable(state.allLogs);
  const cnt = _el('log-count');
  if (cnt) cnt.textContent = state.allLogs.length + ' kayıt';
}

function filterLogs() {
  const search  = (_el('log-search')?.value   || '').toLowerCase();
  const statusF = _el('log-status-filter')?.value || '';
  const methodF = _el('log-method-filter')?.value || '';

  const filtered = state.allLogs.filter(l => {
    const matchS = !search  || l.path.toLowerCase().includes(search) || l.ip.includes(search);
    const matchC = !statusF || String(l.status).startsWith(statusF);
    const matchM = !methodF || l.method === methodF;
    return matchS && matchC && matchM;
  });

  _renderLogsTable(filtered);
  const cnt = _el('log-count');
  if (cnt) cnt.textContent = filtered.length + ' kayıt';
}

function _renderLogsTable(logs) {
  const tbody = _el('logs-tbody');
  if (!tbody) return;

  if (logs.length === 0) {
    tbody.innerHTML = '<tr><td colspan="6" class="td-empty">Kayıt bulunamadı</td></tr>';
    return;
  }

  tbody.innerHTML = logs.map(l => {
    const sc   = _statusClass(l.status);
    const time = new Date(l.timestamp).toLocaleString('tr-TR');
    const slow = l.duration_ms > 500;
    return `<tr class="${l.status >= 400 ? 'row-err' : ''}">
      <td class="td-time">${time}</td>
      <td><span class="method-badge ${l.method}">${l.method}</span></td>
      <td class="td-path" title="${_esc(l.path)}">${_esc(l.path)}</td>
      <td><span class="status-pill ${sc}">${l.status}</span></td>
      <td class="${slow ? 'td-slow' : 'td-dur'}">${l.duration_ms}ms</td>
      <td class="td-ip">${_esc(l.ip)}</td>
    </tr>`;
  }).join('');
}

/* Route map */
function renderRouteMap() {
  const el = _el('route-map');
  if (!el) return;
  const counts = {};
  state.allLogs.forEach(l => { counts[l.path] = (counts[l.path] || 0) + 1; });
  const sorted = Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, 7);
  const max    = sorted[0]?.[1] || 1;
  el.innerHTML = sorted.length === 0
    ? '<div class="traffic-empty">Veri bekleniyor…</div>'
    : sorted.map(([path, cnt]) => `
        <div class="route-item">
          <span class="route-path" title="${_esc(path)}">${_esc(path)}</span>
          <div class="route-bar-wrap">
            <div class="route-bar" style="width:${(cnt / max * 100).toFixed(1)}%"></div>
          </div>
          <span class="route-count">${cnt}</span>
        </div>`).join('');
}

/* Service cards */
function renderServiceCards() {
  const services = [
    { name: 'Dispatcher',    port: 8080, color: 'amber', paths: ['/admin'] },
    { name: 'Auth Service',  port: 8081, color: 'blue',  paths: ['/login', '/register'] },
    { name: 'Event Service', port: 8082, color: 'teal',  paths: ['/events'] },
  ];

  const grid = _el('services-grid');
  if (!grid) return;

  grid.innerHTML = services.map(svc => {
    const svcLogs = state.allLogs.filter(l => svc.paths.some(p => l.path.includes(p)));
    const req     = svcLogs.length;
    const errors  = svcLogs.filter(l => l.status >= 400).length;
    const avg     = req > 0 ? Math.round(svcLogs.reduce((s, l) => s + l.duration_ms, 0) / req) : 0;
    const errRate = req > 0 ? Math.round(errors / req * 100) : 0;
    const healthy = errRate < 5;

    return `
      <div class="svc-status-card svc-${svc.color}">
        <div class="svc-status-top">
          <div>
            <div class="svc-status-name">${svc.name}</div>
            <div class="svc-port-label">:${svc.port}</div>
          </div>
          <div class="svc-health-badge ${healthy ? 'healthy' : 'degraded'}">
            <div class="svc-status-dot ${healthy ? 'up' : 'warn'}"></div>
            ${healthy ? 'Sağlıklı' : 'Dikkat'}
          </div>
        </div>
        <div class="svc-metrics">
          <div class="svc-metric">
            <div class="svc-metric-val">${req}</div>
            <div class="svc-metric-key">İstek</div>
          </div>
          <div class="svc-metric">
            <div class="svc-metric-val">${avg}ms</div>
            <div class="svc-metric-key">Ort. Süre</div>
          </div>
          <div class="svc-metric">
            <div class="svc-metric-val ${errors > 0 ? 'val-danger' : 'val-ok'}">${errRate}%</div>
            <div class="svc-metric-key">Hata Oranı</div>
          </div>
        </div>
        <div class="svc-db-tag">MongoDB · db-${svc.name.split(' ')[0].toLowerCase()}</div>
      </div>`;
  }).join('');
}


const _GRID  = 'rgba(255,255,255,0.04)';
const _TICKS = { color: '#718096', font: { family: 'IBM Plex Mono', size: 10 } };
const _TIP   = { backgroundColor: '#0e1320', borderColor: 'rgba(99,179,237,0.2)', borderWidth: 1, titleColor: '#718096', bodyColor: '#e2e8f0' };

function initCharts() {
  const ctxDur  = _el('chart-duration');
  const ctxStat = _el('chart-status');

  if (ctxDur && !state.durChart) {
    state.durChart = new Chart(ctxDur, {
      type: 'line',
      data: { labels: [], datasets: [{ data: [], borderColor: '#63b3ed', backgroundColor: 'rgba(99,179,237,0.08)', borderWidth: 1.5, fill: true, tension: 0.4, pointRadius: 2, pointBackgroundColor: '#63b3ed', pointHoverRadius: 4 }] },
      options: {
        responsive: true, maintainAspectRatio: false, animation: { duration: 300 },
        plugins: { legend: { display: false }, tooltip: { ..._TIP, callbacks: { label: ctx => ctx.parsed.y + 'ms' } } },
        scales: {
          x: { display: false },
          y: { grid: { color: _GRID }, ticks: _TICKS, border: { display: false } },
        },
      },
    });
  }

  if (ctxStat && !state.statusChart) {
    state.statusChart = new Chart(ctxStat, {
      type: 'doughnut',
      data: { labels: ['2xx Başarı', '4xx İstemci', '5xx Sunucu'], datasets: [{ data: [0, 0, 0], backgroundColor: ['#68d391', '#f6ad55', '#fc8181'], borderWidth: 0, hoverOffset: 6 }] },
      options: {
        responsive: true, maintainAspectRatio: false, cutout: '68%',
        plugins: {
          legend: { position: 'bottom', labels: { color: '#718096', font: { family: 'IBM Plex Mono', size: 10 }, padding: 14, boxWidth: 10 } },
          tooltip: _TIP,
        },
      },
    });
  }
}

function updateCharts() {
  if (state.durChart) {
    const recent = state.allLogs.slice(0, 30).reverse();
    state.durChart.data.labels           = recent.map((_, i) => i);
    state.durChart.data.datasets[0].data = recent.map(l => l.duration_ms);
    state.durChart.update('none');
  }
  if (state.statusChart) {
    const s2 = state.allLogs.filter(l => l.status >= 200 && l.status < 300).length;
    const s4 = state.allLogs.filter(l => l.status >= 400 && l.status < 500).length;
    const s5 = state.allLogs.filter(l => l.status >= 500).length;
    state.statusChart.data.datasets[0].data = [s2, s4, s5];
    state.statusChart.update('none');
  }
}

function renderServiceCharts() {
  if (state.methodChart) { state.methodChart.destroy(); state.methodChart = null; }
  if (state.routeChart)  { state.routeChart.destroy();  state.routeChart  = null; }

  const ctxM = _el('chart-methods');
  const ctxR = _el('chart-routes');
  const mColors = { GET: '#4fd1c5', POST: '#63b3ed', PUT: '#f6ad55', DELETE: '#fc8181', PATCH: '#b794f4' };

  const methods = {};
  state.allLogs.forEach(l => { methods[l.method] = (methods[l.method] || 0) + 1; });
  if (ctxM) {
    state.methodChart = new Chart(ctxM, {
      type: 'bar',
      data: { labels: Object.keys(methods), datasets: [{ data: Object.values(methods), backgroundColor: Object.keys(methods).map(m => mColors[m] || '#718096'), borderWidth: 0, borderRadius: 4 }] },
      options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false }, tooltip: _TIP }, scales: { x: { grid: { display: false }, ticks: _TICKS, border: { display: false } }, y: { grid: { color: _GRID }, ticks: _TICKS, border: { display: false } } } },
    });
  }


  const routes = {};
  state.allLogs.forEach(l => { routes[l.path] = (routes[l.path] || 0) + 1; });
  const top = Object.entries(routes).sort((a, b) => b[1] - a[1]).slice(0, 6);
  if (ctxR) {
    state.routeChart = new Chart(ctxR, {
      type: 'bar',
      data: { labels: top.map(([p]) => p), datasets: [{ data: top.map(([, c]) => c), backgroundColor: 'rgba(99,179,237,0.5)', borderWidth: 0, borderRadius: 4 }] },
      options: { indexAxis: 'y', responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false }, tooltip: _TIP }, scales: { x: { grid: { color: _GRID }, ticks: _TICKS, border: { display: false } }, y: { grid: { display: false }, ticks: { color: '#718096', font: { family: 'IBM Plex Mono', size: 9 } }, border: { display: false } } } },
    });
  }
}

const ENDPOINTS = [
  { label: 'POST Login',                method: 'POST',   path: '/login',       body: '{\n  "username": "user",\n  "password": "pass"\n}',                                                              auth: false },
  { label: 'POST Register',             method: 'POST',   path: '/register',    body: '{\n  "username": "newuser",\n  "password": "pass123"\n}',                                                        auth: false },
  { label: 'POST Admin Register',       method: 'POST',   path: '/register',    body: '{\n  "username": "admin",\n  "password": "pass",\n  "admin_secret": "admin_kayit_gizli_anahtari_2026"\n}',      auth: false },
  { label: 'GET All Events',            method: 'GET',    path: '/events/',      body: '',                                                                                                                auth: true  },
  { label: 'POST Create Event (admin)', method: 'POST',   path: '/events/',      body: '{\n  "name": "Kocaeli Tech Fest",\n  "location": "Kocaeli",\n  "capacity": 500,\n  "date": "2025-09-15"\n}',   auth: true  },
  { label: 'GET Event Detail',          method: 'GET',    path: '/events/{id}',  body: '',                                                                                                                auth: true  },
  { label: 'PUT Update Event (admin)',  method: 'PUT',    path: '/events/{id}',  body: '{\n  "name": "Updated",\n  "location": "Istanbul",\n  "capacity": 300,\n  "date": "2025-10-01"\n}',             auth: true  },
  { label: 'DELETE Event (admin)',      method: 'DELETE', path: '/events/{id}',  body: '',                                                                                                                auth: true  },
  { label: 'PATCH Ticket Capacity',    method: 'PATCH',  path: '/events/{id}',  body: '{\n  "amount": -1\n}',                                                                                           auth: true  },
  { label: 'GET Admin Stats',           method: 'GET',    path: '/admin/stats',  body: '',                                                                                                                auth: true  },
];

function buildEndpointList() {
  const el = _el('endpoint-list');
  if (!el) return;
  el.innerHTML = ENDPOINTS.map((ep, i) => `
    <button class="endpoint-btn" onclick="selectEndpoint(${i}, this)">
      <span class="method-badge ${ep.method}">${ep.method}</span>
      <span class="endpoint-btn-label">${ep.label}</span>
    </button>
  `).join('');
}

function selectEndpoint(i, btn) {
  document.querySelectorAll('.endpoint-btn').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  const ep = ENDPOINTS[i];
  _el('api-panel-title').textContent        = ep.label;
  _el('api-url').value                      = API + ep.path;
  _el('api-method').value                   = ep.method;
  _el('api-body').value                     = ep.body;
  _el('api-token').value                    = ep.auth && state.token ? 'Bearer ' + state.token : '';
  _el('api-response').textContent           = '—';
  _el('api-response-status').textContent    = '';
  _el('api-response-time').textContent      = '';
}

async function sendApiRequest() {
  const url    = _el('api-url')?.value;
  const method = _el('api-method')?.value;
  const body   = _el('api-body')?.value;
  const tok    = _el('api-token')?.value;
  const respEl = _el('api-response');
  const statEl = _el('api-response-status');
  const timeEl = _el('api-response-time');
  if (!url || !respEl) return;

  respEl.textContent = 'Gönderiliyor…';
  statEl.textContent = '';
  timeEl.textContent = '';

  const start = Date.now();
  try {
    const headers = { 'Content-Type': 'application/json' };
    if (tok) headers['Authorization'] = tok;
    const opts = { method, headers };
    if (body && method !== 'GET' && body.trim()) opts.body = body;

    const res  = await fetch(url, opts);
    const dur  = Date.now() - start;
    const txt  = await res.text();
    let pretty;
    try { pretty = JSON.stringify(JSON.parse(txt), null, 2); } catch { pretty = txt; }

    respEl.textContent  = pretty;
    statEl.textContent  = res.status + ' ' + res.statusText;
    statEl.style.color  = res.status >= 500 ? 'var(--danger)' : res.status >= 400 ? 'var(--warn)' : 'var(--success)';
    timeEl.textContent  = dur + 'ms';
  } catch (e) {
    respEl.textContent  = 'Ağ hatası: ' + e.message;
    statEl.textContent  = 'NETWORK ERROR';
    statEl.style.color  = 'var(--danger)';
  }
}

function showPage(pageId, linkEl) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  _el('page-' + pageId)?.classList.add('active');
  document.querySelectorAll('nav a').forEach(a => a.classList.remove('active'));
  if (linkEl) linkEl.classList.add('active');
  if (pageId === 'services') renderServiceCharts();
}

function startTokenTimer() {
  if (state.timerInterval) clearInterval(state.timerInterval);
  const el = _el('token-timer');
  if (!el) return;

  state.timerInterval = setInterval(() => {
    const left = state.tokenExp - Date.now();
    if (left <= 0) {
      el.textContent = '⏱ Süresi doldu';
      el.className   = 'token-timer danger';
      clearInterval(state.timerInterval);
      logout();
      return;
    }
    const h = Math.floor(left / 3_600_000);
    const m = Math.floor((left % 3_600_000) / 60_000);
    const s = Math.floor((left % 60_000) / 1000);
    el.textContent = h > 0
      ? `⏱ ${h}sa ${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`
      : `⏱ ${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;
    el.className = left < 300_000 ? 'token-timer danger' : 'token-timer';
  }, 1000);
}

function showToast(msg, type = '') {
  const el = _el('toast');
  if (!el) return;
  el.textContent = msg;
  el.className   = 'show ' + type;
  setTimeout(() => { el.className = ''; }, 3500);
}

function _el(id)         { return document.getElementById(id); }
function _setText(id, v) { const e = _el(id); if (e) e.textContent = v; }
function _esc(str)        { return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

function _statusClass(status) {
  if (status >= 500) return 's500';
  if (status === 404) return 's404';
  if (status === 403) return 's403';
  if (status === 401) return 's401';
  if (status >= 400)  return 's400';
  if (status >= 200)  return 's200';
  return '';
}

function _percentile(arr, p) {
  if (!arr.length) return 0;
  const sorted = [...arr].sort((a, b) => a - b);
  return sorted[Math.max(0, Math.ceil(p / 100 * sorted.length) - 1)];
}