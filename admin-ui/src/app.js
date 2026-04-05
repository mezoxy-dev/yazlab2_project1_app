/* ============================================================
   EventHub — Robust & Professional UI Logic
   Author: Antigravity Engineer
   Description: Fault-tolerant JS architecture with clear separation
   ============================================================ */

// ---------- GLOBAL STATE ----------
window.userToken   = null;
window.userRole    = null;
window.username    = null;
window.adminToken  = null;
window.logInterval = null;
window.knownIds    = new Set();
window.isPaused    = false;
window.filterMode  = 'all';

// Stats state
let stats = { total: 0, warnings: 0, critical: 0, latencies: [] };

// ---------- API HELPER ----------
async function apiCall(path, opts = {}) {
    const baseUrl = '/api';
    try {
        const response = await fetch(baseUrl + path, opts);
        return response;
    } catch (err) {
        console.error(`API Error (${path}):`, err);
        throw new Error('Sunucuya erişilemiyor. Lütfen bağlantınızı kontrol edin.');
    }
}

// ---------- NOTIFICATIONS ----------
function showFeedback(id, msg, type = 'error') {
    const el = document.getElementById(id);
    if (!el) {
        // Fallback to toast if specific ID not found
        showToast(msg, type);
        return;
    }
    el.textContent = msg;
    el.className = 'msg-box ' + type;
    setTimeout(() => {
        el.className = 'msg-box';
        el.textContent = '';
    }, 5000);
}

function showToast(msg, type = 'success') {
    const t = document.createElement('div');
    t.textContent = msg;
    const bgColor = type === 'success' ? 'rgba(16,185,129,0.95)' : 'rgba(239,68,68,0.95)';
    t.style = `position:fixed;bottom:24px;left:50%;transform:translateX(-50%);background:${bgColor};color:white;padding:14px 28px;border-radius:12px;z-index:9999;font-weight:600;box-shadow:0 8px 32px rgba(0,0,0,0.4);backdrop-filter:blur(8px);transition:all 0.3s ease;font-family:Inter,sans-serif;`;
    document.body.appendChild(t);
    setTimeout(() => {
        t.style.opacity = '0';
        setTimeout(() => t.remove(), 400);
    }, 4000);
}

// ---------- AUTH LOGIC ----------
window.doLogin = async function() {
    const u = document.getElementById('login-username').value.trim();
    const p = document.getElementById('login-password').value;
    if (!u || !p) return showFeedback('auth-msg', 'Kullanıcı adı ve şifre gerekli');

    try {
        const res = await apiCall('/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({username: u, password: p})
        });

        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
            showFeedback('auth-msg', `❌ ${data.error || 'Giriş başarısız'} (${res.status})`);
            return;
        }

        window.userToken = data.token;
        const payload = decodeJwt(data.token);
        window.userRole = payload.role || 'user';
        window.username = u;
        
        finalizeAuth();
    } catch(e) {
        showFeedback('auth-msg', '❌ Hata: ' + e.message);
    }
};

window.doRegister = async function() {
    const u = document.getElementById('reg-username').value.trim();
    const p = document.getElementById('reg-password').value;
    const s = document.getElementById('reg-admin-secret').value.trim();
    if (!u || !p) return showFeedback('auth-msg', 'Eksik bilgi girişi');

    const body = s ? {username: u, password: p, admin_secret: s} : {username: u, password: p};

    try {
        const res = await apiCall('/register', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(body)
        });

        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
            showFeedback('auth-msg', `❌ ${data.error || 'Kayıt başarısız'}`);
            return;
        }

        showFeedback('auth-msg', '✅ Başarı: Giriş yapabilirsiniz.', 'success');
        window.switchTab('login');
    } catch(e) {
        showFeedback('auth-msg', '❌ Hata: ' + e.message);
    }
};

function finalizeAuth() {
    document.getElementById('auth-section').classList.add('hidden');
    document.getElementById('events-section').classList.remove('hidden');
    document.getElementById('my-bookings-section').classList.remove('hidden');
    document.getElementById('logout-section').classList.remove('hidden');

    const badge = document.getElementById('user-badge');
    badge.classList.remove('hidden');
    badge.innerHTML = `👤 ${window.username}${window.userRole === 'admin' ? ' <span class="badge-admin">ADMIN</span>' : ''}`;

    if (window.userRole === 'admin') {
        document.getElementById('create-event-section').classList.remove('hidden');
    }

    window.loadEvents();
    window.loadMyBookings();
}

window.doLogout = function() {
    location.reload(); 
};

// ---------- EVENT MANAGEMENT ----------
window.loadEvents = async function() {
    const list = document.getElementById('events-list');
    list.innerHTML = '<div class="loading">Sistem taranıyor...</div>';

    try {
        const headers = window.userToken ? {'Authorization': `Bearer ${window.userToken}`} : {};
        const res = await apiCall('/events', { headers });

        if (!res.ok) {
            list.innerHTML = '<div class="empty-state">Sistem meşgul, daha sonra tekrar deneyin.</div>';
            return;
        }

        const events = await res.json().catch(() => []);
        if (events.length === 0) {
            list.innerHTML = '<div class="empty-state">Henüz aktif bir etkinlik bulunmuyor.</div>';
            return;
        }

        list.innerHTML = '';
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
                <div class="event-meta-right">
                    <span class="event-capacity" style="color:${isSoldOut ? 'var(--error)' : 'var(--success)'}">🎫 ${avail} / ${ev.capacity}</span>
                    <button class="btn-book" ${isSoldOut ? 'disabled' : ''} onclick="window.bookEvent('${ev.id || ev._id}', '${escapeHTML(ev.name)}')">
                        ${isSoldOut ? 'Tükendi' : 'Bilet Al'}
                    </button>
                </div>
            `;
            list.appendChild(card);
        });
    } catch(e) {
        list.innerHTML = `<div class="empty-state" style="color:var(--error)">⚠️ Hata: ${e.message}</div>`;
    }
};

window.createEvent = async function(evt) {
    // Prevent default and stop propagation
    if (evt) {
        if (evt.preventDefault) evt.preventDefault();
        evt.stopPropagation();
    }

    const btn = evt ? (evt.currentTarget || evt.target) : document.querySelector('.btn-success');
    if (!btn) return console.error('Create button not found in DOM');

    const originalText = btn.textContent;

    try {
        const nameVal     = document.getElementById('event-name').value.trim();
        const locationVal = document.getElementById('event-location').value.trim();
        const capacityVal = parseInt(document.getElementById('event-capacity').value);

        if (!nameVal) return showFeedback('create-event-msg', '❌ Hata: Etkinlik adı boş olamaz.');
        
        btn.disabled = true;
        btn.textContent = '⏱️ İşleniyor...';

        const res = await apiCall('/events', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${window.userToken}`
            },
            body: JSON.stringify({
                name: nameVal, 
                location: locationVal || 'TBD', 
                capacity: isNaN(capacityVal) ? 100 : capacityVal
            })
        });

        if (!res.ok) {
            const errorTxt = await res.text().catch(() => 'Sunucu hatası');
            throw new Error(errorTxt);
        }

        showFeedback('create-event-msg', '✅ Başarı: Etkinlik başarıyla yayına alındı.', 'success');
        
        // Reset form
        document.getElementById('event-name').value = '';
        document.getElementById('event-location').value = '';
        document.getElementById('event-capacity').value = '100';
        
        // Refresh list
        await window.loadEvents();
    } catch (err) {
        showFeedback('create-event-msg', '❌ Hata: ' + err.message);
    } finally {
        btn.disabled = false;
        btn.textContent = originalText;
    }
};

window.bookEvent = async function(eventId, eventName) {
    if (!window.userToken) return showToast('Önce giriş yapmalısınız.', 'error');

    try {
        const res = await apiCall('/bookings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${window.userToken}`
            },
            body: JSON.stringify({event_id: eventId, user_id: window.username, seats: 1})
        });

        if (!res.ok) {
            const data = await res.json().catch(() => ({error: 'İşlem reddedildi'}));
            showToast(`❌ Hata: ${data.error}`, 'error');
            return;
        }

        showToast(`✅ "${eventName}" için biletiniz alındı!`, 'success');
        window.loadEvents();
        window.loadMyBookings();
    } catch(e) {
        showToast('Bağlantı hatası: ' + e.message, 'error');
    }
};

window.loadMyBookings = async function() {
    if (!window.userToken) return;
    const list = document.getElementById('bookings-list');

    try {
        const res = await apiCall(`/bookings?user_id=${encodeURIComponent(window.username)}`, {
            headers: {'Authorization': `Bearer ${window.userToken}`}
        });

        const bookings = await res.json().catch(() => []);
        list.innerHTML = bookings.length === 0 ? '<div class="empty-state">Henüz biletiniz yok.</div>' : '';
        
        bookings.forEach(b => {
            const card = document.createElement('div');
            card.className = 'booking-card';
            card.innerHTML = `🎫 <strong>${escapeHTML(b.event_id)}</strong> — 1 Koltuk`;
            list.appendChild(card);
        });
    } catch(e) {
        list.innerHTML = '<div class="empty-state" style="color:var(--error)">Veriler yüklenemedi.</div>';
    }
};

// ---------- ADMIN LOG CENTER ----------
window.doAdminLogin = async function() {
    const u = document.getElementById('log-admin-username').value.trim();
    const p = document.getElementById('log-admin-password').value;
    
    try {
        const res = await apiCall('/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({username: u, password: p})
        });

        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || 'Giriş reddedildi');

        const payload = decodeJwt(data.token);
        if (payload.role !== 'admin') throw new Error('Yetersiz yetki.');

        window.adminToken = data.token;
        document.getElementById('log-login-section').classList.add('hidden');
        document.getElementById('log-table-wrapper').classList.remove('hidden');
        
        startLogStream();
    } catch(e) {
        showFeedback('log-login-msg', '❌ ' + e.message);
    }
};

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
        if (res.status === 401 || res.status === 403) {
            clearInterval(window.logInterval);
            location.reload();
            return;
        }
        const logs = await res.json().catch(() => []);
        processLogs(logs);
    } catch (e) { console.warn('Stream error:', e); }
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
            <td><span class="badge method-${log.method}">${log.method}</span></td>
            <td style="max-width:140px;overflow:hidden;text-overflow:ellipsis">${log.path}</td>
            <td class="${log.status >= 500 ? 's-5xx' : (log.status >= 400 ? 's-4xx' : 's-2xx')}">${log.status}</td>
            <td>${log.duration_ms}ms</td>
            <td class="msg-cell">${cleanString(log.message)}</td>
        `;
        tbody.prepend(tr);
    });

    // Cleanup redundant DOM
    while (tbody.children.length > 100) tbody.removeChild(tbody.lastChild);
    
    updateGlobalStats();
    applyFilter();
}

function updateGlobalStats() {
    document.getElementById('stat-total').textContent = `Toplam: ${stats.total}`;
    document.getElementById('stat-critical').textContent = `🚨 Kritik: ${stats.critical}`;
    document.getElementById('stat-warning').textContent = `⚠️ Uyarı: ${stats.warnings}`;
    
    const avg = stats.latencies.length ? Math.round(stats.latencies.reduce((a,b)=>a+b,0)/stats.latencies.length) : 0;
    document.getElementById('stat-latency').textContent = `Ort: ${avg}ms`;
}

// ---------- UI CONTROL ----------
window.togglePause = function() {
    window.isPaused = !window.isPaused;
    const btn = document.getElementById('btn-pause');
    if (btn) {
        btn.textContent = window.isPaused ? '▶️ Akışı Başlat' : '⏸️ Duraklat';
        btn.classList.toggle('paused', window.isPaused);
    }
    updateStreamBadge();
};

window.setFilter = function(mode) {
    if (window.filterMode === mode) {
        window.filterMode = 'all';
        window.isPaused = false;
    } else {
        window.filterMode = mode;
        window.isPaused = true;
    }
    
    document.getElementById('stat-critical').classList.toggle('active', window.filterMode === '5xx');
    document.getElementById('stat-warning').classList.toggle('active', window.filterMode === '4xx');
    
    updateStreamBadge();
    applyFilter();
};

function updateStreamBadge() {
    const el = document.getElementById('log-status');
    if (!el) return;
    if (window.isPaused) {
        el.innerHTML = '<span class="pulse" style="background:var(--warning)"></span><span style="color:var(--warning)">DURDURULDU</span>';
    } else {
        el.innerHTML = '<span class="pulse"></span><span style="color:var(--success)">CANLI AKIŞ</span>';
    }
}

function applyFilter() {
    const rows = document.querySelectorAll('#log-body tr');
    rows.forEach(row => {
        const statusCell = row.cells[3];
        if (!statusCell) return;
        const code = parseInt(statusCell.textContent);
        if (window.filterMode === '5xx') row.style.display = code >= 500 ? '' : 'none';
        else if (window.filterMode === '4xx') row.style.display = (code >= 400 && code < 500) ? '' : 'none';
        else row.style.display = '';
    });
}

window.switchTab = function(tab) {
    document.getElementById('tab-login').classList.toggle('active', tab === 'login');
    document.getElementById('tab-register').classList.toggle('active', tab === 'register');
    document.getElementById('form-login').classList.toggle('hidden', tab !== 'login');
    document.getElementById('form-register').classList.toggle('hidden', tab !== 'register');
};

// ---------- UTILS ----------
function decodeJwt(t) {
    try { return JSON.parse(atob(t.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))); }
    catch(e) { return {}; }
}
function pad(n) { return String(n).padStart(2, '0'); }
function cleanString(s) { 
    if(!s) return '-';
    return s.replace(/[{}"]/g, '').substring(0, 60); 
}
function escapeHTML(str) {
    if(!str) return '';
    return str.replace(/[&<>"']/g, m => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
}

// ---------- BOOTSTRAP ----------
document.addEventListener('DOMContentLoaded', () => {
    // Initial fetch logs setup
    const critBadge = document.getElementById('stat-critical');
    const warnBadge = document.getElementById('stat-warning');
    if (critBadge) critBadge.onclick = () => window.setFilter('5xx');
    if (warnBadge) warnBadge.onclick = () => window.setFilter('4xx');
});
