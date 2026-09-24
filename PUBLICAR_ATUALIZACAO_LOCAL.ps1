[CmdletBinding()]
param(
    [switch]$Build,
    [string]$ExePath = "",
    [switch]$SkipCleanCheck
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$DesktopDir = Join-Path $Root "desktop"
$AppGo = Join-Path $DesktopDir "app.go"
$NotesPath = Join-Path $DesktopDir "UPDATE_NOTES.txt"
$Endpoint = "https://igrxqbroklfwujcwbiwh.supabase.co/functions/v1/via-verde-desktop-update"

function Fail([string]$Message) {
    throw $Message
}

function Get-GitHubToken {
    if (-not [string]::IsNullOrWhiteSpace($env:VIAVERDE_GITHUB_TOKEN)) {
        return $env:VIAVERDE_GITHUB_TOKEN.Trim()
    }

    $gh = Get-Command gh -ErrorAction SilentlyContinue
    if ($gh) {
        $token = (& gh auth token 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($token)) {
            return $token
        }
    }

    Write-Host ""
    Write-Host "Autenticacao necessaria para publicar." -ForegroundColor Yellow
    Write-Host "Cole um token pessoal do GitHub da conta pedropapao."
    Write-Host "O token sera usado somente nesta execucao e nao sera salvo pelo script."
    $secure = Read-Host "Token GitHub" -AsSecureString
    $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try {
        return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr)
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)
    }
}

if (-not (Test-Path -LiteralPath $AppGo)) {
    Fail "desktop/app.go nao encontrado. Execute este script a partir da pasta do projeto ViaVerdeCAR."
}

$appContent = Get-Content -LiteralPath $AppGo -Raw
if ($appContent -notmatch 'const AppVersion = "([^"]+)"') {
    Fail "AppVersion nao localizado em desktop/app.go."
}
$Version = $Matches[1].Trim()

$git = Get-Command git -ErrorAction SilentlyContinue
if (-not $git) {
    Fail "Git nao encontrado no Windows."
}

$CommitSha = (& git -C $Root rev-parse HEAD | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $CommitSha -notmatch '^[a-fA-F0-9]{40}$') {
    Fail "Nao foi possivel identificar o commit atual do projeto."
}

if (-not $SkipCleanCheck) {
    $status = (& git -C $Root status --porcelain | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        Fail "Nao foi possivel conferir o estado do repositorio."
    }

    if (-not [string]::IsNullOrWhiteSpace($status)) {
        $lines = @($status -split "\r?\n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
        $paths = @($lines | ForEach-Object {
            if ($_.Length -ge 4) { $_.Substring(3).Trim() } else { $_.Trim() }
        })
        $knownGenerated = @("desktop/go.mod", "desktop/go.sum")
        $onlyKnownGenerated = ($paths.Count -gt 0) -and (@($paths | Where-Object { $_ -notin $knownGenerated }).Count -eq 0)

        if ($onlyKnownGenerated) {
            Write-Host ""
            Write-Host "Limpando arquivos de dependencias gerados pela tentativa anterior..." -ForegroundColor Yellow
            & git -C $Root restore --worktree -- desktop/go.mod 2>$null
            if ($LASTEXITCODE -ne 0) {
                Fail "Nao foi possivel restaurar desktop/go.mod."
            }
            $GeneratedGoSum = Join-Path $DesktopDir "go.sum"
            if (Test-Path -LiteralPath $GeneratedGoSum) {
                Remove-Item -LiteralPath $GeneratedGoSum -Force -ErrorAction SilentlyContinue
            }
            $status = (& git -C $Root status --porcelain | Out-String).Trim()
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($status)) {
        Write-Host ""
        Write-Host "Arquivos alterados ainda nao foram commitados:" -ForegroundColor Yellow
        Write-Host $status
        Fail "Por seguranca, a publicacao foi cancelada. Salve/commite as alteracoes antes de publicar."
    }
}

if ($Build) {
    Write-Host ""
    Write-Host "1/5 - Preparando dependencias e executando testes Go..." -ForegroundColor Cyan
    Push-Location $DesktopDir
    try {
        & go mod download
        if ($LASTEXITCODE -ne 0) {
            Fail "Nao foi possivel baixar as dependencias Go. Nenhuma atualizacao foi publicada."
        }

        & go test -mod=mod ./...
        if ($LASTEXITCODE -ne 0) {
            Fail "Os testes falharam. Nenhuma atualizacao foi publicada."
        }

        $wails = Get-Command wails -ErrorAction SilentlyContinue
        if (-not $wails) {
            Write-Host "Wails nao encontrado. Instalando Wails v2.15.0 localmente..." -ForegroundColor Yellow
            & go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
            if ($LASTEXITCODE -ne 0) {
                Fail "Nao foi possivel instalar o Wails."
            }
        }

        Write-Host "2/5 - Compilando ViaVerdeCAR $Version para Windows x64..." -ForegroundColor Cyan
        & wails build -platform windows/amd64 -clean -webview2 download -o ViaVerdeCAR.exe
        if ($LASTEXITCODE -ne 0) {
            Fail "A compilacao falhou. Nenhuma atualizacao foi publicada."
        }
    }
    finally {
        Pop-Location
        & git -C $Root restore --worktree -- desktop/go.mod 2>$null
        $GeneratedGoSum = Join-Path $DesktopDir "go.sum"
        if (Test-Path -LiteralPath $GeneratedGoSum) {
            Remove-Item -LiteralPath $GeneratedGoSum -Force -ErrorAction SilentlyContinue
        }
    }
}

if ([string]::IsNullOrWhiteSpace($ExePath)) {
    $ExePath = Join-Path $DesktopDir "build\bin\ViaVerdeCAR.exe"
}
$ExePath = [IO.Path]::GetFullPath($ExePath)

if (-not (Test-Path -LiteralPath $ExePath)) {
    Fail "Executavel nao encontrado em: $ExePath"
}

$File = Get-Item -LiteralPath $ExePath
$Size = [int64]$File.Length
if ($Size -lt 1MB -or $Size -gt 50MB) {
    Fail "Tamanho do executavel fora do limite esperado: $Size bytes."
}

$Sha256 = (Get-FileHash -LiteralPath $ExePath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($Sha256 -notmatch '^[a-f0-9]{64}$') {
    Fail "Nao foi possivel calcular o SHA-256 do executavel."
}

$Notes = "Via Verde CAR $Version"
if (Test-Path -LiteralPath $NotesPath) {
    $candidate = (Get-Content -LiteralPath $NotesPath -Raw).Trim()
    if (-not [string]::IsNullOrWhiteSpace($candidate)) {
        $Notes = $candidate
    }
}

Write-Host ""
Write-Host "Versao: $Version"
Write-Host "Commit: $CommitSha"
Write-Host "EXE: $ExePath"
Write-Host "Tamanho: $Size bytes"
Write-Host "SHA-256: $Sha256"

$Token = Get-GitHubToken
if ([string]::IsNullOrWhiteSpace($Token)) {
    Fail "Token GitHub nao informado."
}

$Headers = @{
    "Authorization" = "Bearer $Token"
    "x-publisher-mode" = "local"
    "x-github-sha" = $CommitSha.ToLowerInvariant()
    "x-version" = $Version
    "x-sha256" = $Sha256
    "x-size" = [string]$Size
}

try {
    Write-Host ""
    Write-Host "3/5 - Solicitando canal seguro de upload..." -ForegroundColor Cyan
    $Prepare = Invoke-RestMethod -Method Post -Uri "${Endpoint}?action=prepare" -Headers $Headers -TimeoutSec 60
    if (-not $Prepare.ok -or [string]::IsNullOrWhiteSpace([string]$Prepare.upload_url)) {
        Fail "O servidor nao autorizou o upload."
    }

    Write-Host "4/5 - Enviando executavel ao Supabase..." -ForegroundColor Cyan
    $UploadHeaders = @{
        "cache-control" = "max-age=3600"
        "x-upsert" = "true"
    }
    Invoke-WebRequest -Method Put -Uri ([string]$Prepare.upload_url) -InFile $ExePath -ContentType "application/vnd.microsoft.portable-executable" -Headers $UploadHeaders -TimeoutSec 600 | Out-Null

    $Body = @{ notes = $Notes } | ConvertTo-Json -Compress
    $Finalize = Invoke-RestMethod -Method Post -Uri "${Endpoint}?action=finalize" -Headers $Headers -ContentType "application/json" -Body $Body -TimeoutSec 60

    if (-not $Finalize.ok -or [string]$Finalize.version -ne $Version) {
        Fail "O servidor nao confirmou a publicacao da versao $Version."
    }

    Write-Host "5/5 - Conferindo manifesto publicado..." -ForegroundColor Cyan
    $Manifest = Invoke-RestMethod -Method Get -Uri "${Endpoint}?action=manifest" -TimeoutSec 60

    if ([string]$Manifest.version -ne $Version) {
        Fail "Manifesto retornou versao diferente: $($Manifest.version)"
    }
    if ([string]$Manifest.sha256 -ne $Sha256) {
        Fail "Manifesto retornou SHA-256 diferente do executavel publicado."
    }
    if ([int64]$Manifest.size -ne $Size) {
        Fail "Manifesto retornou tamanho diferente do executavel publicado."
    }

    Write-Host ""
    Write-Host "PUBLICACAO CONCLUIDA COM SUCESSO" -ForegroundColor Green
    Write-Host "ViaVerdeCAR $Version esta no canal estavel."
    Write-Host "Os computadores instalados poderao encontra-la em 'Verificar agora'."
}
finally {
    $Token = $null
    Remove-Variable Token -ErrorAction SilentlyContinue
}
