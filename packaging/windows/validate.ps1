[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $Version,
    [Parameter(Mandatory = $true)] [string] $DesktopPath,
    [Parameter(Mandatory = $true)] [string] $FileManagerHelperPath,
    [Parameter(Mandatory = $true)] [string] $CliPath,
    [string] $Publisher = 'CN=Upit Development',
    [string] $SourceRoot
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($SourceRoot)) {
    $SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Product version must have three numeric components (X.Y.Z), received '$Version'."
}
if ([string]::IsNullOrWhiteSpace($Publisher)) {
    throw 'Publisher must not be empty.'
}

$technicalVersion = "$Version.0"

$inputs = @(
    $DesktopPath,
    $FileManagerHelperPath,
    $CliPath,
    (Join-Path $PSScriptRoot 'assets\logo.png'),
    (Join-Path $SourceRoot 'native\windows\explorer-command\explorer_command.cpp'),
    (Join-Path $SourceRoot 'native\windows\explorer-command\CMakeLists.txt')
)

foreach ($input in $inputs) {
    if (-not (Test-Path -LiteralPath $input -PathType Leaf)) {
        throw "Unsigned package input is missing: $input"
    }
}

$logoBytes = [System.IO.File]::ReadAllBytes((Join-Path $PSScriptRoot 'assets\logo.png'))
if ($logoBytes.Length -lt 24 -or $logoBytes[0] -ne 0x89 -or $logoBytes[1] -ne 0x50 -or $logoBytes[2] -ne 0x4e -or $logoBytes[3] -ne 0x47) {
    throw 'Package logo is not a PNG.'
}
$logoWidth = ($logoBytes[16] -shl 24) -bor ($logoBytes[17] -shl 16) -bor ($logoBytes[18] -shl 8) -bor $logoBytes[19]
$logoHeight = ($logoBytes[20] -shl 24) -bor ($logoBytes[21] -shl 16) -bor ($logoBytes[22] -shl 8) -bor $logoBytes[23]
if ($logoWidth -lt 150 -or $logoHeight -lt 150) {
    throw "Package logo must be at least 150x150; received ${logoWidth}x${logoHeight}."
}

$manifest = (Get-Content (Join-Path $PSScriptRoot 'AppxManifest.xml.in') -Raw).
    Replace('@VERSION@', $technicalVersion).
    Replace('@PUBLISHER@', [System.Security.SecurityElement]::Escape($Publisher))
if ($manifest.Contains('@VERSION@') -or $manifest.Contains('@PUBLISHER@')) {
    throw 'Manifest contains unresolved placeholders.'
}
[xml]$manifest | Out-Null
if (-not $manifest.Contains('windows.fileExplorerContextMenus')) {
    throw 'Manifest does not register the Windows 11 Explorer context-menu extension.'
}
if (-not $manifest.Contains('5C4A83F8-4D35-4C0C-B25B-2A57FDDA8BB4')) {
    throw 'Manifest does not register the Explorer command CLSID.'
}

Write-Output "Unsigned Windows File Manager Upload package inputs are valid for $Version ($technicalVersion)."
