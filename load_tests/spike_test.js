import http from 'k6/http';
import { check, sleep } from 'k6';

// Spike testi: Sistemin aniden gelen devasa yüklere karşı dayanıklılığını ölçer
export let options = {
    stages: [
        { duration: '10s', target: 50 },   // Hızlı ısınma
        { duration: '1m', target: 1000 },  // Anlık ÇOK yüksek trafik (Spike - Bilet satışı başlangıcı hissi)
        { duration: '30s', target: 0 },    // Hızlı düşüş ve soğuma
    ],
    thresholds: {
        http_req_failed: ['rate<0.10'], // Zorlu şartlarda max %10 hata toleransı
    },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export function setup() {
    const timestamp = Date.now();
    const adminUser = `admin_${timestamp}`;
    const adminPass = "12345";
    
    // Admin Kayıt ve Login
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

    // Test Event oluştur (Kapasitesi kasıtlı olarak kısıtlı olabilir, ancak performansı ölçmek için açık bıraktık)
    const eventName = `spike_test_event_${timestamp}`;
    http.post(`${BASE_URL}/events`, JSON.stringify({
        name: eventName,
        location: "Spike Arena",
        capacity: 50000 
    }), {
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${adminToken}` }
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

    const testUser = `spikeuser_${timestamp}`;
    const testPass = "pass123";
    http.post(`${BASE_URL}/register`, JSON.stringify({
        username: testUser,
        password: testPass
    }), { headers: { 'Content-Type': 'application/json' } });

    return { eventId: eventId, testUser: testUser, testPass: testPass };
}

export default function (data) {
    if (!data.eventId) return;

    // Login
    const loginRes = http.post(`${BASE_URL}/login`, JSON.stringify({
        username: data.testUser,
        password: data.testPass
    }), { headers: { 'Content-Type': 'application/json' } });

    let token = null;
    try {
        token = loginRes.json('token');
    } catch(e) {}
    if(!token) return;

    // Booking Request
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
        'Bilet Başarılı (201)': (r) => r.status === 201,
        'Capacity Hatasi Olabilir (400)': (r) => r.status === 400 || r.status === 201,
        'Sistem Çökmedi': (r) => r.status !== 500 && r.status !== 502 && r.status !== 504,
    });

    sleep(0.5); // Spike testi olduğu için request hızı daha agresif
}
