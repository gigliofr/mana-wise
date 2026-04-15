param(
  [string]$ApiBaseUrl = "http://localhost:8080/api/v1",
  [Parameter(Mandatory = $true)]
  [string]$Token,
  [ValidateRange(1, 50)]
  [int]$Calls = 10,
  [string]$AdminSecret = ""
)

$ErrorActionPreference = "Stop"

function Invoke-Analyze {
  param(
    [string]$BaseUrl,
    [string]$BearerToken,
    [string]$Decklist,
    [string]$Format,
    [string]$Locale
  )

  $headers = @{
    Authorization = "Bearer $BearerToken"
    "Content-Type" = "application/json"
    "Accept-Language" = $Locale
  }

  $body = @{
    decklist = $Decklist
    format   = $Format
    locale   = $Locale
  } | ConvertTo-Json

  $response = Invoke-RestMethod -Method Post -Uri "$BaseUrl/analyze" -Headers $headers -Body $body
  return $response
}

function Print-Section {
  param([string]$Title)
  Write-Host ""
  Write-Host "=== $Title ==="
}

$sampleDecks = @(
  @{
    format = "modern"
    locale = "en"
    decklist = @"
4 Lightning Bolt
4 Monastery Swiftspear
4 Lava Spike
4 Rift Bolt
4 Boros Charm
4 Skewer the Critics
20 Mountain
"@
  },
  @{
    format = "commander"
    locale = "it"
    decklist = @"
1 Sol Ring
1 Arcane Signet
1 Command Tower
1 Swords to Plowshares
1 Counterspell
1 Cyclonic Rift
1 Rhystic Study
1 Smothering Tithe
1 Esper Sentinel
1 Flooded Strand
1 Island
1 Plains
1 Swamp
"@
  }
)

$rows = New-Object System.Collections.Generic.List[object]
$hardFailures = 0

Print-Section "AI rollout smoke check"
Write-Host "API base: $ApiBaseUrl"
Write-Host "Calls: $Calls"

for ($i = 0; $i -lt $Calls; $i++) {
  $sample = $sampleDecks[$i % $sampleDecks.Count]
  try {
    $result = Invoke-Analyze -BaseUrl $ApiBaseUrl -BearerToken $Token -Decklist $sample.decklist -Format $sample.format -Locale $sample.locale

    $source = [string]($result.ai_source)
    $aiError = [string]($result.ai_error)
    $latency = [int64]($result.latency_ms)
    $hasSource = -not [string]::IsNullOrWhiteSpace($source)

    if (-not $hasSource) {
      $hardFailures++
    }

    $rows.Add([pscustomobject]@{
      Call      = $i + 1
      Format    = $sample.format
      Locale    = $sample.locale
      Source    = if ($hasSource) { $source } else { "<missing>" }
      HasError  = -not [string]::IsNullOrWhiteSpace($aiError)
      LatencyMs = $latency
      Status    = if ($hasSource) { "ok" } else { "missing_ai_source" }
    }) | Out-Null
  }
  catch {
    $hardFailures++
    $rows.Add([pscustomobject]@{
      Call      = $i + 1
      Format    = $sample.format
      Locale    = $sample.locale
      Source    = "<request_failed>"
      HasError  = $true
      LatencyMs = 0
      Status    = "request_failed"
    }) | Out-Null
  }
}

$rows | Format-Table -AutoSize

$sourceCounts = $rows | Group-Object Source | Sort-Object Count -Descending
$fallbackCount = ($rows | Where-Object { $_.Source -eq "internal_rules" -and $_.HasError }).Count
$avgLatency = [Math]::Round((($rows | Measure-Object -Property LatencyMs -Average).Average), 2)

Print-Section "Summary"
Write-Host "Hard failures: $hardFailures"
Write-Host "Fallback-active calls (internal_rules + ai_error): $fallbackCount"
Write-Host "Average latency (ms): $avgLatency"
Write-Host "Source distribution:"
$sourceCounts | ForEach-Object { Write-Host "- $($_.Name): $($_.Count)" }

if (-not [string]::IsNullOrWhiteSpace($AdminSecret)) {
  Print-Section "Admin metrics snapshot"
  try {
    $metrics = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/admin/metrics/funnel" -Headers @{ "X-Admin-Secret" = $AdminSecret }
    Write-Host "analysis_fallbacks: $($metrics.analysis_fallbacks)"
    Write-Host "forwarding_errors: $($metrics.forwarding_errors)"
    Write-Host "analysis_by_ai_source:"
    if ($metrics.analysis_by_ai_source -ne $null) {
      $metrics.analysis_by_ai_source.PSObject.Properties | ForEach-Object {
        Write-Host "- $($_.Name): $($_.Value)"
      }
    }
  }
  catch {
    Write-Host "failed to fetch admin metrics: $($_.Exception.Message)"
    $hardFailures++
  }
}

if ($hardFailures -gt 0) {
  Write-Error "AI rollout smoke check failed with $hardFailures hard failures"
  exit 1
}

Write-Host ""
Write-Host "AI rollout smoke check passed"
