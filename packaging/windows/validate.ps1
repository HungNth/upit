[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $InstallerPath,
    [Parameter(Mandatory = $true)] [string] $Version,
    [switch] $Signed,
    [string] $SourceRoot
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSHOME 'Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1') -ErrorAction Stop

if ([string]::IsNullOrWhiteSpace($SourceRoot)) {
    $SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Product version must have three numeric components (X.Y.Z), received '$Version'."
}
$technicalVersion = "$Version.0"

if (-not (Test-Path -LiteralPath $InstallerPath -PathType Leaf)) {
    throw "Setup executable not found at: $InstallerPath"
}

$installerLeaf = Split-Path $InstallerPath -Leaf
$expectedInstallerLeaf = "upit-windows-x64-$Version-setup.exe"
if ($installerLeaf -ne $expectedInstallerLeaf) {
    throw "Expected installer filename '$expectedInstallerLeaf', received '$installerLeaf'."
}

# 1. Validate sha256 checksum sidecar
$sha256Path = "$InstallerPath.sha256"
if (-not (Test-Path -LiteralPath $sha256Path -PathType Leaf)) {
    throw "SHA-256 sidecar is missing: $sha256Path"
}
$sha256Content = (Get-Content -LiteralPath $sha256Path -Raw).Trim()
$sha256Parts = $sha256Content -split '\s+'
if ($sha256Parts.Count -lt 2) {
    throw "Malformed SHA-256 sidecar format in: $sha256Path"
}
$expectedHash = $sha256Parts[0].ToLowerInvariant()
$sha256Target = $sha256Parts[1]
if ($sha256Target -ne $installerLeaf) {
    throw "SHA-256 sidecar targets '$sha256Target', expected '$installerLeaf'."
}
$actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $InstallerPath).Hash.ToLowerInvariant()
if ($expectedHash -ne $actualHash) {
    throw "SHA-256 checksum mismatch for ${InstallerPath}: expected $expectedHash, got $actualHash."
}

# 2. Validate metadata sidecar
$metadataPath = "$InstallerPath.metadata.json"
if (-not (Test-Path -LiteralPath $metadataPath -PathType Leaf)) {
    throw "Metadata sidecar is missing: $metadataPath"
}
$metadataJson = Get-Content -LiteralPath $metadataPath -Raw
$metadata = $metadataJson | ConvertFrom-Json

if ($metadata.name -ne 'Upit') {
    throw "Expected metadata name 'Upit', received '$($metadata.name)'."
}
if ($metadata.version -ne $Version) {
    throw "Expected metadata version '$Version', received '$($metadata.version)'."
}
if ($metadata.technicalVersion -ne $technicalVersion) {
    throw "Expected metadata technicalVersion '$technicalVersion', received '$($metadata.technicalVersion)'."
}
if ($metadata.platform -ne 'windows') {
    throw "Expected metadata platform 'windows', received '$($metadata.platform)'."
}
if ($metadata.architecture -ne 'x64') {
    throw "Expected metadata architecture 'x64', received '$($metadata.architecture)'."
}
if ($metadata.signed -ne [bool]$Signed) {
    throw "Expected metadata signed=$([bool]$Signed), received '$($metadata.signed)'."
}
if ($metadata.installer -ne $installerLeaf) {
    throw "Expected metadata installer '$installerLeaf', received '$($metadata.installer)'."
}

# 3. Validate signature status of the installer executable
if ($Signed) {
    $sig = Get-AuthenticodeSignature -LiteralPath $InstallerPath
    if ($sig.Status -ne 'Valid') {
        throw "Installer executable signature status is '$($sig.Status)', expected 'Valid'."
    }
}

# 4. Check for private bin leaks in repository root
$binDir = Join-Path $SourceRoot 'bin'
if (Test-Path -LiteralPath $binDir) {
    $binFiles = Get-ChildItem -LiteralPath $binDir -File -ErrorAction SilentlyContinue
    foreach ($file in $binFiles) {
        if ($file.Name -ne 'upit.exe') {
            throw "bin must contain only upit.exe; found unexpected file: $($file.Name)"
        }
    }
}

# 5. Check that package staging in .build/package has not leaked temporary files
$stagingDir = Join-Path $SourceRoot '.build\package'
if (Test-Path -LiteralPath $stagingDir) {
    $stagedItems = Get-ChildItem -LiteralPath $stagingDir -ErrorAction SilentlyContinue
    if ($null -ne $stagedItems -and $stagedItems.Count -gt 0) {
        throw ".build/package must be empty after packaging; found $($stagedItems.Count) item(s)."
    }
}

Write-Output "Windows installer and sidecars are valid for $Version (Signed=$([bool]$Signed)): $InstallerPath"
