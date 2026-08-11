# Octopus 一键构建脚本（Windows / PowerShell）
# 用法:
#   .\build.ps1                          # 版本号默认取当前 git 分支名
#   .\build.ps1 -Version v0.9.28-forked  # 指定版本号
#
# 产物: build\octopus.exe（前端已内嵌，可直接运行）

param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

# 脚本所在目录即项目根目录
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

# 未指定版本时使用当前 git 分支名
if (-not $Version) {
    $branch = git -C $root branch --show-current 2>$null
    if (-not $branch) { $branch = "dev" }
    $Version = $branch
}

Write-Host "==> [1/3] Building frontend..." -ForegroundColor Cyan
Push-Location "$root\web"
try {
    npm run build
    if ($LASTEXITCODE -ne 0) { throw "Frontend build failed (exit code $LASTEXITCODE)" }
} finally {
    Pop-Location
}

if (-not (Test-Path "$root\static\out\index.html")) {
    throw "Frontend output not found: static/out/index.html"
}

Write-Host "==> [2/3] Building backend (octopus.exe)..." -ForegroundColor Cyan
$buildDir = "$root\build"
New-Item -ItemType Directory -Force -Path $buildDir | Out-Null

$time = Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz"
$ldflags = "-X 'github.com/bestruirui/octopus/internal/conf.Version=$Version' " +
           "-X 'github.com/bestruirui/octopus/internal/conf.BuildTime=$time' " +
           "-X 'github.com/bestruirui/octopus/internal/conf.Commit=$Version' " +
           "-X 'github.com/bestruirui/octopus/internal/conf.Author=bestrui' " +
           "-s -w"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

Push-Location $root
try {
    go build -o "$buildDir\octopus.exe" -ldflags "$ldflags" -tags=jsoniter .
    if ($LASTEXITCODE -ne 0) { throw "Backend build failed (exit code $LASTEXITCODE)" }
} finally {
    Pop-Location
}

Write-Host "==> [3/3] Verify version..." -ForegroundColor Cyan
& "$buildDir\octopus.exe" version

Write-Host ""
Write-Host "Build completed: $buildDir\octopus.exe" -ForegroundColor Green
