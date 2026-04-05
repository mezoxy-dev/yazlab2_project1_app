import http from 'k6/http';
import { check, sleep } from 'k6';

// Uçtan Uca (End-to-End) Gerçekçi Kullanıcı Yolculuğu Testi
export let options = {
    stages: [
        { duration: '30s', target: 50 },
        { duration: '30s', target: 100 },
        { duration: '30s', target: 200 },
        { duration: '30s', target: 500 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<1500'],
        http_req_failed: ['rate<0.05'], 
    },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export function setup() {
    const timestamp = Date.now();
    const adminUser = `admin_${timestamp}`;
    const adminPass = "12345";
    
    // 1. Admin Kayıt ve Giriş
    http.post(`${BASE_URL}/register`, JSON.stringify({
        username: adminUser,
        password: adminPass,
        admin_secret: "admin_kayit_gizli_anahtari_2026"
    }), { headers: { 'Content-Type': 'application/json' } });

    const loginRes = http.post(`${BASE_URL}/login`, JSON.stringify({
        username: adminUser,
        password: adminPass
    }), { headers: { 'Content-Type': 'application/json' } });
    
    const adminToken = loginRes.json('token');

    // 2. Özel Etkinlik Oluştur (Gerçekçi bir kapasiteyle, sistem hatalarını da test etmek için)
    const eventName = `e2e_event_${timestamp}`;
    http.post(`${BASE_URL}/events`, JSON.stringify({
        name: eventName,
        location: "Gercekci Test Alani",
        capacity: 1000000 
    }), {
        headers: { 
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${adminToken}`
        }
    });

    const eventsRes = http.get(`${BASE_URL}/events`, {
        headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    
    const events = eventsRes.json();
    let eventId = null;
    if (events && Array.isArray(events)) {
        const matchedEvent = events.find(e => e.name === eventName);
        if (matchedEvent) eventId = matchedEvent.id;
    }

    // 3. Ortak Test Kullanıcısı (Tüm VUs bu hesapla dolaşsın ya da veri patlamaması için her biri kendi hesabını açabilir. Gerçekçilik için ortak hesap)
    const testUser = `e2e_user_${timestamp}`;
    const testPass = "pass123";
    http.post(`${BASE_URL}/register`, JSON.stringify({
        username: testUser,
        password: testPass
    }), { headers: { 'Content-Type': 'application/json' } });

    return { eventId: eventId, testUser: testUser, testPass: testPass };
}

export default function (data) {
    if (!data.eventId) return;

    // Her VU (Virtual User) kendi kullanıcısını kullansın.
    // Bu sayede tek bir kullanıcının bilet listesi şişmez ve test daha gerçekçi olur.
    const vuUser = `vu_${__VU}_${data.testUser}`;
    const vuPass = data.testPass;

    // ADIM 0: Kayıt (Sadece sanal kullanıcının ilk iterasyonunda çalışır)
    // Böylece test boyunca gereksiz 400 Bad Request (Kullanıcı zaten var) hatalarından kurtuluruz.
    if (__ITER === 0) {
        let regRes = http.post(`${BASE_URL}/register`, JSON.stringify({
            username: vuUser,
            password: vuPass
        }), { headers: { 'Content-Type': 'application/json' }, tags: { name: 'Register' } });
        
        check(regRes, {
            '[Register] Başarılı (201/200)': (r) => r.status === 201 || r.status === 200,
        });
    }

    // ADIM 1: Login Olma
    const loginRes = http.post(`${BASE_URL}/login`, JSON.stringify({
        username: vuUser,
        password: vuPass
    }), { headers: { 'Content-Type': 'application/json' } });

    check(loginRes, {
        '[Login] Başarılı (200)': (r) => r.status === 200,
    });

    let token = null;
    try {
        token = loginRes.json('token');
    } catch(e) {}
    if(!token) return;

    // Gerçekçi Bekleme: Kullanıcı login oldu, anasayfada geziniyor
    sleep(0.5);

    // ADIM 2: Tüm Etkinlikleri Listele (GET /events)
    let getEventsRes = http.get(`${BASE_URL}/events`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });

    check(getEventsRes, {
        '[Events] Listeleme başarılı (200)': (r) => r.status === 200,
    });

    sleep(0.5);

    // ADIM 3: İlgili Etkinliğin Detayına Bak (GET /events/${id})
    let getEventDetailRes = http.get(`${BASE_URL}/events/${data.eventId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });

    check(getEventDetailRes, {
        '[Events] Detay okuma başarılı (200)': (r) => r.status === 200,
    });

    sleep(1); // Kullanıcı etkinliği inceliyor

    // ADIM 4: Bilet Rezervasyonu Yap (POST /bookings)
    const bookingPayload = JSON.stringify({
        event_id: data.eventId,
        user_id: vuUser, // Bu VU'nun kendi kullanıcı adı
        seats: 1
    });

    const bookingRes = http.post(`${BASE_URL}/bookings`, bookingPayload, {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        },
    });

    check(bookingRes, {
        '[Bookings] Rezervasyon başarılı (201)': (r) => r.status === 201,
    });

    sleep(0.5);

    // ADIM 5: Kendi biletlerimi kontrol et (GET /bookings?user_id=...)
    let getBookingsRes = http.get(`${BASE_URL}/bookings?user_id=${vuUser}`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });

    check(getBookingsRes, {
        '[Bookings] Biletleri listeleme başarılı (200)': (r) => r.status === 200,
    });

    sleep(1);
}
