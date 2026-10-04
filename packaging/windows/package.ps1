[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $Version,
    [Parameter(Mandatory = $true)] [string] $DesktopPath,
    [Parameter(Mandatory = $true)] [string] $FileManagerHelperPath,
    [Parameter(Mandatory = $true)] [string] $CliPath,
    [Parameter(Mandatory = $true)] [string] $AdapterOutputPath,
    [Parameter(Mandatory = $true)] [string] $Workspace,
    [string] $Publisher = 'CN=Upit Development',
    [string] $SourceRoot,
    [string] $OutputDirectory,
    [string] $CertificatePath,
    [string] $CertificatePassword,
    [string] $TimestampServer = 'http://timestamp.digicert.com',
    [string] $ProtectedTag
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

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

if ([string]::IsNullOrWhiteSpace($Publisher)) {
    throw 'Publisher must not be empty.'
}
if ($CertificatePath -and ($ProtectedTag -notmatch '^v\d+\.\d+\.\d+$' -or $env:UPIT_PROTECTED_RELEASE -ne 'true')) {
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

$makeAppx = Find-SdkTool 'makeappx.exe'
$cmakeCommand = Get-Command cmake.exe -ErrorAction SilentlyContinue
if ($null -eq $cmakeCommand) { throw 'cmake.exe is required to build the Explorer command adapter.' }
$cmake = $cmakeCommand.Source

$stage = Join-Path $Workspace 'msix-stage'
$nativeBuild = Join-Path $Workspace 'native-build'
$installerPayload = Join-Path $Workspace 'installer-payload'
New-Item -ItemType Directory -Path $stage -Force | Out-Null
New-Item -ItemType Directory -Path $installerPayload -Force | Out-Null
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

try {
    & $cmake -S (Join-Path $SourceRoot 'native\windows\explorer-command') -B $nativeBuild
    if ($LASTEXITCODE -ne 0) { throw "cmake configure failed with exit code $LASTEXITCODE." }
    & $cmake --build $nativeBuild --config Release
    if ($LASTEXITCODE -ne 0) { throw "cmake build failed with exit code $LASTEXITCODE." }

    $nativeDll = Get-ChildItem -Path $nativeBuild -Filter 'upit-explorer-command.dll' -Recurse -File | Select-Object -First 1
    if ($null -eq $nativeDll) { throw 'CMake did not produce upit-explorer-command.dll.' }

    Copy-Item -LiteralPath $nativeDll.FullName -Destination $AdapterOutputPath -Force

    $required = @(
        $DesktopPath,
        $FileManagerHelperPath,
        $CliPath,
        $AdapterOutputPath,
        (Join-Path $PSScriptRoot 'assets\logo.png')
    )
    foreach ($file in $required) {
        if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
            throw "Required package input is missing: $file"
        }
    }

    $logoBytes = [System.IO.File]::ReadAllBytes((Join-Path $PSScriptRoot 'assets\logo.png'))
    if ($logoBytes.Length -lt 24 -or $logoBytes[0] -ne 0x89 -or $logoBytes[1] -ne 0x50 -or $logoBytes[2] -ne 0x4e -or $logoBytes[3] -ne 0x47) {
        throw 'Package logo must be a valid PNG.'
    }
    $logoWidth = ($logoBytes[16] -shl 24) -bor ($logoBytes[17] -shl 16) -bor ($logoBytes[18] -shl 8) -bor $logoBytes[19]
    $logoHeight = ($logoBytes[20] -shl 24) -bor ($logoBytes[21] -shl 16) -bor ($logoBytes[22] -shl 8) -bor $logoBytes[23]
    if ($logoWidth -lt 150 -or $logoHeight -lt 150) {
        throw "Package logo must be at least 150x150; received ${logoWidth}x${logoHeight}."
    }

    Copy-Item -LiteralPath $DesktopPath -Destination (Join-Path $stage 'upit-desktop.exe') -Force
    Copy-Item -LiteralPath $FileManagerHelperPath -Destination (Join-Path $stage 'upit-file-manager.exe') -Force
    Copy-Item -LiteralPath $CliPath -Destination (Join-Path $stage 'upit.exe') -Force
    Copy-Item -LiteralPath $AdapterOutputPath -Destination (Join-Path $stage 'upit-explorer-command.dll') -Force

    New-Item -ItemType Directory -Path (Join-Path $stage 'assets') -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'assets\logo.png') -Destination (Join-Path $stage 'assets\logo.png') -Force

    $manifest = (Get-Content (Join-Path $PSScriptRoot 'AppxManifest.xml.in') -Raw).
        Replace('@VERSION@', $technicalVersion).
        Replace('@PUBLISHER@', [System.Security.SecurityElement]::Escape($Publisher))
    if ($manifest.Contains('@VERSION@') -or $manifest.Contains('@PUBLISHER@')) {
        throw 'Manifest replacement left unresolved placeholders.'
    }
    $manifestPath = Join-Path $stage 'AppxManifest.xml'
    Set-Content -LiteralPath $manifestPath -Value $manifest -Encoding UTF8
    [xml](Get-Content -LiteralPath $manifestPath -Raw) | Out-Null

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
            (Join-Path $stage 'upit-explorer-command.dll'),
            (Join-Path $stage 'upit-desktop.exe'),
            (Join-Path $stage 'upit-file-manager.exe'),
            (Join-Path $stage 'upit.exe')
        )
        foreach ($target in $signTargets) {
            & $signTool @signArgs $target
            if ($LASTEXITCODE -ne 0) { throw "signtool failed for $target with exit code $LASTEXITCODE." }
        }
    }

    $payloadHashes = @{}
    foreach ($name in @('upit.exe', 'upit-desktop.exe', 'upit-file-manager.exe', 'upit-explorer-command.dll')) {
        $file = Join-Path $stage $name
        $payloadHashes[$name] = (Get-FileHash -Algorithm SHA256 -LiteralPath $file).Hash.ToLowerInvariant()
        Copy-Item -LiteralPath $file -Destination $installerPayload -Force
        Remove-Item -LiteralPath $file -Force
    }
    $payloadHashes | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $stage 'payload-sha256.json') -Encoding UTF8

    $packagePath = Join-Path $Workspace "upit-windows-x64-$technicalVersion.msix"
    & $makeAppx pack /d $stage /p $packagePath /nv
    if ($LASTEXITCODE -ne 0) { throw "makeappx failed with exit code $LASTEXITCODE." }

    if ($CertificatePath) {
        & $signTool @signArgs $packagePath
        if ($LASTEXITCODE -ne 0) { throw "signtool failed for the MSIX with exit code $LASTEXITCODE." }
    }

    New-Item -ItemType Directory -Path (Join-Path $installerPayload 'repair') -Force | Out-Null
    Copy-Item -LiteralPath $packagePath -Destination (Join-Path $installerPayload 'repair\Upit.msix') -Force

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
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
    if (Test-Path -LiteralPath $nativeBuild) { Remove-Item -LiteralPath $nativeBuild -Recurse -Force }
    if (Test-Path -LiteralPath $installerPayload) { Remove-Item -LiteralPath $installerPayload -Recurse -Force }
}
