[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $Version,
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
if ([string]::IsNullOrWhiteSpace($SourceRoot)) { $SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path }
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $SourceRoot 'dist\windows' }
if ($Version -notmatch '^\d+\.\d+\.\d+\.\d+$') {
    throw "Version must have four numeric components for MSIX, received '$Version'."
}
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
        (Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin'),
        (Join-Path ${env:ProgramFiles} 'Windows Kits\10\bin')
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
$stage = Join-Path ([System.IO.Path]::GetTempPath()) ("upit-msix-" + [guid]::NewGuid().ToString('N'))
$nativeBuild = Join-Path ([System.IO.Path]::GetTempPath()) ("upit-explorer-command-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $stage -Force | Out-Null
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
try {
    & $cmake -S (Join-Path $SourceRoot 'native\windows\explorer-command') -B $nativeBuild
    if ($LASTEXITCODE -ne 0) { throw "cmake configure failed with exit code $LASTEXITCODE." }
    & $cmake --build $nativeBuild --config Release
    if ($LASTEXITCODE -ne 0) { throw "cmake build failed with exit code $LASTEXITCODE." }
    $nativeDll = Get-ChildItem -Path $nativeBuild -Filter 'upit-explorer-command.dll' -Recurse -File | Select-Object -First 1
    if ($null -eq $nativeDll) { throw 'CMake did not produce upit-explorer-command.dll.' }
    $nativeOutput = Join-Path $SourceRoot 'bin\upit-explorer-command.dll'
    Copy-Item $nativeDll.FullName $nativeOutput -Force
    $required = @(
        (Join-Path $SourceRoot 'bin\upit-desktop.exe'),
        (Join-Path $SourceRoot 'bin\upit-file-manager.exe'),
        $nativeOutput,
        (Join-Path $PSScriptRoot 'assets\logo.png')
    )
    foreach ($file in $required) {
        if (-not (Test-Path $file -PathType Leaf)) { throw "Required package input is missing: $file" }
    }
    $logoBytes = [System.IO.File]::ReadAllBytes((Join-Path $PSScriptRoot 'assets\logo.png'))
    if ($logoBytes.Length -lt 24 -or $logoBytes[0] -ne 0x89 -or $logoBytes[1] -ne 0x50 -or $logoBytes[2] -ne 0x4e -or $logoBytes[3] -ne 0x47) {
        throw 'Package logo must be a valid PNG.'
    }
    $logoWidth = ($logoBytes[16] -shl 24) -bor ($logoBytes[17] -shl 16) -bor ($logoBytes[18] -shl 8) -bor $logoBytes[19]
    $logoHeight = ($logoBytes[20] -shl 24) -bor ($logoBytes[21] -shl 16) -bor ($logoBytes[22] -shl 8) -bor $logoBytes[23]
    if ($logoWidth -lt 150 -or $logoHeight -lt 150) { throw "Package logo must be at least 150x150; received ${logoWidth}x${logoHeight}." }

    Copy-Item (Join-Path $SourceRoot 'bin\upit-desktop.exe') $stage
    Copy-Item (Join-Path $SourceRoot 'bin\upit-file-manager.exe') $stage
    Copy-Item $nativeOutput $stage
    New-Item -ItemType Directory -Path (Join-Path $stage 'assets') -Force | Out-Null
    Copy-Item (Join-Path $PSScriptRoot 'assets\logo.png') (Join-Path $stage 'assets\logo.png')

    $manifest = (Get-Content (Join-Path $PSScriptRoot 'AppxManifest.xml.in') -Raw).
        Replace('@VERSION@', $Version).
        Replace('@PUBLISHER@', $Publisher)
    if ($manifest.Contains('@VERSION@') -or $manifest.Contains('@PUBLISHER@')) {
        throw 'Manifest replacement left unresolved placeholders.'
    }
    $manifestPath = Join-Path $stage 'AppxManifest.xml'
    Set-Content -Path $manifestPath -Value $manifest -Encoding UTF8
    [xml](Get-Content $manifestPath -Raw) | Out-Null

    $signTool = $null
    $signArgs = $null
    if ($CertificatePath) {
        if (-not (Test-Path $CertificatePath -PathType Leaf)) { throw "Signing certificate is missing: $CertificatePath" }
        $signTool = Find-SdkTool 'signtool.exe'
        $signArgs = @('sign', '/fd', 'SHA256', '/f', $CertificatePath)
        if ($CertificatePassword) { $signArgs += @('/p', $CertificatePassword) }
        if ($TimestampServer) { $signArgs += @('/tr', $TimestampServer, '/td', 'SHA256') }
        $signTargets = @(
            $nativeOutput,
            (Join-Path $SourceRoot 'bin\upit-desktop.exe'),
            (Join-Path $SourceRoot 'bin\upit-file-manager.exe')
        )
        $cliPath = Join-Path $SourceRoot 'bin\upit.exe'
        if (Test-Path $cliPath -PathType Leaf) { $signTargets += $cliPath }
        foreach ($target in $signTargets) {
            & $signTool @signArgs $target
            if ($LASTEXITCODE -ne 0) { throw "signtool failed for $target with exit code $LASTEXITCODE." }
        }
        Copy-Item $nativeOutput (Join-Path $stage 'upit-explorer-command.dll') -Force
		Copy-Item (Join-Path $SourceRoot 'bin\upit-desktop.exe') $stage -Force
		Copy-Item (Join-Path $SourceRoot 'bin\upit-file-manager.exe') $stage -Force
    }

    $packagePath = Join-Path $OutputDirectory ("upit-windows-x64-$Version.msix")
    & $makeAppx pack /d $stage /p $packagePath /nv
    if ($LASTEXITCODE -ne 0) { throw "makeappx failed with exit code $LASTEXITCODE." }

    if ($CertificatePath) {
        & $signTool @signArgs $packagePath
        if ($LASTEXITCODE -ne 0) { throw "signtool failed for the MSIX with exit code $LASTEXITCODE." }
    }

    $hash = (Get-FileHash -Algorithm SHA256 $packagePath).Hash.ToLowerInvariant()
    Set-Content -Path "$packagePath.sha256" -Value "$hash  $(Split-Path $packagePath -Leaf)" -Encoding ASCII
    Write-Output $packagePath
} finally {
    if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
    if (Test-Path $nativeBuild) { Remove-Item $nativeBuild -Recurse -Force }
}
