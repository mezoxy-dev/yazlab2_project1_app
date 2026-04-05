# EventHub Proje Temizlik Betiği
# Bu script tüm mikroservislerin veritabanlarını ve log tablolarını sıfırlar.

Write-Host "`n🚀 EventHub Temizlik Operasyonu Başlıyor..." -ForegroundColor Cyan

# 1. Auth DB Temizle (Kullanıcılar)
Write-Host "🧹 Auth veritabanı temizleniyor..." -ForegroundColor Yellow
docker exec yazlab2_project1_app-db-auth-1 mongosh authdb --eval "db.users.deleteMany({});" | Out-Null

# 2. Event DB Temizle (Etkinlikler)
Write-Host "🧹 Event veritabanı temizleniyor..." -ForegroundColor Yellow
docker exec yazlab2_project1_app-db-event-1 mongosh eventdb --eval "db.events.deleteMany({});" | Out-Null

# 3. Reservation DB Temizle (Biletler)
Write-Host "🧹 Reservation veritabanı temizleniyor..." -ForegroundColor Yellow
docker exec yazlab2_project1_app-db-booking-1 mongosh reservationdb --eval "db.bookings.deleteMany({});" | Out-Null

# 4. Dispatcher Log DB Temizle (Admin UI Logları)
Write-Host "🧹 Dispatcher Log veritabanı temizleniyor..." -ForegroundColor Yellow
docker exec yazlab2_project1_app-db-dispatcher-1 mongosh dispatcher_logs --eval "db.traffic.deleteMany({});" | Out-Null

# Ekstra: Eskiden Auth DB içinde kalan logları da temizle (Yapılandırma hatası düzeltildi ama eskiler kalmış olabilir)
docker exec yazlab2_project1_app-db-auth-1 mongosh dispatcher_logs --eval "db.traffic.deleteMany({});" | Out-Null

Write-Host "`n✨ Sistem Pırıl Pırıl! Tüm Test Verileri ve Loglar Silindi." -ForegroundColor Green
Write-Host "👉 Admin UI'ı yenileyip sıfırdan pürüzsüz bir başlangıç yapabilirsiniz.`n" -ForegroundColor Cyan
