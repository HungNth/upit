[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $Tag,
    [Parameter(Mandatory = $true)] [string] $CertificatePath,
    [Parameter(Mandatory = $true)] [string] $CertificatePassword,
    [string] $Publisher = 'CN=Upit Development',
    [string] $SourceRoot,
    [string] $OutputDirectory,
    [string] $TimestampServer = 'http://timestamp.digicert.com'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($SourceRoot)) { $SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path }
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $SourceRoot 'dist\windows' }

if ($Tag -notmatch '^v(\d+)\.(\d+)\.(\d+)$') {
    throw "Signing and publishing require a protected SemVer tag vX.Y.Z; received '$Tag'."
}
if ($env:UPIT_PROTECTED_RELEASE -ne 'true') {
    throw 'UPIT_PROTECTED_RELEASE=true is required by the protected release environment.'
}
$version = "$($Matches[1]).$($Matches[2]).$($Matches[3]).0"

& (Join-Path $PSScriptRoot 'validate.ps1') -Version $version -Publisher $Publisher -SourceRoot $SourceRoot
& (Join-Path $PSScriptRoot 'package.ps1') -Version $version -Publisher $Publisher -SourceRoot $SourceRoot -OutputDirectory $OutputDirectory -CertificatePath $CertificatePath -CertificatePassword $CertificatePassword -TimestampServer $TimestampServer -ProtectedTag $Tag
