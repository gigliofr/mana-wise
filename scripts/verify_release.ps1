param(
  [string]$ApiBaseUrl = "http://localhost:8080/api/v1",
  [string]$Token = "",
  [string]$AdminSecret = "",
  [switch]$SkipBackend,
  [switch]$SkipFrontend,
  [switch]$SkipAISmoke
)

$ErrorActionPreference = "Stop"

function Run-Step {
  param(
    [string]$Name,
    [scriptblock]$Action
  )

  Write-Host ""
  Write-Host "=== $Name ==="
  & $Action
}

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
  if (-not $SkipBackend) {
    Run-Step -Name "Backend tests" -Action {
      go test ./...
    }
  }

  if (-not $SkipFrontend) {
    Run-Step -Name "Frontend tests" -Action {
      Push-Location web
      try {
        npm test
      }
      finally {
        Pop-Location
      }
    }

    Run-Step -Name "Frontend build" -Action {
      Push-Location web
      try {
        npm run build
      }
      finally {
        Pop-Location
      }
    }
  }

  if (-not $SkipAISmoke) {
    if ([string]::IsNullOrWhiteSpace($Token)) {
      Write-Host ""
      Write-Host "Skipping AI smoke check: provide -Token to enable it."
    }
    else {
      Run-Step -Name "AI rollout smoke check" -Action {
        $scriptPath = Join-Path $PSScriptRoot "verify_ai_rollout.ps1"
        if ([string]::IsNullOrWhiteSpace($AdminSecret)) {
          & $scriptPath -ApiBaseUrl $ApiBaseUrl -Token $Token -Calls 10
        }
        else {
          & $scriptPath -ApiBaseUrl $ApiBaseUrl -Token $Token -Calls 10 -AdminSecret $AdminSecret
        }
      }
    }
  }

  Write-Host ""
  Write-Host "Release verification completed successfully."
}
finally {
  Pop-Location
}
