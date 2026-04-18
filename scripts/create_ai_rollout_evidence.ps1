param(
  [string]$OutputDir = "docs/rollout-evidence",
  [string]$TemplatePath = "docs/AI_ROLLOUT_EVIDENCE_TEMPLATE.md",
  [ValidateSet("ai", "commander-brackets")]
  [string]$TemplateKind = "ai",
  [string]$Owner = "",
  [string]$ReleaseCommit = "",
  [string]$EnvironmentSequence = "dev -> staging -> canary -> prod"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
  if ($PSBoundParameters.ContainsKey("TemplateKind") -or -not $PSBoundParameters.ContainsKey("TemplatePath")) {
    switch ($TemplateKind) {
      "commander-brackets" { $TemplatePath = "docs/COMMANDER_BRACKETS_EVIDENCE_TEMPLATE.md" }
      default { $TemplatePath = "docs/AI_ROLLOUT_EVIDENCE_TEMPLATE.md" }
    }
  }

  if (-not (Test-Path $TemplatePath)) {
    throw "Template not found: $TemplatePath"
  }

  if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
  }

  $ts = Get-Date
  $stamp = $ts.ToString("yyyyMMdd-HHmmss")
  $outPath = Join-Path $OutputDir ("AI_ROLLOUT_EVIDENCE_{0}.md" -f $stamp)

  $template = Get-Content -Path $TemplatePath -Raw

  if ([string]::IsNullOrWhiteSpace($ReleaseCommit)) {
    try {
      $ReleaseCommit = (git rev-parse --short HEAD).Trim()
    }
    catch {
      $ReleaseCommit = ""
    }
  }

  $content = $template
  $content = $content -replace "(?m)^Date:\s*$", ("Date: {0}" -f $ts.ToString("yyyy-MM-dd"))
  if (-not [string]::IsNullOrWhiteSpace($Owner)) {
    $content = $content -replace "(?m)^Owner:\s*$", ("Owner: {0}" -f $Owner)
  }
  $content = $content -replace "(?m)^Environment sequence:\s*.*$", ("Environment sequence: {0}" -f $EnvironmentSequence)
  if (-not [string]::IsNullOrWhiteSpace($ReleaseCommit)) {
    $content = $content -replace "(?m)^Release/commit:\s*$", ("Release/commit: {0}" -f $ReleaseCommit)
  }

  Set-Content -Path $outPath -Value $content -Encoding UTF8

  Write-Host "Created rollout evidence file: $outPath"
  Write-Host "Template kind: $TemplateKind"
  Write-Host "Next step: fill sections and attach CI/metrics artifacts."
}
finally {
  Pop-Location
}
