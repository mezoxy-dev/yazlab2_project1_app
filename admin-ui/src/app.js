/* ============================================================
   Etkinlik Merkezi — Kurumsal UI Mantığı
   Author: Antigravity Engineer
   Description: Event Listeners tabanlı, çakışmasız oturum yönetimi.
   ============================================================ */

// ---------- GLOBAL DURUM ----------
window.userToken   = null;
window.userRole    = null;
window.username    = null;
window.adminToken  = null;
window.logInterval = null;
window.knownIds    = new Set();
window.isPaused    = false;
window.filterMode  = 'all';

let stats = { total: 0, warnings: 0, critical: 0, latencies: [] };

// ---------- API YARDIMCI ----------
async function apiCall(path, opts = {}) {
    const baseUrl = '/api'; 
    try {
        const response = await fetch(baseUrl + path, opts);
        return response;
    } catch (err) {
        console.error(`API Error (${path}):`, err);
        throw new Error('Sunucuyla bağlantı kurulamadı.');
    }
}

// ---------- BİLDİRİMLER ----------
function showFeedback(id, msg, type = 'error') {
    const el = document.getElementById(id);
    if (!el) return showToast(msg, type);
    el.textContent = msg;
    el.className = 'msg-box ' + type;
    setTimeout(() => { if(el) { el.className = 'msg-box'; el.textContent = ''; } }, 5000);
}

function showToast(msg, type = 'success') {
    const t = document.createElement('div');
    t.textContent = msg;
    const bgColor = type === 'success' ? 'rgba(16,185,129,0.95)' : 'rgba(239,68,68,0.95)';
    t.style = `position:fixed;bottom:24px;left:50%;transform:translateX(-50%);background:${bgColor};color:white;padding:14px 28px;border-radius:12px;z-index:9999;font-weight:600;box-shadow:0 8px 32px rgba(0,0,0,0.4);backdrop-filter:blur(8px);transition:all 0.4s ease;font-family:Outfit,sans-serif;`;
    document.body.appendChild(t);
    setTimeout(() => { t.style.opacity = '0'; setTimeout(() => t.remove(), 400); }, 4000);
}

// ---------- GİRİŞ/KAYIT MANTIĞI ----------
async function handleUserLogin() {
    const u = document.getElementById('login-username').value.trim();
    const p = document.getElementById('login-password').value;
    if (!u || !p) return showFeedback('auth-msg', 'Eksik bilgi.');

    try {
        const res = await apiCall('/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({username: u, password: p})
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Giriş reddedildi.');

        window.userToken = data.token;
        const payload = decodeJwt(data.token);
        window.userRole = payload.role || 'user';
        window.username = u;
        
        finalizeAuth();
        showToast('Portal girişi başarılı.', 'success');
    } catch(e) { showFeedback('auth-msg', '❌ ' + e.message); }
}

async function handleUserRegister() {
    const u = document.getElementById('reg-username').value.trim();
    const p = document.getElementById('reg-password').value;
    const s = document.getElementById('reg-admin-secret').value.trim();
    if (!u || !p) return showFeedback('auth-msg', 'Eksik bilgi.');
    const body = s ? {username: u, password: p, admin_secret: s} : {username: u, password: p};

    try {
        const res = await apiCall('/register', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(body)
        });
        if (!res.ok) throw new Error((await res.json()).error || 'Kayıt hatası.');
        showFeedback('auth-msg', '✅ Kayıt başarılı.', 'success');
        handleTabSwitch('login');
    } catch(e) { showFeedback('auth-msg', '❌ ' + e.message); }
}

function finalizeAuth() {
    safeToggle('auth-section', true);
    safeToggle('events-section', false);
    safeToggle('my-bookings-section', false);
    safeToggle('logout-section', false);

    const badge = document.getElementById('user-badge');
    if (badge) {
        badge.classList.remove('hidden');
        badge.innerHTML = `👤 ${window.username}${window.userRole === 'admin' ? ' <span class="badge-admin">ADMIN</span>' : ''}`;
    }
    if (window.userRole === 'admin') safeToggle('create-event-section', false);

    handleLoadEvents();
    handleLoadBookings();
}

function handlePortalLogout() {
    console.log("Portal Logout Executing...");
    window.userToken = null;
    window.userRole = null;
    window.username = null;
    
    safeToggle('auth-section', false);
    safeToggle('events-section', true);
    safeToggle('my-bookings-section', true);
    safeToggle('logout-section', true);
    safeToggle('create-event-section', true);
    safeToggle('user-badge', true);
    
    showToast('Portal oturumu kapatıldı.', 'success');
}

// ---------- ETKİNLİK YÖNETİMİ ----------
async function handleLoadEvents() {
    const list = document.getElementById('events-list');
    if (!list) return;
    list.innerHTML = '<div class="loading">Yükleniyor...</div>';
    try {
        const headers = window.userToken ? {'Authorization': `Bearer ${window.userToken}`} : {};
        const res = await apiCall('/events', { headers });
        const events = await res.json().catch(() => []);
        list.innerHTML = events.length === 0 ? '<div class="empty-state">Etkinlik yok.</div>' : '';
        events.forEach(ev => {
            const card = document.createElement('div');
            card.className = 'event-card';
            const avail = ev.available ?? ev.capacity ?? 0;
            const isSoldOut = avail <= 0;
            card.innerHTML = `
                <div class="event-info">
                    <div class="event-name">${escapeHTML(ev.name)}</div>
                    <div class="event-meta">📍 ${escapeHTML(ev.location)}</div>
                </div>
                <div style="text-align: right">
                    <div class="event-capacity" style="color:${isSoldOut ? 'var(--error)' : 'var(--success)'}">${avail} / ${ev.capacity}</div>
                    <button class="btn-book" data-id="${ev.id || ev._id}" data-name="${escapeHTML(ev.name)}" ${isSoldOut ? 'disabled' : ''}>
                        ${isSoldOut ? 'Tükendi' : 'Bilet Al'}
                    </button>
                </div>
            `;
            list.appendChild(card);
        });
    } catch(e) { list.innerHTML = '<div class="empty-state">Hata.</div>'; }
}

async function handleCreateEventSubmission() {
    try {
        const nameVal = document.getElementById('event-name').value.trim();
        const locVal = document.getElementById('event-location').value.trim();
        const capVal = parseInt(document.getElementById('event-capacity').value);
        if (!nameVal) throw new Error('Etkinlik adı gerekli.');

        const res = await apiCall('/events', {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'Authorization': `Bearer ${window.userToken}`},
            body: JSON.stringify({name: nameVal, location: locVal || 'Online', capacity: isNaN(capVal) ? 100 : capVal})
        });
        if (!res.ok) throw new Error('Kayıt başarısız.');
        showFeedback('create-event-msg', '✅ Etkinlik yayında.', 'success');
        document.getElementById('event-name').value = '';
        document.getElementById('event-location').value = '';
        await handleLoadEvents();
    } catch (err) { showFeedback('create-event-msg', '❌ ' + err.message); }
}

async function handleBookingSubmission(eventId, eventName) {
    if (!window.userToken) return showToast('Bilet almak için giriş yapın.', 'error');
    try {
        const res = await apiCall('/bookings', {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'Authorization': `Bearer ${window.userToken}`},
            body: JSON.stringify({event_id: eventId, user_id: window.username, seats: 1})
        });
        if (!res.ok) throw new Error('Kontenjan dolu veya yetki hatası.');
        showToast(`✅ "${eventName}" bileti alındı.`, 'success');
        handleLoadEvents(); handleLoadBookings();
    } catch(e) { showToast('Hata: ' + e.message, 'error'); }
}

async function handleLoadBookings() {
    if (!window.userToken) return;
    const list = document.getElementById('bookings-list');
    if (!list) return;
    try {
        const res = await apiCall(`/bookings?user_id=${encodeURIComponent(window.username)}`, {
            headers: {'Authorization': `Bearer ${window.userToken}`}
        });
        const bookings = await res.json().catch(() => []);
        list.innerHTML = bookings.length === 0 ? '<div class="empty-state">Biletiniz yok.</div>' : '';
        bookings.forEach(b => {
            const card = document.createElement('div');
            card.className = 'booking-card';
            card.style = "margin-bottom:8px; padding:12px; background:rgba(255,255,255,0.03); border-radius:8px; font-size:13px;";
            card.innerHTML = `🎫 <strong>${escapeHTML(b.event_id)}</strong> — 1 Koltuk`;
            list.appendChild(card);
        });
    } catch(e) { list.innerHTML = '<div class="empty-state">Hata.</div>'; }
}

// ---------- ADMIN LOG CENTER ----------
async function handleAdminLogin() {
    const u = document.getElementById('log-admin-username').value.trim();
    const p = document.getElementById('log-admin-password').value;
    try {
        const res = await apiCall('/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({username: u, password: p})
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Giriş başarısız.');
        const payload = decodeJwt(data.token);
        if (payload.role !== 'admin') throw new Error('Admin yetkisi gerekli.');
        window.adminToken = data.token;
        safeToggle('log-login-section', true);
        safeToggle('log-table-wrapper', false);
        startLogStream();
        showToast('Hattat stream başlatıldı.', 'success');
    } catch(e) { showFeedback('log-login-msg', '❌ ' + e.message); }
}

function handleAdminLogout() {
    console.log("Admin Logout Executing...");
    window.adminToken = null;
    if (window.logInterval) clearInterval(window.logInterval);
    window.logInterval = null;
    safeToggle('log-login-section', false);
    safeToggle('log-table-wrapper', true);
    showToast('Log merkezi kapatıldı.', 'success');
}

function startLogStream() {
    if (window.logInterval) clearInterval(window.logInterval);
    updateStreamBadge();
    fetchLogs();
    window.logInterval = setInterval(fetchLogs, 2000);
}

async function fetchLogs() {
    if (window.isPaused || !window.adminToken) return;
    try {
        const res = await apiCall('/admin/stats', {
            headers: {'Authorization': `Bearer ${window.adminToken}`}
        });
        if (res.status === 401 || res.status === 403) { handleAdminLogout(); return; }
        const logs = await res.json().catch(() => []);
        processLogs(logs);
    } catch (e) { }
}

function processLogs(logs) {
    const tbody = document.getElementById('log-body');
    if (!tbody) return;
    const newLogs = logs.filter(l => !window.knownIds.has(l.id || l._id)).reverse();
    newLogs.forEach(log => {
        window.knownIds.add(log.id || log._id);
        stats.total++;
        if (log.status >= 500) stats.critical++;
        else if (log.status >= 400) stats.warnings++;
        if (log.duration_ms) stats.latencies.push(log.duration_ms);
        const tr = document.createElement('tr');
        tr.className = 'new-row';
        const d = new Date(log.timestamp);
        const timeStr = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
        tr.innerHTML = `
            <td style="color:var(--text-muted)">${timeStr}</td>
            <td><span class="method-${log.method}">${log.method}</span></td>
            <td style="max-width:140px;overflow:hidden;text-overflow:ellipsis">${log.path}</td>
            <td class="${log.status >= 500 ? 's-5xx' : (log.status >= 400 ? 's-4xx' : 's-2xx')}">${log.status}</td>
            <td>${log.duration_ms}ms</td>
            <td style="opacity:0.6; font-size:10px">${cleanString(log.message)}</td>
        `;
        tbody.prepend(tr);
    });
    while (tbody.children.length > 150) tbody.removeChild(tbody.lastChild);
    updateGlobalStats();
    applyFilter();
}

function updateGlobalStats() {
    document.getElementById('stat-total').textContent = `${stats.total} LOG`;
    document.getElementById('stat-critical').textContent = `🚨 ${stats.critical} KRİTİK`;
    document.getElementById('stat-warning').textContent = `⚠️ ${stats.warnings} UYARI`;
    const avg = stats.latencies.length ? Math.round(stats.latencies.reduce((a,b)=>a+b,0)/stats.latencies.length) : 0;
    const el = document.getElementById('stat-latency');
    if(el) el.textContent = `AVG: ${avg}ms`;
}

// ---------- UI KONTROLLERİ ----------
function handleTabSwitch(tab) {
    document.getElementById('tab-login')?.classList.toggle('active', tab === 'login');
    document.getElementById('tab-register')?.classList.toggle('active', tab === 'register');
    safeToggle('form-login', tab !== 'login');
    safeToggle('form-register', tab !== 'register');
}

function safeToggle(id, isHidden) {
    const el = document.getElementById(id);
    if (el) {
        if (isHidden) el.classList.add('hidden');
        else el.classList.remove('hidden');
    }
}

function updateStreamBadge() {
    const el = document.getElementById('log-status');
    if (!el) return;
    el.innerHTML = window.isPaused ? 
        '<span class="pulse" style="background:var(--warning)"></span><span style="color:var(--warning)">DURDURULDU</span>' :
        '<span class="pulse"></span><span style="color:var(--success)">STREAMING</span>';
}

function applyFilter() {
    document.querySelectorAll('#log-body tr').forEach(row => {
        const code = parseInt(row.cells[3].textContent);
        if (window.filterMode === '5xx') row.style.display = code >= 500 ? '' : 'none';
        else if (window.filterMode === '4xx') row.style.display = (code >= 400 && code < 500) ? '' : 'none';
        else row.style.display = '';
    });
}

function decodeJwt(t) {
    try { return JSON.parse(atob(t.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))); }
    catch(e) { return {}; }
}
function pad(n) { return String(n).padStart(2, '0'); }
function cleanString(s) { return s ? s.replace(/[{}"]/g, '').substring(0, 40) : '-'; }
function escapeHTML(str) {
    return str ? str.replace(/[&<>"']/g, m => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m])) : '';
}

// ---------- MODERM OLAY DİNLEYİCİLER ----------
document.addEventListener('DOMContentLoaded', () => {
    console.log("Modern Event Architecture Active");

    // Login & Register
    document.getElementById('btn-login')?.addEventListener('click', handleUserLogin);
    document.getElementById('btn-register')?.addEventListener('click', handleUserRegister);
    
    // Tabs
    document.getElementById('btn-tab-login')?.addEventListener('click', () => handleTabSwitch('login'));
    document.getElementById('btn-tab-register')?.addEventListener('click', () => handleTabSwitch('register'));

    // Logoutlar
    document.getElementById('btn-portal-logout')?.addEventListener('click', (e) => {
        e.preventDefault();
        handlePortalLogout();
    });
    document.getElementById('btn-admin-logout')?.addEventListener('click', (e) => {
        e.preventDefault();
        handleAdminLogout();
    });

    // Create Event
    document.getElementById('btn-create-event')?.addEventListener('click', handleCreateEventSubmission);

    // Refreshers
    document.getElementById('btn-refresh-events')?.addEventListener('click', handleLoadEvents);
    document.getElementById('btn-refresh-bookings')?.addEventListener('click', handleLoadBookings);

    // Admin Controls
    document.getElementById('btn-admin-login')?.addEventListener('click', handleAdminLogin);
    document.getElementById('btn-pause')?.addEventListener('click', () => {
        window.isPaused = !window.isPaused;
        const btn = document.getElementById('btn-pause');
        if (btn) btn.textContent = window.isPaused ? '▶️ Başlat' : '⏸️ Duraklat';
        updateStreamBadge();
    });

    // Delegasyon: Bilet Al butonları dinamik olduğu için ana listeyi dinliyoruz
    document.getElementById('events-list')?.addEventListener('click', (e) => {
        const btn = e.target.closest('.btn-book');
        if (btn && !btn.disabled) {
            handleBookingSubmission(btn.dataset.id, btn.dataset.name);
        }
    });

    // Filtreler
    document.getElementById('stat-critical')?.addEventListener('click', () => {
        window.filterMode = window.filterMode === '5xx' ? 'all' : '5xx';
        window.isPaused = (window.filterMode !== 'all');
        updateStreamBadge();
        applyFilter();
    });
    document.getElementById('stat-warning')?.addEventListener('click', () => {
        window.filterMode = window.filterMode === '4xx' ? 'all' : '4xx';
        window.isPaused = (window.filterMode !== 'all');
        updateStreamBadge();
        applyFilter();
    });
});
