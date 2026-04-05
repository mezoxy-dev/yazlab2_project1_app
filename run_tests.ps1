param(
    [Parameter(Mandatory=$false)][string]$TestName = "load"
)

$ValidTests = @("load", "stress", "spike")
if ($ValidTests -notcontains $TestName) {
    Write-Host "Gecersiz test adi: $TestName" -ForegroundColor Red
    Write-Host "Gecerli testler: load, stress, spike" -ForegroundColor Yellow
    exit 1
}

$ScriptFile = "/k6-scripts/${TestName}_test.js"
Write-Host ">>> $TestName load testi baslatiliyor: $ScriptFile" -ForegroundColor Cyan

# K6 docker icerisinde calistiriliyor
docker compose --profile testing run --rm k6 run $ScriptFile

Write-Host ">>> Test tamamlandi! Dashboard icin http://localhost:3001 adresini kontrol edin." -ForegroundColor Green
