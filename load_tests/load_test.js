import http from 'k6/http';
import { check, sleep } from 'k6';

// Proje İsterleri: 50, 100, 200, 500 eş zamanlı istek senaryosu
export let options = {
    stages: [
        { duration: '30s', target: 50 },
        { duration: '30s', target: 100 },
        { duration: '30s', target: 200 },
        { duration: '30s', target: 500 },
        { duration: '30s', target: 0 }, // soğuma
    ],
    thresholds: {
        http_req_duration: ['p(95)<1000'], // 95% of requests should be < 1000ms
        http_req_failed: ['rate<0.05'], // < 5% error rate
    },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

// Setup her test senaryosunun başında 1 kez çalışır (VU'lardan bağımsız)
export function setup() {
    const timestamp = Date.now();
    const adminUser = `admin_${timestamp}`;
    const adminPass = "12345";
    
    // 1. Yeni bir Admin Kaydet ve Giriş Yap
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

    // 2. Özel bir Test Etkinliği Oluştur
    const eventName = `load_test_event_${timestamp}`;
    http.post(`${BASE_URL}/events`, JSON.stringify({
        name: eventName,
        location: "K6 Load Test Arena",
        capacity: 1000000 // Yük testinde kapasite sınırı olmaması için büyük bir değer
    }), {
        headers: { 
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${adminToken}`
        }
    });

    // Event ID'sini tespit et (GetAll üzerinden)
    const eventsRes = http.get(`${BASE_URL}/events`, {
        headers: { 'Authorization': `Bearer ${adminToken}` }
    });
    
    const events = eventsRes.json();
    let eventId = null;
    
    if (events && Array.isArray(events)) {
        const matchedEvent = events.find(e => e.name === eventName);
        if (matchedEvent) eventId = matchedEvent.id;
    }

    // 3. Test yapacak standart kullanıcıyı kaydet
    const testUser = `loadtestuser_${timestamp}`;
    const testPass = "pass123";
    http.post(`${BASE_URL}/register`, JSON.stringify({
        username: testUser,
        password: testPass
    }), { headers: { 'Content-Type': 'application/json' } });

    return { eventId: eventId, testUser: testUser, testPass: testPass };
}

export default function (data) {
    if (!data.eventId) {
        console.error("Event ID setup asamasinda alinamadi.");
        return;
    }

    // Kullanıcı ile Login Ol
    const loginRes = http.post(`${BASE_URL}/login`, JSON.stringify({
        username: data.testUser,
        password: data.testPass
    }), { headers: { 'Content-Type': 'application/json' } });

    let token = null;
    try {
        token = loginRes.json('token');
    } catch(e) {}
    
    if(!token) return;

    // Bilet Rezervasyonu Gönder
    const bookingPayload = JSON.stringify({
        event_id: data.eventId,
        user_id: data.testUser,
        seats: 1
    });

    const bookingRes = http.post(`${BASE_URL}/bookings`, bookingPayload, {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        },
    });

    check(bookingRes, {
        'Bilet oluşturuldu (201)': (r) => r.status === 201,
        'Sunucu hatası yok': (r) => r.status !== 500 && r.status !== 504 && r.status !== 400,
    });

    sleep(1);
}
