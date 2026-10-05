[CmdletBinding()]
param(
    [string] $Version = '0.0.0',
    [string] $Publisher = $(if ($env:WINDOWS_PUBLISHER) { $env:WINDOWS_PUBLISHER } else { 'CN=Upit Development' }),
    [string] $CertificatePath = $env:WINDOWS_CERTIFICATE_PATH,
    [string] $CertificatePassword = $env:WINDOWS_CERTIFICATE_PASSWORD,
    [string] $TimestampServer = $(if ($env:WINDOWS_TIMESTAMP_SERVER) { $env:WINDOWS_TIMESTAMP_SERVER } else { 'http://timestamp.digicert.com' }),
    [string] $ProtectedTag = $env:WINDOWS_PROTECTED_TAG,
    [string] $SourceRoot,
    [string] $OutputDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
if ($env:OS -ne 'Windows_NT' -or $arch -ne 'AMD64' -or [Environment]::OSVersion.Version.Build -lt 22000) {
    throw 'Desktop packaging requires a native Windows 11 x64 host.'
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Product version must have three numeric components (X.Y.Z), received '$Version'."
}

if ([string]::IsNullOrWhiteSpace($SourceRoot)) {
    $SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $SourceRoot 'dist\windows'
}

$goCommand = Get-Command go.exe -ErrorAction SilentlyContinue
if ($null -eq $goCommand) { $goCommand = Get-Command go -ErrorAction SilentlyContinue }
if ($null -eq $goCommand) { throw 'go is required to build Upit binaries.' }
$go = $goCommand.Source

$npmCommand = Get-Command npm.cmd -ErrorAction SilentlyContinue
if ($null -eq $npmCommand) { $npmCommand = Get-Command npm -ErrorAction SilentlyContinue }
if ($null -eq $npmCommand) { throw 'npm is required to build frontend assets.' }
$npm = $npmCommand.Source

$stagingRoot = Join-Path $SourceRoot '.build\package'
$runDir = Join-Path $stagingRoot ("run-" + [guid]::NewGuid().ToString('N'))

try {
    New-Item -ItemType Directory -Path $stagingRoot -Force | Out-Null
    New-Item -ItemType Directory -Path $runDir -Force | Out-Null
    New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

    $frontendDir = Join-Path $SourceRoot 'cmd\upit-desktop\frontend'
    & $npm --prefix $frontendDir ci
    if ($LASTEXITCODE -ne 0) { throw "npm ci failed with exit code $LASTEXITCODE." }
    & $npm --prefix $frontendDir run build
    if ($LASTEXITCODE -ne 0) { throw "npm run build failed with exit code $LASTEXITCODE." }

    $stagedCli = Join-Path $runDir 'upit.exe'
    $stagedFileManager = Join-Path $runDir 'upit-file-manager.exe'
    $stagedDesktop = Join-Path $runDir 'upit-desktop.exe'

    & $go build -o $stagedCli (Join-Path $SourceRoot 'cmd\upit')
    if ($LASTEXITCODE -ne 0) { throw "Building CLI failed with exit code $LASTEXITCODE." }

    & $go build -o $stagedFileManager (Join-Path $SourceRoot 'cmd\upit-file-manager')
    if ($LASTEXITCODE -ne 0) { throw "Building file manager helper failed with exit code $LASTEXITCODE." }

    & $go build -o $stagedDesktop (Join-Path $SourceRoot 'cmd\upit-desktop')
    if ($LASTEXITCODE -ne 0) { throw "Building desktop executable failed with exit code $LASTEXITCODE." }

    & (Join-Path $PSScriptRoot 'validate.ps1') `
        -Version $Version `
        -Publisher $Publisher `
        -SourceRoot $SourceRoot `
        -DesktopPath $stagedDesktop `
        -FileManagerHelperPath $stagedFileManager `
        -CliPath $stagedCli

    $stagedAdapterDll = Join-Path $runDir 'upit-explorer-command.dll'
    $packageWorkspace = Join-Path $runDir 'workspace'
    New-Item -ItemType Directory -Path $packageWorkspace -Force | Out-Null

    $packageArgs = @{
        Version = $Version
        Publisher = $Publisher
        SourceRoot = $SourceRoot
        OutputDirectory = $OutputDirectory
        DesktopPath = $stagedDesktop
        FileManagerHelperPath = $stagedFileManager
        CliPath = $stagedCli
        AdapterOutputPath = $stagedAdapterDll
        Workspace = $packageWorkspace
    }

    if ($CertificatePath) {
        $packageArgs['CertificatePath'] = $CertificatePath
        if ($CertificatePassword) { $packageArgs['CertificatePassword'] = $CertificatePassword }
        if ($TimestampServer) { $packageArgs['TimestampServer'] = $TimestampServer }
        if ($ProtectedTag) { $packageArgs['ProtectedTag'] = $ProtectedTag }
    }

    & (Join-Path $PSScriptRoot 'package.ps1') @packageArgs
} finally {
    if (Test-Path -LiteralPath $runDir) {
        Remove-Item -LiteralPath $runDir -Recurse -Force
    }
}
