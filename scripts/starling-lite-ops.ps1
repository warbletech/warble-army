#Requires -Version 5.1
<#
.SYNOPSIS
  Starling Lite ops helper (Windows PowerShell)

.DESCRIPTION
  Build, smoke-test, Railway deploy, optional Cloudflare DNS for custom domain.

.EXAMPLE
  .\scripts\starling-lite-ops.ps1 help
  .\scripts\starling-lite-ops.ps1 smoke
  .\scripts\starling-lite-ops.ps1 smoke -Url https://starling-lite-production.up.railway.app
  .\scripts\starling-lite-ops.ps1 railway-up
  .\scripts\starling-lite-ops.ps1 cloudflare-dns -HostName mcp -Zone warbleoss.org
  .\scripts\starling-lite-ops.ps1 all
#>
[CmdletBinding()]
param(
  [Parameter(Position = 0)]
  [ValidateSet(
    'help', 'build', 'run-local', 'smoke', 'railway-up', 'railway-domain',
    'railway-status', 'cloudflare-dns', 'all'
  )]
  [string]$Command = 'help',

  [string]$Url = $(if ($env:RAILWAY_PUBLIC_URL) { $env:RAILWAY_PUBLIC_URL } else { 'https://starling-lite-production.up.railway.app' }),
  [string]$HostName = 'mcp',
  [string]$Zone = 'warbleoss.org',
  [string]$Target = $(if ($env:RAILWAY_CNAME_TARGET) { $env:RAILWAY_CNAME_TARGET } else { 'p4a4cpbh.up.railway.app' }),
  [switch]$DnsOnly
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

function Write-Log([string]$Message) { Write-Host "==> $Message" }
function Die([string]$Message) { Write-Error $Message; exit 1 }
function Need([string]$Cmd) {
  if (-not (Get-Command $Cmd -ErrorAction SilentlyContinue)) { Die "missing dependency: $Cmd" }
}

function Show-Help {
  @'
Starling Lite ops helper (Windows PowerShell)

  .\scripts\starling-lite-ops.ps1 help
  .\scripts\starling-lite-ops.ps1 build
  .\scripts\starling-lite-ops.ps1 run-local
  .\scripts\starling-lite-ops.ps1 smoke [-Url <https://...>]
  .\scripts\starling-lite-ops.ps1 railway-up
  .\scripts\starling-lite-ops.ps1 railway-domain
  .\scripts\starling-lite-ops.ps1 railway-status
  .\scripts\starling-lite-ops.ps1 cloudflare-dns [-HostName mcp] [-Zone warbleoss.org] [-Target <cname>] [-DnsOnly]
  .\scripts\starling-lite-ops.ps1 all

Env:
  RAILWAY_PUBLIC_URL, RAILWAY_CNAME_TARGET
  CLOUDFLARE_API_TOKEN, CLOUDFLARE_ZONE_ID
'@ | Write-Host
}

function Invoke-Build {
  Need 'go'
  Write-Log 'building starling-lite'
  New-Item -ItemType Directory -Force -Path (Join-Path $Root 'bin') | Out-Null
  Push-Location (Join-Path $Root 'mcp\starling-lite')
  try {
    go build -o (Join-Path $Root 'bin\starling-lite.exe') ./cmd/starling-lite
  } finally { Pop-Location }
  Write-Log "built $(Join-Path $Root 'bin\starling-lite.exe')"
}

function Invoke-RunLocal {
  Invoke-Build
  $env:STARLING_MODE = if ($env:STARLING_MODE) { $env:STARLING_MODE } else { 'readonly' }
  $env:STARLING_TRANSPORT = 'http'
  if (-not $env:PORT) { $env:PORT = '8822' }
  Write-Log "listening on :$($env:PORT) (Ctrl-C to stop)"
  & (Join-Path $Root 'bin\starling-lite.exe') `
    --mode=$env:STARLING_MODE `
    --transport=http `
    --registry=(Join-Path $Root 'mcp\tools\registry.json')
}

function Invoke-Smoke {
  param([string]$BaseUrl = $Url)
  Need 'curl'
  $BaseUrl = $BaseUrl.TrimEnd('/')
  Write-Log "health  $BaseUrl/health"
  $health = curl.exe -fsS --max-time 30 "$BaseUrl/health"
  $health | Write-Host
  Write-Log "tools/list  $BaseUrl/mcp"
  $body = '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
  $raw = curl.exe -fsS --max-time 60 -X POST "$BaseUrl/mcp" `
    -H 'Content-Type: application/json' `
    -H 'MCP-Protocol-Version: 2026-07-28' `
    -d $body
  $obj = $raw | ConvertFrom-Json
  $n = @($obj.result.tools).Count
  Write-Log "tools count: $n"
  if ($n -le 0) { Die 'no tools returned' }
  Write-Log 'smoke OK'
}

function Invoke-RailwayUp {
  Need 'railway'
  railway whoami | Out-Null
  Write-Log "deploying from $Root"
  if (-not (Test-Path (Join-Path $Root 'railway.toml'))) {
    Copy-Item (Join-Path $Root 'deploy\railway\railway.toml') (Join-Path $Root 'railway.toml') -ErrorAction SilentlyContinue
  }
  try {
    railway variable set STARLING_MODE=readonly STARLING_TRANSPORT=http STARLING_REGISTRY=/etc/warble-army/registry.json --skip-deploys
  } catch { }
  railway up -y -d
  Write-Log 'deploy queued — railway status'
  try { railway domain } catch { }
  railway status
}

function Invoke-RailwayDomain {
  Need 'railway'
  railway domain --json
  railway domain list --json
}

function Invoke-RailwayStatus {
  Need 'railway'
  railway status
  try { railway domain list } catch { }
}

function Invoke-CloudflareDns {
  Need 'curl'
  $token = $env:CLOUDFLARE_API_TOKEN
  $zoneId = $env:CLOUDFLARE_ZONE_ID
  if (-not $token) { Die 'set CLOUDFLARE_API_TOKEN' }
  if (-not $zoneId -and $Zone) {
    $zr = curl.exe -fsS "https://api.cloudflare.com/client/v4/zones?name=$Zone" `
      -H "Authorization: Bearer $token" -H 'Content-Type: application/json'
    $zj = $zr | ConvertFrom-Json
    if ($zj.result -and $zj.result.Count -gt 0) { $zoneId = $zj.result[0].id }
  }
  if (-not $zoneId) { Die 'set CLOUDFLARE_ZONE_ID or pass -Zone' }

  $proxied = -not $DnsOnly
  $fqdn = if ($HostName -match '\.') { $HostName } else { "$HostName.$Zone" }
  Write-Log "upsert CNAME $fqdn → $Target (proxied=$proxied)"

  $list = curl.exe -fsS "https://api.cloudflare.com/client/v4/zones/$zoneId/dns_records?type=CNAME&name=$fqdn" `
    -H "Authorization: Bearer $token" -H 'Content-Type: application/json'
  $lj = $list | ConvertFrom-Json
  $recId = $null
  if ($lj.result -and $lj.result.Count -gt 0) { $recId = $lj.result[0].id }

  $payload = @{
    type    = 'CNAME'
    name    = $HostName
    content = $Target
    ttl     = 1
    proxied = $proxied
  } | ConvertTo-Json -Compress

  if ($recId) {
    curl.exe -fsS -X PUT "https://api.cloudflare.com/client/v4/zones/$zoneId/dns_records/$recId" `
      -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d $payload
  } else {
    curl.exe -fsS -X POST "https://api.cloudflare.com/client/v4/zones/$zoneId/dns_records" `
      -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d $payload
  }
  Write-Host ''

  if (Get-Command railway -ErrorAction SilentlyContinue) {
    Write-Log "adding Railway custom domain $fqdn"
    try { railway domain $fqdn } catch { }
  }
  Write-Log "DNS set. Then: .\scripts\starling-lite-ops.ps1 smoke -Url https://$fqdn"
}

switch ($Command) {
  'help' { Show-Help }
  'build' { Invoke-Build }
  'run-local' { Invoke-RunLocal }
  'smoke' { Invoke-Smoke -BaseUrl $Url }
  'railway-up' { Invoke-RailwayUp }
  'railway-domain' { Invoke-RailwayDomain }
  'railway-status' { Invoke-RailwayStatus }
  'cloudflare-dns' { Invoke-CloudflareDns }
  'all' {
    try { Invoke-RailwayStatus } catch { }
    Invoke-Smoke -BaseUrl $Url
  }
  default { Die "unknown command: $Command" }
}
