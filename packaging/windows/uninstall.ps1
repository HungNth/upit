[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$packages = Get-AppxPackage -Name 'HungNth.Upit'
foreach ($package in $packages) {
    Remove-AppxPackage -Package $package.PackageFullName
}
if ($packages.Count -eq 0) {
    Write-Output 'Upit File Manager Upload package was not registered for the current user.'
} else {
    Write-Output 'Upit File Manager Upload package and Explorer registration removed.'
}
