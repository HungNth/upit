[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $PackagePath,
    [Parameter(Mandatory = $true)] [string] $InstallDirectory,
    [string] $EvidencePath = (Join-Path (Get-Location) 'native-smoke-evidence.json')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$os = Get-CimInstance Win32_OperatingSystem
if ([int]$os.BuildNumber -lt 22000) { throw "Windows 11 build required; received $($os.BuildNumber)." }
if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw "Windows x64 runner required; received $env:PROCESSOR_ARCHITECTURE." }
if (-not (Test-Path $PackagePath -PathType Leaf)) { throw "Signed package not found: $PackagePath" }

$payloadNames = @('upit-desktop.exe', 'upit-file-manager.exe', 'upit-explorer-command.dll')
$payloadRoot = Resolve-Path $InstallDirectory
foreach ($name in $payloadNames) {
    $path = Join-Path $payloadRoot $name
    if (-not (Test-Path $path -PathType Leaf)) { throw "External-location payload is missing $name." }
}
if (-not (Test-Path (Join-Path $payloadRoot 'upit.exe') -PathType Leaf)) { throw 'External-location payload is missing upit.exe for CLI continuity smoke.' }
& (Join-Path $PSScriptRoot 'install.ps1') -PackagePath $PackagePath -InstallDirectory $payloadRoot
$installed = Get-AppxPackage -Name 'HungNth.Upit'
if ($null -eq $installed) { throw 'Package identity was not registered.' }
$manifest = [xml](Get-AppxPackageManifest -Package $installed.PackageFullName)
$extension = $manifest.Package.Applications.Application.Extensions.Extension |
    Where-Object { $_.Category -eq 'windows.fileExplorerContextMenus' }
if ($null -eq $extension) { throw 'Primary File Explorer context-menu extension was not registered.' }

$fixtureDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("upit-native-smoke-" + [guid]::NewGuid().ToString('N'))
$configBackup = Join-Path ([System.IO.Path]::GetTempPath()) ("upit-config-backup-" + [guid]::NewGuid().ToString('N'))
 $serverJob = $null
 $helper = $null
$configDirectory = Join-Path $HOME '.config\upit'
$configFiles = @('config.json', 'custom-uploader.json', 'custom-shortener.json')
$backedUp = @{}
$automated = [ordered]@{}
$operatorObserved = [ordered]@{}
New-Item -ItemType Directory -Path $fixtureDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $configBackup -Force | Out-Null
Set-Content -Path (Join-Path $fixtureDirectory 'upit-smoke.txt') -Value 'protected Windows native smoke fixture' -Encoding UTF8

try {
    foreach ($name in $configFiles) {
        $path = Join-Path $configDirectory $name
        if (Test-Path $path -PathType Leaf) {
            Copy-Item $path (Join-Path $configBackup $name)
            $backedUp[$name] = $true
        } else {
            $backedUp[$name] = $false
        }
    }
    New-Item -ItemType Directory -Path $configDirectory -Force | Out-Null

    $port = Get-Random -Minimum 18000 -Maximum 28000
    $prefix = "http://127.0.0.1:$port/"
    $requestLog = Join-Path $configBackup 'requests.log'
    $serverJob = Start-Job -ArgumentList $prefix, $requestLog -ScriptBlock {
        param($jobPrefix, $jobRequestLog)
        $server = [System.Net.HttpListener]::new()
        $server.Prefixes.Add($jobPrefix)
        $server.Start()
        Set-Content -Path $jobRequestLog -Value 'READY' -Encoding ASCII
        while ($server.IsListening) {
            try {
                $context = $server.GetContext()
                $context.Response.StatusCode = 200
                $bytes = [Text.Encoding]::UTF8.GetBytes('https://files.example.test/native-smoke')
                $context.Response.OutputStream.Write($bytes, 0, $bytes.Length)
                $context.Response.Close()
                Add-Content -Path $jobRequestLog -Value 'REQUEST'
            } catch {
                break
            }
        }
        $server.Close()
    }
    $serverReady = $false
    for ($attempt = 0; $attempt -lt 50; $attempt++) {
        Start-Sleep -Milliseconds 100
        if (Test-Path $requestLog -PathType Leaf -and (Get-Content $requestLog) -contains 'READY') {
            $serverReady = $true
            break
        }
        if ((Get-Job -Id $serverJob.Id).State -eq 'Failed') { throw 'Local smoke endpoint failed to start.' }
    }
    if (-not $serverReady) { throw 'Timed out waiting for local smoke endpoint.' }

    $config = @"
{
  "version": 2,
  "defaultUploader": "smoke",
  "defaultShortener": "",
  "copyToClipboard": false
}
"@
    $uploader = @"
{
  "version": 2,
  "uploaders": {
    "smoke": {
      "request": { "method": "POST", "url": "${prefix}upload", "body": "binary" },
      "response": { "url": { "type": "body" } }
    }
  }
}
"@
    Set-Content -Path (Join-Path $configDirectory 'config.json') -Value $config -Encoding UTF8
    Set-Content -Path (Join-Path $configDirectory 'custom-uploader.json') -Value $uploader -Encoding UTF8
    Remove-Item (Join-Path $configDirectory 'custom-shortener.json') -Force -ErrorAction SilentlyContinue

    $env:UPIT_FILE_MANAGER_HEADLESS = '1'
    $helper = Start-Process -FilePath (Join-Path $payloadRoot 'upit-file-manager.exe') -ArgumentList (Join-Path $fixtureDirectory 'upit-smoke.txt') -PassThru -NoNewWindow
    if (-not $helper.WaitForExit(30000)) {
        Stop-Process -Id $helper.Id -Force
        throw 'One-shot helper did not exit after the local endpoint completed.'
    }
    if ($helper.ExitCode -ne 0) { throw "One-shot helper exited with code $($helper.ExitCode)." }
    $requestCount = @(Get-Content $requestLog | Where-Object { $_ -eq 'REQUEST' }).Count
    if ($requestCount -lt 1) { throw 'The local endpoint received no helper request.' }
    $automated.helperUpload = $true
    $automated.endpointRequests = $requestCount
    $automated.helperExited = $true

    $cli = Start-Process -FilePath (Join-Path $payloadRoot 'upit.exe') -ArgumentList '--help' -PassThru -Wait -NoNewWindow
    if ($cli.ExitCode -ne 0) { throw "CLI help exited with code $($cli.ExitCode)." }
    $automated.cliHelp = $true

    Start-Process explorer.exe $fixtureDirectory | Out-Null
    Write-Host 'Interactive smoke required on this protected Windows 11 x64 runner:'
    Write-Host '  1. Verify Upload with Upit appears in the primary menu for exactly one regular file.'
    Write-Host '  2. Verify folders and multi-selection do not expose or invoke the command.'
    Write-Host '  3. Exercise local success, warning, cancellation, explicit Retry, Copy, and configuration recovery.'
    Write-Host '  4. Verify no Upit Desktop window opens during direct upload.'
    $checks = [ordered]@{
        primaryContextMenu = 'primary File Explorer menu observed for one regular file'
        foldersAndMultiSelectionHidden = 'folder and multi-selection command suppression observed'
        lifecycle = 'success, warning, cancellation, Retry, Copy, and configuration recovery observed'
        noDesktopWindow = 'Upit Desktop did not open during direct upload'
    }
    foreach ($name in $checks.Keys) {
        $answer = Read-Host "Type YES after: $($checks[$name])"
        if ($answer -cne 'YES') { throw "Operator did not confirm $name." }
        $operatorObserved[$name] = $true
    }

    $evidence = [ordered]@{
        package = (Split-Path $PackagePath -Leaf)
        packageSha256 = (Get-FileHash -Algorithm SHA256 $PackagePath).Hash.ToLowerInvariant()
        osBuild = $os.BuildNumber
        architecture = $env:PROCESSOR_ARCHITECTURE
        automated = $automated
        operatorObserved = $operatorObserved
        timestampUtc = [DateTime]::UtcNow.ToString('O')
        operator = [Environment]::UserName
        machine = [Environment]::MachineName
    }
    $evidence | ConvertTo-Json -Depth 5 | Set-Content -Path $EvidencePath -Encoding UTF8
} finally {
    Remove-Item Env:UPIT_FILE_MANAGER_HEADLESS -ErrorAction SilentlyContinue
    if ($null -ne $serverJob) {
        Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
        Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
    }
    if ($null -ne $helper -and -not $helper.HasExited) { Stop-Process -Id $helper.Id -Force -ErrorAction SilentlyContinue }
    Remove-Item $fixtureDirectory -Recurse -Force -ErrorAction SilentlyContinue
    foreach ($name in $configFiles) {
        $path = Join-Path $configDirectory $name
        if ($backedUp.ContainsKey($name) -and $backedUp[$name]) {
            Copy-Item (Join-Path $configBackup $name) $path -Force
        } else {
            Remove-Item $path -Force -ErrorAction SilentlyContinue
        }
    }
    Remove-Item $configBackup -Recurse -Force -ErrorAction SilentlyContinue
    & (Join-Path $PSScriptRoot 'uninstall.ps1')
    if (Get-AppxPackage -Name 'HungNth.Upit') { throw 'Uninstall left package identity or Explorer registration behind.' }
}
