param(
    [Parameter(Mandatory=$false)][string]$TestName = "load"
)

$ValidTests = @("load", "stress", "spike")
if ($ValidTests -notcontains $TestName) {
    Write-Host "Geçersiz test adı: $TestName" -ForegroundColor Red
    Write-Host "Geçerli testler: load, stress, spike" -ForegroundColor Yellow
    exit 1
}

$ScriptFile = "/k6-scripts/${TestName}_test.js"
Write-Host "🔥 $TestName test başlatılıyor: $ScriptFile" -ForegroundColor Cyan

# Docker Compose kullanarak k6'yı geçici bir container olarak çalıştırır ve influxdb'ye yazar
docker compose --profile testing run --rm k6 run $ScriptFile

Write-Host "✅ Test tamamlandı! Grafana paneli için tarayıcınızda http://localhost:3001 adresine gidin." -ForegroundColor Green
