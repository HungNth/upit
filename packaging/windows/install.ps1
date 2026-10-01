[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $PackagePath,
    [Parameter(Mandatory = $true)] [string] $InstallDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path $PackagePath -PathType Leaf)) { throw "Package not found: $PackagePath" }
if (-not (Test-Path $InstallDirectory -PathType Container)) { throw "Install directory not found: $InstallDirectory" }

Add-AppxPackage -Path (Resolve-Path $PackagePath) -ExternalLocation (Resolve-Path $InstallDirectory)
Write-Output 'Upit File Manager Upload package registered for the current user.'
