$ErrorActionPreference = 'Stop'

$projectDir = Split-Path -Parent $PSScriptRoot
Push-Location $projectDir
try {
    & docker compose build @args
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose build failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}
