param(
  [Parameter(Mandatory=$true)]
  [string]$Email,
  [ValidateSet('free','pro')]
  [string]$Plan = 'pro'
)

$ErrorActionPreference = 'Stop'

$goScript = @"
package main

import (
  "context"
  "fmt"
  "os"
  "time"

  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/mongo"
  "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
  email := os.Getenv("TARGET_EMAIL")
  plan := os.Getenv("TARGET_PLAN")
  uri := os.Getenv("MONGODB_URI")
  dbName := os.Getenv("MONGODB_DB_NAME")
  if uri == "" { uri = "mongodb://localhost:27017" }
  if dbName == "" { dbName = "manawise" }

  ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
  defer cancel()

  client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
  if err != nil { panic(err) }
  defer client.Disconnect(ctx)

  res, err := client.Database(dbName).Collection("users").UpdateOne(
    ctx,
    bson.M{"email": email},
    bson.M{"$set": bson.M{"plan": plan, "updated_at": time.Now().UTC()}},
  )
  if err != nil { panic(err) }

  if res.MatchedCount == 0 {
    fmt.Printf("no user found for %s\n", email)
    os.Exit(1)
  }
  fmt.Printf("updated %s to plan=%s (matched=%d modified=%d)\n", email, plan, res.MatchedCount, res.ModifiedCount)
}
"@

if (Test-Path .env) {
  Get-Content .env | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
    $parts = $_ -split '=',2
    if ($parts.Length -eq 2) {
      [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), 'Process')
    }
  }
}

[Environment]::SetEnvironmentVariable('TARGET_EMAIL', $Email, 'Process')
[Environment]::SetEnvironmentVariable('TARGET_PLAN', $Plan, 'Process')

$tmp = Join-Path $env:TEMP 'mw_set_user_plan.go'
Set-Content -Path $tmp -Value $goScript
try {
  go run $tmp
}
finally {
  if (Test-Path $tmp) { Remove-Item $tmp -Force }
}
