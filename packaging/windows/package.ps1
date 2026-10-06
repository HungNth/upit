[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $Version,
    [Parameter(Mandatory = $true)] [string] $DesktopPath,
    [Parameter(Mandatory = $true)] [string] $FileManagerHelperPath,
    [Parameter(Mandatory = $true)] [string] $CliPath,
    [string] $LauncherPath,
    [string] $InstallWorkerPath,
    [Parameter(Mandatory = $true)] [string] $Workspace,
    [string] $SourceRoot,
    [string] $OutputDirectory,
    [string] $CertificatePath,
    [string] $CertificatePassword,
    [string] $TimestampServer = 'http://timestamp.digicert.com',
    [string] $ProtectedTag
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSHOME 'Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1') -ErrorAction Stop

if ([string]::IsNullOrWhiteSpace($SourceRoot)) {
    $SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $SourceRoot 'dist\windows'
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Product version must have three numeric components (X.Y.Z), received '$Version'."
}
$technicalVersion = "$Version.0"

if ($CertificatePath -and ($ProtectedTag -ne "v$Version" -or $env:UPIT_PROTECTED_RELEASE -ne 'true')) {
    throw 'Signing is permitted only for a protected SemVer tag with UPIT_PROTECTED_RELEASE=true.'
}

function Find-SdkTool([string] $Name) {
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -ne $command) { return $command.Source }
    $roots = @(
        'C:\Program Files (x86)\Windows Kits\10\bin',
        'C:\Program Files\Windows Kits\10\bin'
    ) | Where-Object { Test-Path $_ }
    $candidate = Get-ChildItem -Path $roots -Filter $Name -Recurse -File -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match '\\x64\\' } |
        Sort-Object FullName -Descending |
        Select-Object -First 1
    if ($null -eq $candidate) { throw "$Name is required but was not found in PATH or the Windows SDK." }
    return $candidate.FullName
}

$installerPayload = Join-Path $Workspace 'installer-payload'
New-Item -ItemType Directory -Path $installerPayload -Force | Out-Null
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

try {
    $required = @(
        $DesktopPath,
        $FileManagerHelperPath,
        $CliPath
    )
    if ($LauncherPath) {
        $required += $LauncherPath
    }
    if ($InstallWorkerPath) {
        $required += $InstallWorkerPath
    }
    foreach ($file in $required) {
        if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
            throw "Required package input is missing: $file"
        }
    }

    Copy-Item -LiteralPath $DesktopPath -Destination (Join-Path $installerPayload 'upit-desktop.exe') -Force
    Copy-Item -LiteralPath $FileManagerHelperPath -Destination (Join-Path $installerPayload 'upit-file-manager.exe') -Force
    Copy-Item -LiteralPath $CliPath -Destination (Join-Path $installerPayload 'upit.exe') -Force
    if ($LauncherPath) {
        Copy-Item -LiteralPath $LauncherPath -Destination (Join-Path $installerPayload 'upit-launcher.exe') -Force
    }
    if ($InstallWorkerPath) {
        Copy-Item -LiteralPath $InstallWorkerPath -Destination (Join-Path $installerPayload 'upit-install.exe') -Force
    }

    $signTool = $null
    $signArgs = $null
    if ($CertificatePath) {
        if (-not (Test-Path -LiteralPath $CertificatePath -PathType Leaf)) {
            throw "Signing certificate is missing: $CertificatePath"
        }
        $signTool = Find-SdkTool 'signtool.exe'
        $signArgs = @('sign', '/fd', 'SHA256', '/f', $CertificatePath)
        if ($CertificatePassword) { $signArgs += @('/p', $CertificatePassword) }
        if ($TimestampServer) { $signArgs += @('/tr', $TimestampServer, '/td', 'SHA256') }

        $signTargets = @(
            (Join-Path $installerPayload 'upit-desktop.exe'),
            (Join-Path $installerPayload 'upit-file-manager.exe'),
            (Join-Path $installerPayload 'upit.exe')
        )
        if ($LauncherPath) {
            $signTargets += (Join-Path $installerPayload 'upit-launcher.exe')
        }
        if ($InstallWorkerPath) {
            $signTargets += (Join-Path $installerPayload 'upit-install.exe')
        }
        foreach ($target in $signTargets) {
            & $signTool @signArgs $target
            if ($LASTEXITCODE -ne 0) { throw "signtool failed for $target with exit code $LASTEXITCODE." }
        }
    }

    $makensis = Get-Command makensis.exe -ErrorAction Stop
    $installerPath = Join-Path $OutputDirectory ("upit-windows-x64-$Version-setup.exe")
    & $makensis.Source "/DUPIT_VERSION=$Version" "/DUPIT_TECHNICAL_VERSION=$technicalVersion" "/DUPIT_PAYLOAD=$installerPayload" "/DUPIT_OUTPUT=$installerPath" (Join-Path $PSScriptRoot 'installer.nsi')
    if ($LASTEXITCODE -ne 0) { throw "NSIS installer compilation failed with exit code $LASTEXITCODE." }

    if ($CertificatePath) {
        & $signTool @signArgs $installerPath
        if ($LASTEXITCODE -ne 0) { throw "Installer signing failed with exit code $LASTEXITCODE." }
    }

    $installerHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $installerPath).Hash.ToLowerInvariant()
    Set-Content -LiteralPath "$installerPath.sha256" -Value "$installerHash  $(Split-Path $installerPath -Leaf)" -Encoding ASCII

    @{
        name = 'Upit'
        version = $Version
        technicalVersion = $technicalVersion
        platform = 'windows'
        architecture = 'x64'
        signed = [bool]$CertificatePath
        installer = (Split-Path $installerPath -Leaf)
    } | ConvertTo-Json | Set-Content -LiteralPath "$installerPath.metadata.json" -Encoding UTF8

    Write-Output $installerPath
} finally {
    if (Test-Path -LiteralPath $installerPayload) {
        Remove-Item -LiteralPath $installerPayload -Recurse -Force
    }
    if (Test-Path -LiteralPath $Workspace) {
        $remaining = Get-ChildItem -LiteralPath $Workspace -Force -ErrorAction SilentlyContinue
        if ($null -eq $remaining -or $remaining.Count -eq 0) {
            Remove-Item -LiteralPath $Workspace -Force -ErrorAction SilentlyContinue
        }
    }
}
