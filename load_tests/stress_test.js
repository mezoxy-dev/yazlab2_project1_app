import http from 'k6/http';
import { check, sleep } from 'k6';

// Stress Test: Sistemin maksimum ne kadar yük kaldırabildiğini ve nerede kırıldığını bulur.
export let options = {
    stages: [
        { duration: '30s', target: 50 },  // Normal yük
        { duration: '1m', target: 300 },  // Kırılmayı bulmak için agresif artış
        { duration: '1m', target: 600 },  // Limitleri zorla
        { duration: '30s', target: 0 },   // Soğuma
    ],
    thresholds: {
        http_req_failed: ['rate<0.05'], // Limitler zorlansa bile %5'ten az hata beklenir.
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

    // Test Event oluştur
    const eventName = `stress_test_event_${timestamp}`;
    http.post(`${BASE_URL}/events`, JSON.stringify({
        name: eventName,
        location: "Stress Arena",
        capacity: 1000000 
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

    const testUser = `stressuser_${timestamp}`;
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
        'Sistem Sağlıklı': (r) => r.status !== 500 && r.status !== 502 && r.status !== 504,
    });

    sleep(1);
}