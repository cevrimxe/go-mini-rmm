# Go Mini RMM - Windows Agent Installer
# Usage: irm http://SERVER:PORT/install.ps1 | iex

$ServerURL = "{{.ServerURL}}"

# When piped (irm | iex), Read-Host doesn't work.
# Download script to temp and re-run interactively.
$scriptPath = "$env:TEMP\rmm-install.ps1"
$selfContent = $MyInvocation.MyCommand.ScriptBlock.ToString()

if ($selfContent -match 'RERUN_MARKER_OK') {
    # Already re-running from file, continue
} else {
    # Download and re-run
    Invoke-WebRequest -Uri "$ServerURL/install.ps1" -OutFile $scriptPath -UseBasicParsing
    # Add marker so re-run doesn't loop
    Add-Content -Path $scriptPath -Value "`n# RERUN_MARKER_OK"
    # Check admin
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Start-Process powershell -Verb RunAs -ArgumentList "-NoExit -ExecutionPolicy Bypass -File `"$scriptPath`""
    } else {
        & $scriptPath
    }
    return
}

$ErrorActionPreference = "Stop"
$InstallDir = "C:\rmm"
$TaskName = "RMM-Agent"

function Log($msg) { Write-Host "[RMM] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "[RMM] $msg" -ForegroundColor Yellow }
function Err($msg) { Write-Host "[RMM] $msg" -ForegroundColor Red; Read-Host "Enter ile kapat"; exit 1 }

Write-Host ""
Write-Host "=================================" -ForegroundColor Cyan
Write-Host "   Go Mini RMM - Agent Setup     " -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Server: $ServerURL" -ForegroundColor DarkGray
Write-Host ""

# Check admin
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Err "Yonetici olarak calistirin!"
}

# Interactive prompts
$DefaultName = $env:COMPUTERNAME
$AgentName = Read-Host "  Agent ismi [$DefaultName]"
if ([string]::IsNullOrWhiteSpace($AgentName)) { $AgentName = $DefaultName }

$DefaultKey = ($AgentName.ToLower() -replace '[^a-z0-9]', '-').Trim('-')
$AgentKey = Read-Host "  Agent key [$DefaultKey]"
if ([string]::IsNullOrWhiteSpace($AgentKey)) { $AgentKey = $DefaultKey }

Write-Host ""
Log "Server:     $ServerURL"
Log "Agent Key:  $AgentKey"
Log "Agent Name: $AgentName"
Write-Host ""

# Stop existing
$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Warn "Mevcut agent durduruluyor..."
    Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue
}
Get-Process -Name agent -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# Download agent binary
Log "Agent indiriliyor..."
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

try {
    Invoke-WebRequest -Uri "$ServerURL/api/v1/update/download?os=windows&arch=amd64" -OutFile "$InstallDir\agent.exe" -UseBasicParsing
} catch {
    Err "Agent indirilemedi. Server calistigina emin olun: $ServerURL"
}
Log "Agent indirildi: $InstallDir\agent.exe"

# Save config
@{ server = $ServerURL; key = $AgentKey; name = $AgentName } | ConvertTo-Json | Out-File -FilePath "$InstallDir\config.json" -Encoding UTF8
Log "Config kaydedildi: $InstallDir\config.json"

# Create scheduled task (runs at startup, as SYSTEM)
Log "Gorev zamanlayici olusturuluyor..."
$action = New-ScheduledTaskAction -Execute "$InstallDir\agent.exe" -Argument "-server $ServerURL -key $AgentKey" -WorkingDirectory $InstallDir
$trigger = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit (New-TimeSpan -Days 365)
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "Go Mini RMM Agent ($AgentName)" -Force | Out-Null

# Start now
Start-ScheduledTask -TaskName $TaskName
Start-Sleep -Seconds 3

# Verify
$proc = Get-Process -Name agent -ErrorAction SilentlyContinue
if ($proc) {
    $statusText = "Aktif"
    $statusColor = "Green"
} else {
    $statusText = "Basarisiz (log kontrol edin)"
    $statusColor = "Red"
}

Write-Host ""
Write-Host "=================================" -ForegroundColor Cyan
Write-Host "    Kurulum Tamamlandi!          " -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Durum:     $statusText" -ForegroundColor $statusColor
Write-Host "  Dashboard: $ServerURL" -ForegroundColor Cyan
Write-Host "  Agent Key: $AgentKey" -ForegroundColor White
Write-Host ""
Write-Host "  Komutlar:" -ForegroundColor White
Write-Host "    Get-ScheduledTask RMM-Agent    # durum"
Write-Host "    Start-ScheduledTask RMM-Agent  # baslat"
Write-Host "    Stop-ScheduledTask RMM-Agent   # durdur"
Write-Host ""

Read-Host "  Enter ile kapat"
