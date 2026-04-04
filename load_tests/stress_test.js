import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    stages: [
        { duration: '30s', target: 50 },  // Normal yük
        { duration: '1m', target: 300 },  // Kırılmayı bulmak için agresif artış
        { duration: '30s', target: 0 },   // Soğuma
    ],
};

const BASE_URL = 'http://localhost:8080';

export default function () {
    const loginPayload = JSON.stringify({
        username: "standart_ahmet",
        password: "123"
    });

    const loginRes = http.post(`${BASE_URL}/login`, loginPayload, {
        headers: { 'Content-Type': 'application/json' },
    });

    check(loginRes, { 'Login basarili (200)': (r) => r.status === 200 });

    let token;
    try {
        token = loginRes.json('token');
    } catch (e) {
        return;
    }

    sleep(Math.random() * 2 + 1);

    const bookingPayload = JSON.stringify({
        event_id: "69d113c309699be7d5784d82",
        user_id: "standart_ahmet",
        seats: 1
    });

    const bookingRes = http.post(`${BASE_URL}/bookings`, bookingPayload, {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        },
    });

    check(bookingRes, {
        'Bilet olusturuldu (201)': (r) => r.status === 201,
        'Sunucu hatasi yok': (r) => r.status !== 500 && r.status !== 504,
    });
}