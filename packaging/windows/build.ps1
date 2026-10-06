[CmdletBinding()]
param(
    [string] $Version = '0.0.0',
    [string] $CertificatePath = $env:WINDOWS_CERTIFICATE_PATH,
    [string] $CertificatePassword = $env:WINDOWS_CERTIFICATE_PASSWORD,
    [string] $TimestampServer = $(if ($env:WINDOWS_TIMESTAMP_SERVER) { $env:WINDOWS_TIMESTAMP_SERVER } else { 'http://timestamp.digicert.com' }),
    [string] $ProtectedTag = $env:WINDOWS_PROTECTED_TAG,
    [string] $LauncherFormat = '1',
    [string] $SourceRoot,
    [string] $OutputDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# PowerShell 7 parents can put incompatible modules ahead of Windows PowerShell's.
Import-Module (Join-Path $PSHOME 'Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1') -ErrorAction Stop
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
    $stagedInstall = Join-Path $runDir 'upit-install.exe'
    $stagedLauncher = Join-Path $runDir 'upit-launcher.exe'

    $vParts = $Version.Split('.')
    $rcVersion = "$($vParts[0]),$($vParts[1]),$($vParts[2]),0"
    $icoPath = (Join-Path $SourceRoot 'assets\branding\app.ico') -replace '\\', '/'

    $stagedCmdRoot = Join-Path $runDir 'cmd'
    New-Item -ItemType Directory -Path $stagedCmdRoot -Force | Out-Null

    function Build-PackageResource([string] $targetDir, [bool] $includeIcon = $false) {
        $rcPath = Join-Path $targetDir 'package_version.rc'
        $iconLine = if ($includeIcon) { "1 ICON `"$icoPath`"`r`n" } else { "" }
        $rcText = @"
${iconLine}1 VERSIONINFO
FILEVERSION $rcVersion
PRODUCTVERSION $rcVersion
FILEFLAGSMASK 0x3fL
FILEFLAGS 0x0L
FILEOS 0x40004L
FILETYPE 0x1L
FILESUBTYPE 0x0L
BEGIN
    BLOCK "StringFileInfo"
    BEGIN
        BLOCK "040904b0"
        BEGIN
            VALUE "CompanyName", "Upit\0"
            VALUE "FileDescription", "Upit\0"
            VALUE "FileVersion", "$Version.0\0"
            VALUE "InternalName", "Upit\0"
            VALUE "LegalCopyright", "Upit contributors\0"
            VALUE "OriginalFilename", "Upit.exe\0"
            VALUE "ProductName", "Upit\0"
            VALUE "ProductVersion", "$Version\0"
        END
    END
    BLOCK "VarFileInfo"
    BEGIN
        VALUE "Translation", 0x409, 1200
    END
END
"@
        Set-Content -LiteralPath $rcPath -Value $rcText -Encoding ASCII
        $sysoPath = Join-Path $targetDir 'package_version_windows_amd64.syso'
        $windresCmd = Get-Command windres.exe -ErrorAction SilentlyContinue
        if ($null -eq $windresCmd) {
            throw "windres.exe is required for embedding Windows PE resources but was not found in PATH."
        }
        & $windresCmd.Source -i $rcPath -O coff -F pe-x86-64 -o $sysoPath
        if ($LASTEXITCODE -ne 0) { throw "windres compilation failed for $rcPath with exit code $LASTEXITCODE." }
    }

    function Stage-Command([string] $cmdName, [bool] $includeIcon = $false) {
        $srcCmdDir = Join-Path $SourceRoot ("cmd\" + $cmdName)
        $dstCmdDir = Join-Path $stagedCmdRoot $cmdName
        New-Item -ItemType Directory -Path $dstCmdDir -Force | Out-Null
        # Copy ONLY top-level .go files (skipping tests)
        Get-ChildItem -LiteralPath $srcCmdDir -File -Filter '*.go' |
            Where-Object { -not $_.Name.EndsWith('_test.go') } |
            ForEach-Object { Copy-Item -LiteralPath $_.FullName -Destination $dstCmdDir -Force }
        Build-PackageResource -targetDir $dstCmdDir -includeIcon $includeIcon
        return $dstCmdDir
    }

    $cliDir = Stage-Command 'upit'
    $fmDir = Stage-Command 'upit-file-manager'
    $installDir = Stage-Command 'upit-install'
    $launcherDir = Stage-Command 'upit-launcher'

    # For desktop: also copy required frontend/dist assets for go:embed
    $desktopDir = Stage-Command 'upit-desktop' -includeIcon $true
    $srcFrontendDist = Join-Path $SourceRoot 'cmd\upit-desktop\frontend\dist'
    $dstFrontendDist = Join-Path $desktopDir 'frontend\dist'
    New-Item -ItemType Directory -Path (Split-Path $dstFrontendDist) -Force | Out-Null
    Copy-Item -LiteralPath $srcFrontendDist -Destination $dstFrontendDist -Recurse -Force

    & $go build -o $stagedCli $cliDir
    if ($LASTEXITCODE -ne 0) { throw "Building CLI failed with exit code $LASTEXITCODE." }

    & $go build -ldflags '-H=windowsgui' -o $stagedFileManager $fmDir
    if ($LASTEXITCODE -ne 0) { throw "Building file manager helper failed with exit code $LASTEXITCODE." }

    & $go build -ldflags '-H=windowsgui' -o $stagedDesktop $desktopDir
    if ($LASTEXITCODE -ne 0) { throw "Building desktop executable failed with exit code $LASTEXITCODE." }

    & $go build -o $stagedInstall $installDir
    if ($LASTEXITCODE -ne 0) { throw "Building install worker failed with exit code $LASTEXITCODE." }

    $launcherLdflags = "-X main.launcherFormat=$LauncherFormat"
    & $go build -ldflags $launcherLdflags -o $stagedLauncher $launcherDir
    if ($LASTEXITCODE -ne 0) { throw "Building launcher failed with exit code $LASTEXITCODE." }
    $packageWorkspace = Join-Path $runDir 'workspace'
    New-Item -ItemType Directory -Path $packageWorkspace -Force | Out-Null

$packageArgs = @{
    Version = $Version
    SourceRoot = $SourceRoot
    OutputDirectory = $OutputDirectory
    DesktopPath = $stagedDesktop
    FileManagerHelperPath = $stagedFileManager
    CliPath = $stagedCli
    LauncherPath = $stagedLauncher
    InstallWorkerPath = $stagedInstall
    Workspace = $packageWorkspace
}
    if ($CertificatePath) {
        $packageArgs['CertificatePath'] = $CertificatePath
        if ($CertificatePassword) { $packageArgs['CertificatePassword'] = $CertificatePassword }
        if ($TimestampServer) { $packageArgs['TimestampServer'] = $TimestampServer }
        if ($ProtectedTag) { $packageArgs['ProtectedTag'] = $ProtectedTag }
    }

    $packageOutput = & (Join-Path $PSScriptRoot 'package.ps1') @packageArgs
    $installerPath = if ($packageOutput -is [array]) { $packageOutput[-1] } else { $packageOutput }

    & (Join-Path $PSScriptRoot 'validate.ps1') `
        -InstallerPath $installerPath `
        -Version $Version `
        -DesktopPath $stagedDesktop `
        -LauncherPath $stagedLauncher `
        -ExpectedLauncherFormat $LauncherFormat `
        -Signed:([bool]$CertificatePath) `
        -SourceRoot $SourceRoot

    if (Test-Path -LiteralPath $runDir) {
        Remove-Item -LiteralPath $runDir -Recurse -Force
    }
    Write-Output $installerPath
} finally {
    if (Test-Path -LiteralPath $runDir) {
        Remove-Item -LiteralPath $runDir -Recurse -Force
    }
}
