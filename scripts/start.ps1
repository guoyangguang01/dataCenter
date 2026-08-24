$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
$BinDir = Join-Path $Root "bin"
$LogDir = Join-Path $Root "logs"
$PidDir = Join-Path $Root "pids"

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
New-Item -ItemType Directory -Force -Path $PidDir | Out-Null

if (-not (Test-Path "$BinDir\console.exe")) {
    Write-Host "[ERROR] bin/console.exe not found, run 'make build' first" -ForegroundColor Red
    exit 1
}

Write-Host "========================================"
Write-Host "  IoT Data Center - Starting Services"
Write-Host "========================================"
Write-Host ""

Write-Host "[INFRA] Starting infrastructure..." -ForegroundColor Cyan
$composeFile = Join-Path $Root "deploy\docker-compose.yml"
if (Test-Path $composeFile) {
    docker compose -f $composeFile up -d 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[INFRA] Infrastructure started" -ForegroundColor Green
    } else {
        Write-Host "[INFRA] Docker Compose failed, skipped" -ForegroundColor Yellow
    }
} else {
    Write-Host "[INFRA] docker-compose.yml not found, skipped" -ForegroundColor Yellow
}
Start-Sleep -Seconds 5

$services = @("device-service", "alert-service", "rule-engine", "timeseries-writer", "console")

# 服务间间隔，等基础设施就绪
$delayBetweenServices = 2

foreach ($svc in $services) {
    $svcPidFile = Join-Path $PidDir "$svc.pid"
    $logFile = Join-Path $LogDir "$svc.log"
    $exe = Join-Path $BinDir "$svc.exe"

    if (Test-Path $svcPidFile) {
        $oldPid = Get-Content $svcPidFile -ErrorAction SilentlyContinue
        if ($oldPid) {
            $proc = Get-Process -Id $oldPid -ErrorAction SilentlyContinue
            if ($proc) {
                Write-Host "[$svc] Already running (PID: $oldPid)" -ForegroundColor Yellow
                continue
            }
        }
    }

    Write-Host "[$svc] Starting..." -ForegroundColor Cyan
    $stdoutLog = Join-Path $LogDir "$svc.stdout.log"
    $stderrLog = Join-Path $LogDir "$svc.stderr.log"
    $proc = Start-Process -FilePath $exe -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog -PassThru -WindowStyle Hidden
    if ($proc) {
        $proc.Id | Out-File -FilePath $svcPidFile -Encoding ascii -NoNewline
        Write-Host "[$svc] Started (PID: $($proc.Id))" -ForegroundColor Green
    } else {
        Write-Host "[$svc] Failed to start" -ForegroundColor Red
    }
    Start-Sleep -Seconds $delayBetweenServices
}

Write-Host ""
Write-Host "========================================"
Write-Host "  All services started!" -ForegroundColor Green
Write-Host "  Gateways are managed via web console"
Write-Host "========================================"
Write-Host ""

Write-Host "Service Status:" -ForegroundColor Cyan
Write-Host "----------------------------------------"
foreach ($svc in $services) {
    $svcPidFile = Join-Path $PidDir "$svc.pid"
    if (Test-Path $svcPidFile) {
        $svcPid = Get-Content $svcPidFile -ErrorAction SilentlyContinue
        if ($svcPid) {
            $proc = Get-Process -Id $svcPid -ErrorAction SilentlyContinue
            if ($proc) {
                Write-Host "  [RUNNING] $svc  (PID: $svcPid)" -ForegroundColor Green
            } else {
                Write-Host "  [STOPPED] $svc" -ForegroundColor Red
            }
        } else {
            Write-Host "  [STOPPED] $svc" -ForegroundColor Red
        }
    } else {
        Write-Host "  [STOPPED] $svc" -ForegroundColor Red
    }
}
Write-Host "----------------------------------------"
