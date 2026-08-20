# dev.ps1 — Windows-native equivalent of the Makefile (make is Unix-only).
# Usage examples:
#   pwsh -File scripts\dev.ps1 build          (or: powershell -File …)
#   pwsh -File scripts\dev.ps1 build-nocgo
#   pwsh -File scripts\dev.ps1 web
#   pwsh -File scripts\dev.ps1 run
#   pwsh -File scripts\dev.ps1 test
#   pwsh -File scripts\dev.ps1 check
#   pwsh -File scripts\dev.ps1 midi-list
[CmdletBinding()]
param(
  [Parameter(Position = 0)][ValidateSet("help","build","build-nocgo","build-windows","web","run","test","vet","check","midi-list")]
  [string]$Task = "help"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$mod = "github.com/Qreepex/show-mapper"
try { $ver = (git describe --tags --always --dirty 2>$null) -replace '^v','' } catch { $ver = "dev" }
if (-not $ver) { $ver = "dev" }
try { $commit = (git rev-parse HEAD 2>$null).Trim() } catch { $commit = "none" }
$date = [DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")
$ldflags = "-s -w -X $mod/internal/version.Version=$ver -X $mod/internal/version.Commit=$commit -X $mod/internal/version.Date=$date"

function Invoke-Checked($exe, $argv) {
  & $exe @argv
  if ($LASTEXITCODE -ne 0) { throw "$exe $argv failed ($LASTEXITCODE)" }
}

function Test-Cpp { return $null -ne (Get-Command g++ -ErrorAction SilentlyContinue) }

New-Item -ItemType Directory -Force bin | Out-Null

switch ($Task) {
  "help" {
    Write-Host "tasks: build | build-nocgo | build-windows | web | run | test | vet | check | midi-list"
    Write-Host "notes: build* need a C++ toolchain (winget install -e --id BrechtSanders.WinLibs.POSIX.UCRT)"
  }
  "test" { Invoke-Checked go @("test","./...") }
  "vet"  { Invoke-Checked go @("vet","./...") }
  "web"  { Set-Location web; Invoke-Checked cmd @("/c","npm ci"); Invoke-Checked cmd @("/c","npm run build"); Set-Location $root }
  "build-nocgo" {
    $env:CGO_ENABLED = "0"
    Invoke-Checked go @("build","-trimpath","-ldflags",$ldflags,"-o","bin\show-mapper-nomidi.exe",".\cmd\show-mapper")
    Write-Host "built bin\show-mapper-nomidi.exe (no MIDI)"
  }
  { $_ -in "build","build-windows" } {
    if (-not (Test-Cpp)) {
      throw "no C++ toolchain. Install: winget install -e --id BrechtSanders.WinLibs.POSIX.UCRT  (then open a NEW shell)"
    }
    $env:CGO_ENABLED = "1"
    Invoke-Checked go @("build","-trimpath","-ldflags",("$ldflags -extldflags=-static"),"-o","bin\show-mapper.exe",".\cmd\show-mapper")
    Write-Host "built bin\show-mapper.exe ($ver, MIDI enabled, static runtime)"
  }
  "run" {
    if (Test-Cpp) { $env:CGO_ENABLED = "1" } else {
      $env:CGO_ENABLED = "0"
      Write-Host "no C++ toolchain -> no-MIDI build; use the Surface tab at http://127.0.0.1:8484/surface"
    }
    Invoke-Checked go @("run",".\cmd\show-mapper","serve")
  }
  "midi-list" { Invoke-Checked go @("run","./cmd/show-mapper","midi","list") }
  "check" {
    & $PSCommandPath vet
    & $PSCommandPath test
    & $PSCommandPath web
    Write-Host "check done (svelte-check inside web; golangci-lint optional: make lint)"
  }
}
