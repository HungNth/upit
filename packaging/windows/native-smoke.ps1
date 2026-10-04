[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $InstallerPath,
    [Parameter(Mandatory = $true)] [string] $InstallDirectory,
    [string] $EvidencePath = (Join-Path (Get-Location) 'native-smoke-evidence.json')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$os = Get-CimInstance Win32_OperatingSystem
if ([int]$os.BuildNumber -lt 22000) { throw "Windows 11 build required; received $($os.BuildNumber)." }
if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw "Windows x64 runner required; received $env:PROCESSOR_ARCHITECTURE." }
if (-not (Test-Path $InstallerPath -PathType Leaf)) { throw "Signed installer not found: $InstallerPath" }
if ((Get-AuthenticodeSignature -LiteralPath $InstallerPath).Status -ne 'Valid') { throw 'The consumer installer signature is not trusted.' }
if (Test-Path 'HKCU:\Software\Upit') { throw 'This smoke requires a clean dedicated runner with no existing Upit consumer installation.' }
$installer = Start-Process -FilePath $InstallerPath -ArgumentList "/S /D=$InstallDirectory" -PassThru -Wait
if ($installer.ExitCode -ne 0) { throw "Consumer installer failed with exit code $($installer.ExitCode)." }
$payloadRoot = (Get-ItemProperty 'HKCU:\Software\Upit').PayloadPath
$payloadNames = @('upit.exe', 'upit-desktop.exe', 'upit-file-manager.exe', 'upit-explorer-command.dll', 'repair\Upit.msix')
foreach ($name in $payloadNames) {
    if (-not (Test-Path (Join-Path $payloadRoot $name) -PathType Leaf)) { throw "Installed product is missing $name." }
}
$installed = Get-AppxPackage -Name 'HungNth.Upit'
if ($null -eq $installed) { throw 'Consumer installation did not register package identity.' }
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
$priorHeadless = $env:UPIT_FILE_MANAGER_HEADLESS
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
                Add-Content -Path $jobRequestLog -Value 'REQUEST'
                Start-Sleep -Seconds 2 # Leave time to observe progress or choose Cancel.
                $context.Response.StatusCode = 200
                $bytes = [Text.Encoding]::UTF8.GetBytes('https://files.example.test/native-smoke')
                $context.Response.OutputStream.Write($bytes, 0, $bytes.Length)
                $context.Response.Close()
            } catch {
                break
            }
        }
        $server.Close()
    }
    $serverReady = $false
    for ($attempt = 0; $attempt -lt 50; $attempt++) {
        Start-Sleep -Milliseconds 100
        if ((Test-Path $requestLog -PathType Leaf) -and (Get-Content $requestLog) -contains 'READY') {
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
    if ($requestCount -ne 1) { throw "The one-shot upload received $requestCount requests, expected one." }
    $automated.helperUpload = $true
    $automated.endpointRequests = $requestCount
    $automated.helperExited = $true
    if ((Get-Clipboard -Raw).Trim() -ne 'https://files.example.test/native-smoke') { throw 'File Manager Upload did not copy the Final URL with the preference disabled.' }
    $automated.alwaysCopy = $true

    # Always remove the test-only override during real native surface observations.
    # Restore the caller's environment only in finally, including an inherited value of 1.
    Remove-Item Env:UPIT_FILE_MANAGER_HEADLESS -ErrorAction SilentlyContinue

    $cli = Start-Process -FilePath (Join-Path $payloadRoot 'upit.exe') -ArgumentList '--help' -PassThru -Wait -NoNewWindow
    if ($cli.ExitCode -ne 0) { throw "CLI help exited with code $($cli.ExitCode)." }
    $automated.cliHelp = $true

    Start-Process explorer.exe $fixtureDirectory | Out-Null
    Write-Host 'Interactive smoke required on this Windows 11 x64 runner:'
    Write-Host '  1. Verify Upload with Upit appears in the primary context menu for exactly one regular file.'
    Write-Host '  2. Verify folders and multi-selection do not expose or invoke the command.'
    Write-Host '  3. Exercise real clean success from Explorer on upit-smoke.txt: progress Task Dialog closes, silent Windows Toast appears.'
    Write-Host '  4. Confirm Toast title is exactly "Upload complete" and body is exactly "Final URL copied to clipboard."'
    Write-Host '  5. Confirm Toast auto-hides/dismisses under standard OS control and clicking it triggers NO Upit action.'
    Write-Host '  6. Repeat upload for second clean success: confirm distinct second Toast without aggregation or replacement.'
    Write-Host '  7. Confirm clean success leaves NO modal Task Dialog or Message Box.'
    Write-Host '  8. Select a valid Shortener with an unreachable endpoint in Desktop; verify completed upload warning remains interactive, then restore the Shortener.'
    Write-Host '  9. Stop local endpoint or point uploader to unreachable port; verify failure feedback remains interactive and offers Retry.'
    Write-Host ' 10. Cancel during upload progress; verify feedback remains interactive without automatic retry.'
    Write-Host ' 11. Verify Copy Final URL, Retry, Open Upit, and configuration recovery remain interactive.'
    Write-Host ' 12. Verify no Upit Desktop window, tray icon, or resident worker process remains after direct upload.'
    Write-Host ' 13. Verify Desktop Repair status, atomic update, and Manual Upload + CLI continuity.'

    $checks = [ordered]@{
        primaryContextMenu = 'primary File Explorer menu observed for one regular file'
        foldersAndMultiSelectionHidden = 'folder and multi-selection command suppression observed'
        lifecycle = 'success, warning, cancellation, Retry, Copy, and configuration recovery observed'
        noDesktopWindow = 'Upit Desktop did not open during direct upload'
        cleanSuccessFirst = 'first Upload with Upit from Explorer: progress Task Dialog closed, silent Windows Toast appeared'
        cleanSuccessTitleBodyExact = 'Toast title is exactly "Upload complete" and body is exactly "Final URL copied to clipboard."'
        cleanSuccessSilent = 'Toast is silent and respected Windows notification and focus assist policy'
        cleanSuccessOSDismissal = 'Toast auto-hided or dismissed under standard Windows 11 notification center control'
        cleanSuccessNoActionOnSelection = 'selecting or clicking Toast triggered NO Upit action (no window, no retry, no URL open)'
        cleanSuccessSecondDistinctEvent = 'second sequential Upload with Upit showed a second distinct Toast without replacing or aggregating the first'
        noCleanSuccessModalFallback = 'clean success left NO modal Task Dialog, Message Box, or Desktop window'
        warningRemainsInteractive = 'Shortener or clipboard warning remained interactive and visible (not silent)'
        failureRemainsInteractive = 'runtime failure remained interactive with Task Dialog/Message Box and Retry'
        cancellationRemainsInteractive = 'cancelling progress Task Dialog stopped upload without retry and remained interactive'
        recoveryActionsRemainInteractive = 'Copy Final URL, Retry, Open Upit, and configuration recovery actions remained interactive'
        privacyNoSensitiveData = 'Toast and all native feedback omitted file names, paths, endpoints, URLs, request/response values, credentials, and action tokens'
        noResidentProcess = 'ordinary File Manager Upload opened no Upit Desktop window, tray icon, or resident worker'
        helperExitedImmediately = 'one-shot helper exited immediately after notification delivery / completion'
        integrationStatusAndRepair = 'Desktop reported truthful integration status; removing registration then choosing Repair restored it without elevation'
        unsafeRepairFailsClosed = 'missing, unsigned, untrusted, and mismatched repair inputs produced Reinstall Upit without registration changes'
        clipboardRecoveryWithoutReupload = 'clipboard failure preserved success and Copy Final URL reused the existing URL without another endpoint request'
        atomicUpdate = 'a signed product update preserved one identity and did not duplicate Explorer commands'
        manualAndCLIContinuity = 'Manual Upload and a CLI upload worked against the same Configuration Set'
    }
    foreach ($name in $checks.Keys) {
        $answer = Read-Host "Type YES after: $($checks[$name])"
        if ($answer -cne 'YES') { throw "Operator did not confirm $name." }
        $operatorObserved[$name] = $true
    }

    $evidence = [ordered]@{
        installer = (Split-Path $InstallerPath -Leaf)
        installerSha256 = (Get-FileHash -Algorithm SHA256 $InstallerPath).Hash.ToLowerInvariant()
        osBuild = $os.BuildNumber
        architecture = $env:PROCESSOR_ARCHITECTURE
        evidenceType = 'signedSupportingSmoke'
        authoritativeProductionProof = $false
        umbrellaReleaseGatePreserved = $true
        notes = 'Unsigned or local evidence does not prove publisher trust, MSIX package identity registration, or production readiness. Authoritative production proof requires the umbrella signed-release workflow execution.'
        automated = $automated
        operatorObserved = $operatorObserved
        timestampUtc = [DateTime]::UtcNow.ToString('O')
        operator = [Environment]::UserName
        machine = [Environment]::MachineName
    }
} finally {
    if ($null -ne $priorHeadless) {
        $env:UPIT_FILE_MANAGER_HEADLESS = $priorHeadless
    } else {
        Remove-Item Env:UPIT_FILE_MANAGER_HEADLESS -ErrorAction SilentlyContinue
    }
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
    $uninstaller = Start-Process -FilePath (Join-Path $InstallDirectory 'Uninstall.exe') -ArgumentList '/S' -PassThru -Wait
    if ($uninstaller.ExitCode -ne 0) { throw 'Consumer uninstall failed; retained payload requires manual recovery.' }
    if (Get-AppxPackage -Name 'HungNth.Upit') { throw 'Uninstall left package identity or Explorer registration behind.' }
    $automated.uninstallRegistrationClean = $true
}
$evidence | ConvertTo-Json -Depth 5 | Set-Content -Path $EvidencePath -Encoding UTF8
