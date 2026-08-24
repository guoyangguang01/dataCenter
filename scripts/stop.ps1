$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
$PidDir = Join-Path $Root "pids"

Write-Host "========================================"
Write-Host "  IoT Data Center - Stopping Services"
Write-Host "========================================"
Write-Host ""

$services = @("console", "timeseries-writer", "rule-engine", "alert-service", "device-service")

foreach ($svc in $services) {
    $svcPidFile = Join-Path $PidDir "$svc.pid"

    if (-not (Test-Path $svcPidFile)) {
        Write-Host "[$svc] Not running" -ForegroundColor Yellow
        continue
    }

    $svcPid = Get-Content $svcPidFile -ErrorAction SilentlyContinue
    if (-not $svcPid) {
        Write-Host "[$svc] Not running" -ForegroundColor Yellow
        Remove-Item $svcPidFile -ErrorAction SilentlyContinue
        continue
    }

    $proc = Get-Process -Id $svcPid -ErrorAction SilentlyContinue
    if (-not $proc) {
        Write-Host "[$svc] Not running" -ForegroundColor Yellow
        Remove-Item $svcPidFile -ErrorAction SilentlyContinue
        continue
    }

    Write-Host "[$svc] Stopping (PID: $svcPid)..." -ForegroundColor Cyan
    Stop-Process -Id $svcPid -Force -ErrorAction SilentlyContinue
    Write-Host "[$svc] Stopped" -ForegroundColor Green
    Remove-Item $svcPidFile -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "[DOCKER] Stopping infrastructure..." -ForegroundColor Cyan
$composeFile = Join-Path $Root "deploy\docker-compose.yml"
if (Test-Path $composeFile) {
    docker compose -f $composeFile down 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[DOCKER] Infrastructure stopped" -ForegroundColor Green
    } else {
        Write-Host "[DOCKER] Not running or not configured" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "========================================"
Write-Host "  All services stopped" -ForegroundColor Green
Write-Host "========================================"
