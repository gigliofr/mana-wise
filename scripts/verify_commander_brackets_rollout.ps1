param(
  [string]$ApiBaseUrl = "http://localhost:8080/api/v1",
  [Parameter(Mandatory = $true)]
  [string]$Token,
  [ValidateRange(1, 50)]
  [int]$Calls = 8,
  [string]$AdminSecret = ""
)

$ErrorActionPreference = "Stop"

function Invoke-JsonPost {
  param(
    [string]$Uri,
    [hashtable]$Headers,
    [hashtable]$Body
  )

  return Invoke-RestMethod -Method Post -Uri $Uri -Headers $Headers -Body ($Body | ConvertTo-Json)
}

$headers = @{
  Authorization = "Bearer $Token"
  "Content-Type" = "application/json"
  "Accept-Language" = "en"
}

$sampleCommanderDecks = @(
  @"
1 Sol Ring
1 Mana Crypt
1 Jeweled Lotus
1 Vampiric Tutor
1 Demonic Tutor
1 Rhystic Study
1 Mystic Remora
1 Dockside Extortionist
1 Underworld Breach
1 Thassa's Oracle
1 Command Tower
1 Island
1 Swamp
1 Mountain
"@,
  @"
1 Arcane Signet
1 Cultivate
1 Kodama's Reach
1 Swords to Plowshares
1 Beast Within
1 Heroic Intervention
1 Command Tower
1 Path of Ancestry
1 Forest
1 Plains
1 Island
"@
)

$rows = New-Object System.Collections.Generic.List[object]
$hardFailures = 0

Write-Host ""
Write-Host "=== Commander bracket rollout smoke check ==="
Write-Host "API base: $ApiBaseUrl"
Write-Host "Calls: $Calls"

for ($i = 0; $i -lt $Calls; $i++) {
  $deck = $sampleCommanderDecks[$i % $sampleCommanderDecks.Count]
  try {
    $response = Invoke-JsonPost -Uri "$ApiBaseUrl/analyze" -Headers $headers -Body @{
      decklist = $deck
      format   = "commander"
      locale   = "en"
    }

    $bracket = $response.commander_bracket
    $bracketNum = [int]($bracket.bracket)
    $bracketLabel = [string]($bracket.label)
    $reason = [string]($bracket.reason)

    $ok = ($bracketNum -ge 1 -and $bracketNum -le 5 -and -not [string]::IsNullOrWhiteSpace($bracketLabel))
    if (-not $ok) {
      $hardFailures++
    }

    $rows.Add([pscustomobject]@{
      Call      = $i + 1
      Bracket   = if ($ok) { $bracketNum } else { "<invalid>" }
      Label     = if ($ok) { $bracketLabel } else { "<missing>" }
      HasReason = -not [string]::IsNullOrWhiteSpace($reason)
      Status    = if ($ok) { "ok" } else { "invalid_commander_bracket" }
    }) | Out-Null
  }
  catch {
    $hardFailures++
    $rows.Add([pscustomobject]@{
      Call      = $i + 1
      Bracket   = "<request_failed>"
      Label     = "<request_failed>"
      HasReason = $false
      Status    = "request_failed"
    }) | Out-Null
  }
}

$rows | Format-Table -AutoSize

$distribution = $rows | Group-Object Bracket | Sort-Object Count -Descending

Write-Host ""
Write-Host "=== Summary ==="
Write-Host "Hard failures: $hardFailures"
Write-Host "Bracket distribution:"
$distribution | ForEach-Object { Write-Host "- $($_.Name): $($_.Count)" }

if (-not [string]::IsNullOrWhiteSpace($AdminSecret)) {
  Write-Host ""
  Write-Host "=== Admin bracket config ==="
  try {
    $cfgResponse = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/admin/commander-brackets" -Headers @{ "X-Admin-Secret" = $AdminSecret }
    $cfg = $cfgResponse.config
    Write-Host "enabled: $($cfg.enabled)"
    Write-Host "decisive_cards count: $(@($cfg.decisive_cards).Count)"
    Write-Host "tutor_keywords count: $(@($cfg.tutor_keywords).Count)"
  }
  catch {
    $hardFailures++
    Write-Host "failed to fetch admin commander brackets config: $($_.Exception.Message)"
  }
}

if ($hardFailures -gt 0) {
  Write-Error "Commander bracket rollout smoke check failed with $hardFailures hard failures"
  exit 1
}

Write-Host ""
Write-Host "Commander bracket rollout smoke check passed"
