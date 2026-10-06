[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)] [string] $InstallerPath,
    [Parameter(Mandatory = $true)] [string] $InstallDirectory,
    [string] $EvidencePath = (Join-Path (Get-Location) 'native-smoke-evidence.json'),
    [switch] $AutomatedOnly,
    [switch] $RequireSignature,
    [string] $LegacyInstallerPath,
    [switch] $VerifyLegacyMigration,
    [switch] $FreshInstallOnly,
    [string] $Format0InstallerPath,
    [string] $LegacyRandomInstallerPath,
    [string] $PreviousInstallerPath,
    [string] $CleanupInstallerPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Import-Module (Join-Path $PSHOME 'Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1') -ErrorAction Stop
Import-Module (Join-Path $PSHOME 'Modules\Microsoft.PowerShell.Security\Microsoft.PowerShell.Security.psd1') -ErrorAction Stop

if (-not ([System.Management.Automation.PSTypeName]'PeIconValidator').Type) {
    Add-Type -Path (Join-Path $PSScriptRoot 'PeIconValidator.cs')
}

$os = Get-CimInstance Win32_OperatingSystem
if ([int]$os.BuildNumber -lt 22000) { throw "Windows 11 build required; received $($os.BuildNumber)." }
if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw "Windows x64 runner required; received $env:PROCESSOR_ARCHITECTURE." }

$isWorkstation = ([int]$os.ProductType -eq 1)
if (-not $AutomatedOnly) {
    if (-not $isWorkstation) {
        throw "Interactive native smoke requires Windows 11 Workstation (ProductType 1); received ProductType $($os.ProductType) ($($os.Caption)). Server runners may only run -AutomatedOnly."
    }
    if (-not [Environment]::UserInteractive) {
        throw "Interactive native smoke requires an interactive user desktop session."
    }
}

$classicVerbKey = 'HKCU:\Software\Classes\*\shell\Upit.Upload'
$classicVerbCommandKey = 'HKCU:\Software\Classes\*\shell\Upit.Upload\command'
$productKey = 'HKCU:\Software\Upit'
$uninstallKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit'

if (-not (Test-Path -LiteralPath $InstallerPath -PathType Leaf)) {
    throw "Installer not found: $InstallerPath"
}
$installerResolved = (Resolve-Path -LiteralPath $InstallerPath).Path
$installerLeaf = Split-Path -Leaf $installerResolved
$installerHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $installerResolved).Hash.ToLowerInvariant()
$signature = Get-AuthenticodeSignature -LiteralPath $installerResolved
$signatureStatus = [string]$signature.Status
$isSigned = ($signature.Status -eq 'Valid')

if ($RequireSignature -and -not $isSigned) {
    throw "Installer signature is required (-RequireSignature) but signature status is '$signatureStatus'."
}

$versionInfo = (Get-Item -LiteralPath $installerResolved).VersionInfo
$artifactProductVersion = [string]$versionInfo.ProductVersion
$artifactFileVersion = [string]$versionInfo.FileVersion

$hasExistingProduct = Test-Path -LiteralPath $productKey
$hasExistingVerb = Test-Path -LiteralPath $classicVerbKey
$hasExistingUninstall = Test-Path -LiteralPath $uninstallKey
$hasExistingLegacy = @(Get-AppxPackage -Name 'HungNth.Upit').Count -gt 0

$legacyMigrationRequested = [bool]($LegacyInstallerPath -or $VerifyLegacyMigration)
$existingMigration = $legacyMigrationRequested -and $hasExistingLegacy -and -not $hasExistingVerb

if (-not $legacyMigrationRequested -and $hasExistingLegacy) {
    throw "An existing legacy HungNth.Upit package was detected. Smoke requires a clean runner unless testing legacy migration (-VerifyLegacyMigration or -LegacyInstallerPath)."
}

if (($hasExistingProduct -or $hasExistingVerb -or $hasExistingUninstall) -and -not $existingMigration) {
    throw "An existing Upit installation was detected (Product: $hasExistingProduct, Verb: $hasExistingVerb, Uninstall: $hasExistingUninstall). Native smoke refuses to overwrite an existing installation without refusal."
}
if (-not $existingMigration -and (Test-Path -LiteralPath $InstallDirectory) -and (Get-ChildItem -LiteralPath $InstallDirectory -Force | Select-Object -First 1)) {
    throw "Install directory already exists and is not empty: $InstallDirectory. Native smoke refuses to overwrite an existing installation without refusal."
}
if ($existingMigration -and $hasExistingProduct) {
    $oldPayload = (Get-Item -LiteralPath $productKey).GetValue('PayloadPath')
    $oldRoot = [IO.Path]::GetFullPath((Split-Path (Split-Path $oldPayload)))
    if (-not [string]::Equals($oldRoot, [IO.Path]::GetFullPath($InstallDirectory), [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Existing legacy product must belong to the requested smoke install directory.'
    }
}

$fixtureDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("upit-native-smoke " + [guid]::NewGuid().ToString('N'))
$configDirectory = Join-Path $HOME '.config\upit'
$configBackupDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("upit-config-backup-" + [guid]::NewGuid().ToString('N'))
$backedUpFiles = @{}
$configDirectoryExisted = $false
$configurationChanged = $false
$backupComplete = $false
$originalFiles = @{}

$serverJob = $null
$helper = $null
$desktopProc = $null
$uninstallerRunner = Join-Path $fixtureDirectory 'uninstall-smoke.exe'
$smokeSuccess = $false
$smokeError = $null
$smokeNotes = ''

$priorHeadless = $env:UPIT_FILE_MANAGER_HEADLESS
$userPathBefore = [Environment]::GetEnvironmentVariable('Path', 'User')
$automated = [ordered]@{}
$operatorObserved = [ordered]@{}
$operatorChecksExercised = $false
$legacyPackageObserved = $false
$legacyMigrationExercised = $false
$legacyMigrationRemovedOldPackage = $false
$fullAcceptance = $false

function Get-RegistryState([string] $path) {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($path)
    if ($null -eq $key) { return $null }
    try {
        $values = [ordered]@{}
        foreach ($name in @($key.GetValueNames() | Sort-Object)) {
            $values[$name] = @([string]$key.GetValueKind($name), $key.GetValue($name, $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames))
        }
        $children = [ordered]@{}
        foreach ($name in @($key.GetSubKeyNames() | Sort-Object)) {
            $children[$name] = Get-RegistryState ($path + '\' + $name)
        }
        return [ordered]@{ values = $values; children = $children }
    } finally { $key.Close() }
}

function Get-CallableState([string] $root) {
    $files = [ordered]@{}
    foreach ($path in @((Join-Path $root 'Uninstall.exe'), (Join-Path $root 'upit.exe'), (Join-Path ([Environment]::GetFolderPath('Programs')) 'Upit.lnk'))) {
        $files[$path] = if (Test-Path -LiteralPath $path -PathType Leaf) { (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash } else { $null }
    }
    return ([ordered]@{
        product = Get-RegistryState 'Software\Upit'
        verb = Get-RegistryState 'Software\Classes\*\shell\Upit.Upload'
        uninstall = Get-RegistryState 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit'
        files = $files
        userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    } | ConvertTo-Json -Depth 20 -Compress)
}

function Normalize-WindowsPath([string] $path) {
    return $path.Trim().Replace('/', '\').TrimEnd('\').ToLowerInvariant()
}

function Invoke-InstalledCLI([string[]] $arguments, [string] $workingDirectory) {
    $quoted = @($arguments | ForEach-Object { "'" + $_.Replace("'", "''") + "'" }) -join ','
    $script = '[Console]::OutputEncoding = [Text.Encoding]::UTF8; & upit @(' + $quoted + '); exit $LASTEXITCODE'
    $process = [Diagnostics.Process]::new()
    $process.StartInfo.FileName = (Get-Command powershell.exe -ErrorAction Stop).Source
    $process.StartInfo.Arguments = '-NoProfile -NonInteractive -EncodedCommand ' + [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($script))
    $process.StartInfo.UseShellExecute = $false
    $process.StartInfo.CreateNoWindow = $true
    $process.StartInfo.RedirectStandardOutput = $true
    $process.StartInfo.RedirectStandardError = $true
    $process.StartInfo.WorkingDirectory = $workingDirectory
    $process.StartInfo.EnvironmentVariables['PATH'] = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
    try {
        $process.Start() | Out-Null
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit(30000)) { throw 'Stable CLI invocation timed out.' }
        return @{ exit = $process.ExitCode; stdout = $stdout.GetAwaiter().GetResult(); stderr = $stderr.GetAwaiter().GetResult() }
    } finally { $process.Dispose() }
}

function Invoke-ConsoleHelper([string] $script) {
    $script = '$ErrorActionPreference="Stop"; $ProgressPreference="SilentlyContinue"; try { ' + $script + '; exit 0 } catch { [Console]::Error.WriteLine($_.Exception.ToString()); exit 1 }'
    $process = [Diagnostics.Process]::new()
    $process.StartInfo.FileName = (Get-Command powershell.exe -ErrorAction Stop).Source
    $process.StartInfo.Arguments = '-NoProfile -NonInteractive -EncodedCommand ' + [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($script))
    $process.StartInfo.UseShellExecute = $false
    $process.StartInfo.CreateNoWindow = $true
    $process.StartInfo.RedirectStandardOutput = $true
    $process.StartInfo.RedirectStandardError = $true
    try {
        $process.Start() | Out-Null
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit(20000)) { throw 'Real console helper timed out.' }
        return @{ exit = $process.ExitCode; stdout = $stdout.GetAwaiter().GetResult(); stderr = $stderr.GetAwaiter().GetResult() }
    } finally { $process.Dispose() }
}

function Assert-ClassicRegistration([string] $targetInstallDir) {
    if (-not (Test-Path -LiteralPath $productKey)) {
        throw "Active product metadata key not found: $productKey"
    }
    $productItem = Get-Item -LiteralPath $productKey
    $activePayload = $productItem.GetValue('PayloadPath')
    if ([string]::IsNullOrWhiteSpace($activePayload) -or -not (Test-Path -LiteralPath $activePayload -PathType Container)) {
        throw "Active PayloadPath is missing or invalid: '$activePayload'"
    }
    if (-not $activePayload.StartsWith($targetInstallDir, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Active PayloadPath '$activePayload' does not reside under expected install directory '$targetInstallDir'."
    }
    $requiredPayload = @('upit.exe', 'upit-desktop.exe', 'upit-file-manager.exe')
    foreach ($file in $requiredPayload) {
        $filePath = Join-Path $activePayload $file
        if (-not (Test-Path -LiteralPath $filePath -PathType Leaf)) {
            throw "Required payload file missing: $file at '$filePath'"
        }
    }
    $obsoleteFiles = @('upit-explorer-command.dll', 'repair\Upit.msix')
    foreach ($file in $obsoleteFiles) {
        $filePath = Join-Path $activePayload $file
        if (Test-Path -LiteralPath $filePath) {
            throw "Obsolete payload file found: $file at '$filePath'"
        }
    }

    if (-not (Test-Path -LiteralPath $classicVerbKey)) {
        throw "Classic Verb key missing: $classicVerbKey"
    }
    $verbItem = Get-Item -LiteralPath $classicVerbKey
    if ($verbItem.GetValue('') -ne 'Upload with Upit') {
        throw "Classic Verb default value mismatch: expected 'Upload with Upit', got '$($verbItem.GetValue(''))'"
    }
    $expectedIcon = '"' + (Join-Path $activePayload 'upit-desktop.exe') + '",0'
    if ($verbItem.GetValue('Icon') -ne $expectedIcon) {
        throw "Classic Verb Icon mismatch: expected '$expectedIcon', got '$($verbItem.GetValue('Icon'))'"
    }
    if ($verbItem.GetValue('MultiSelectModel') -ne 'Single') {
        throw "Classic Verb MultiSelectModel mismatch: expected 'Single', got '$($verbItem.GetValue('MultiSelectModel'))'"
    }

    if (-not (Test-Path -LiteralPath $classicVerbCommandKey)) {
        throw "Classic Verb command key missing: $classicVerbCommandKey"
    }
    $commandItem = Get-Item -LiteralPath $classicVerbCommandKey
    $expectedCommand = '"' + (Join-Path $activePayload 'upit-file-manager.exe') + '" "%1"'
    if ($commandItem.GetValue('') -cne $expectedCommand) {
        throw "Classic Verb command mismatch: expected '$expectedCommand', got '$($commandItem.GetValue(''))'"
    }

    $legacyPackages = @(Get-AppxPackage -Name 'HungNth.Upit')
    if ($legacyPackages.Count -ne 0) {
        throw "Legacy HungNth.Upit package is still registered ($($legacyPackages.Count) found)."
    }

    return $activePayload
}

try {
    New-Item -ItemType Directory -Path $fixtureDirectory -Force | Out-Null
    New-Item -ItemType Directory -Path $configBackupDirectory -Force | Out-Null

    $fixtureFile = Join-Path $fixtureDirectory 'upit smoke fixture.txt'
    Set-Content -LiteralPath $fixtureFile -Value 'Windows native smoke fixture' -Encoding UTF8

    # 1. Configuration backup
    $configDirectoryExisted = Test-Path -LiteralPath $configDirectory -PathType Container
    foreach ($name in @('config.json', 'custom-uploader.json', 'custom-shortener.json')) {
        $source = Join-Path $configDirectory $name
        $backedUpFiles[$name] = Test-Path -LiteralPath $source -PathType Leaf
        if ($backedUpFiles[$name]) {
            [IO.File]::Copy($source, (Join-Path $configBackupDirectory $name), $true)
        }
    }
    $backupComplete = $true
    if (-not $legacyMigrationRequested) {
        New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
        $blockedLauncher = Join-Path $InstallDirectory 'upit.exe'
        [IO.File]::WriteAllBytes($blockedLauncher, [byte[]]@())
        $freshBefore = Get-CallableState $InstallDirectory
        $publicationLock = [IO.File]::Open($blockedLauncher, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
        try {
            $failedFresh = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
            if (-not $failedFresh.WaitForExit(60000)) { throw 'Failed fresh-publication fixture timed out.' }
            if ($failedFresh.ExitCode -eq 0) { throw 'Fresh install succeeded despite an unavailable stable launcher path.' }
            if ((Get-CallableState $InstallDirectory) -cne $freshBefore) { throw 'Failed fresh install left new routes, PATH, metadata, or modified user files.' }
            $retainedExecutables = @(Get-ChildItem -LiteralPath $InstallDirectory -Recurse -Filter '*.exe' | Where-Object { $_.FullName -ne $blockedLauncher })
            if ($retainedExecutables.Count -ne 0) { throw 'Successfully compensated fresh install retained an incomplete payload.' }
            $automated.failedFreshPublicationRestoresState = $true
        } finally { $publicationLock.Dispose() }
        Remove-Item -LiteralPath $InstallDirectory -Recurse -Force
    }

    # 2. Migration mode or clean installation
    $payloadRoot1 = $null
    if ($legacyMigrationRequested) {
        if (-not [string]::IsNullOrWhiteSpace($LegacyInstallerPath)) {
            if (-not (Test-Path -LiteralPath $LegacyInstallerPath -PathType Leaf)) {
                throw "Legacy installer not found: $LegacyInstallerPath"
            }
            $legacyInstallerResolved = (Resolve-Path -LiteralPath $LegacyInstallerPath).Path
            $legacyProc = Start-Process -FilePath $legacyInstallerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
            if (-not $legacyProc.WaitForExit(60000)) {
                throw 'Legacy installer timed out.'
            }
            if ($legacyProc.ExitCode -ne 0) {
                throw "Legacy installer failed with exit code $($legacyProc.ExitCode)."
            }
        }

        $preMigrationPackages = @(Get-AppxPackage -Name 'HungNth.Upit')
        if ($preMigrationPackages.Count -eq 0) {
            throw 'Real registered old HungNth.Upit package was NOT observed before Classic update. Cannot verify legacy migration.'
        }
        $legacyPackageObserved = $true

        $classicProc = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $classicProc.WaitForExit(60000)) {
            throw 'Classic installer update from legacy timed out.'
        }
        if ($classicProc.ExitCode -ne 0) {
            throw "Classic installer update from legacy failed with exit code $($classicProc.ExitCode)."
        }

        $postMigrationPackages = @(Get-AppxPackage -Name 'HungNth.Upit')
        if ($postMigrationPackages.Count -ne 0) {
            throw "Legacy HungNth.Upit package was still registered after Classic update ($($postMigrationPackages.Count) found)."
        }
        $legacyMigrationRemovedOldPackage = $true
        $legacyMigrationExercised = $true
        $payloadRoot1 = Assert-ClassicRegistration $InstallDirectory
    } else {
        $installerProc = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $installerProc.WaitForExit(60000)) {
            throw 'Installer timed out.'
        }
        if ($installerProc.ExitCode -ne 0) {
            throw "Installer exited with code $($installerProc.ExitCode)."
        }
        $payloadRoot1 = Assert-ClassicRegistration $InstallDirectory
    }
    $automated.firstInstall = $true
    $automated.registryVerb = $true
    $automated.registryCommand = $true
    $automated.payloadIntact = $true
    $automated.obsoletePayloadAbsent = $true
    $expectedPayload = Join-Path (Join-Path $InstallDirectory 'versions') $artifactProductVersion
    if ($payloadRoot1 -cne $expectedPayload) { throw 'Active Payload is not the requested readable version directory.' }
    foreach ($name in @('upit.exe', 'upit-desktop.exe', 'upit-file-manager.exe')) {
        if ((Get-Item -LiteralPath (Join-Path $payloadRoot1 $name)).VersionInfo.ProductVersion -ne $artifactProductVersion) { throw "Incoherent payload product version: $name" }
    }
    $stableLauncher = Join-Path $InstallDirectory 'upit.exe'
    if (-not (Test-Path -LiteralPath $stableLauncher -PathType Leaf)) { throw 'Stable root launcher is missing.' }
    $automated.readableVersionLayout = $true
    $userPathAfter = [Environment]::GetEnvironmentVariable('Path', 'User')
    $rootNormalized = Normalize-WindowsPath $InstallDirectory
    $priorRootEntries = @($userPathBefore -split ';' | Where-Object { (Normalize-WindowsPath $_) -eq $rootNormalized })
    $newRootEntries = @($userPathAfter -split ';' | Where-Object { (Normalize-WindowsPath $_) -eq $rootNormalized })
    if ($newRootEntries.Count -ne [Math]::Max(1, $priorRootEntries.Count)) { throw 'Installer added duplicate equivalent stable-root PATH entries.' }
    $unrelatedBefore = @($userPathBefore -split ';' | Where-Object { (Normalize-WindowsPath $_) -ne $rootNormalized }) -join ';'
    $unrelatedAfter = @($userPathAfter -split ';' | Where-Object { (Normalize-WindowsPath $_) -ne $rootNormalized }) -join ';'
    if ($unrelatedBefore -cne $unrelatedAfter) { throw 'Installation modified unrelated PATH entry text or order.' }
    $expectedPathOwned = if ($priorRootEntries.Count -eq 0) { 1 } else { 0 }
    if ((Get-Item -LiteralPath $productKey).GetValue('PathEntryOwned') -ne $expectedPathOwned) { throw 'Installer did not record correct PATH ownership.' }
    $automated.stablePathOwnership = $true
    $automated.legacyPackageAbsent = $true
    $startMenuShortcut = Join-Path ([Environment]::GetFolderPath('Programs')) 'Upit.lnk'
    if (-not (Test-Path -LiteralPath $startMenuShortcut)) {
        throw 'Start Menu shortcut Upit.lnk missing after installation.'
    }
    $uninstallerPath = Join-Path $InstallDirectory 'Uninstall.exe'
    if (-not (Test-Path -LiteralPath $uninstallerPath)) {
        throw 'Uninstaller Uninstall.exe missing after installation.'
    }
    $desktopInstalled = Join-Path $payloadRoot1 'upit-desktop.exe'
    if (-not [PeIconValidator]::HasGroupIcon($desktopInstalled)) {
        throw "Installed desktop executable lacks RT_GROUP_ICON resource: $desktopInstalled"
    }
    if (-not [PeIconValidator]::HasGroupIcon($uninstallerPath)) {
        throw "Installed uninstaller lacks RT_GROUP_ICON resource: $uninstallerPath"
    }
    [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
    $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('HungNth.Upit')
    if ($null -eq $notifier -or $notifier.Setting -ne [Windows.UI.Notifications.NotificationSetting]::Enabled) {
        throw "WinRT ToastNotifier setting is not Enabled for HungNth.Upit: $($notifier.Setting)"
    }
    $automated.startMenuShortcutPresent = $true
    $automated.winrtNotifierEnabled = $true

    # 3. Local endpoint configuration and execution
    $port = Get-Random -Minimum 18000 -Maximum 28000
    $prefix = "http://127.0.0.1:${port}/"
    $requestLog = Join-Path $fixtureDirectory 'requests.log'
    $responseDelay = if ($AutomatedOnly) { 500 } else { 3000 }
    $serverJob = Start-Job -ArgumentList $prefix, $requestLog, $responseDelay -ScriptBlock {
        param($jobPrefix, $jobRequestLog, $responseDelay)
        $server = [System.Net.HttpListener]::new()
        $server.Prefixes.Add($jobPrefix)
        $server.Start()
        Set-Content -LiteralPath $jobRequestLog -Value 'READY' -Encoding ASCII
        try {
            while ($server.IsListening) {
                $pending = $server.GetContextAsync()
                while (-not $pending.IsCompleted) { Start-Sleep -Milliseconds 50 }
                $context = $pending.GetAwaiter().GetResult()
                Add-Content -LiteralPath $jobRequestLog -Value 'REQUEST'
                if ($context.Request.RawUrl -eq '/cancel') { Start-Sleep -Seconds 10 }
                elseif ($context.Request.RawUrl -eq '/hold') { Start-Sleep -Seconds 180 }
                else { Start-Sleep -Milliseconds $responseDelay }
                try {
                    $context.Response.StatusCode = 200
                    $bytes = [Text.Encoding]::UTF8.GetBytes('https://files.example.test/native-smoke')
                    $context.Response.OutputStream.Write($bytes, 0, $bytes.Length)
                } catch [System.Net.HttpListenerException], [System.IO.IOException] {
                    if ($context.Request.RawUrl -ne '/cancel') { throw }
                } finally { $context.Response.Close() }
            }
        } finally { $server.Close() }
    }
    $serverReady = $false
    for ($attempt = 0; $attempt -lt 50; $attempt++) {
        Start-Sleep -Milliseconds 100
        if ((Test-Path -LiteralPath $requestLog -PathType Leaf) -and (Get-Content -LiteralPath $requestLog) -contains 'READY') {
            $serverReady = $true
            break
        }
        if ((Get-Job -Id $serverJob.Id).State -eq 'Failed') { throw 'Local smoke endpoint failed to start.' }
    }
    if (-not $serverReady) { throw 'Timed out waiting for local smoke endpoint.' }

    New-Item -ItemType Directory -Path $configDirectory -Force | Out-Null
    $configurationChanged = $true
    # Same-directory rename preserves bytes, ownership, DACLs, and file attributes.
    foreach ($name in $backedUpFiles.Keys) {
        if ($backedUpFiles[$name]) {
            $original = Join-Path $configDirectory ('.upit-smoke-original-' + [guid]::NewGuid().ToString('N'))
            [IO.File]::Move((Join-Path $configDirectory $name), $original)
            $originalFiles[$name] = $original
        }
    }
    $configContent = @"
{
  "version": 2,
  "defaultUploader": "smoke",
  "defaultShortener": "",
  "copyToClipboard": false
}
"@
    $uploaderContent = @"
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
    [IO.File]::WriteAllText((Join-Path $configDirectory 'config.json'), $configContent)
    [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $uploaderContent)
    Remove-Item -LiteralPath (Join-Path $configDirectory 'custom-shortener.json') -Force -ErrorAction SilentlyContinue

    Set-Clipboard -Value 'before-smoke-clipboard'
    $env:UPIT_FILE_MANAGER_HEADLESS = '1'
    # Own the native process handle: PowerShell 5.1 NoNewWindow can lose ExitCode.
    $helper = [Diagnostics.Process]::new()
    $helper.StartInfo.FileName = Join-Path $payloadRoot1 'upit-file-manager.exe'
    $helper.StartInfo.Arguments = '"' + $fixtureFile + '"'
    $helper.StartInfo.UseShellExecute = $false
    $helper.StartInfo.CreateNoWindow = $true
    $helper.Start() | Out-Null
    if (-not $helper.WaitForExit(30000)) {
        Stop-Process -Id $helper.Id -Force
        throw 'One-shot helper did not exit after the local endpoint completed.'
    }
    if ($helper.ExitCode -ne 0) { throw "One-shot helper exited with code $($helper.ExitCode)." }
    $requestCount = @(Get-Content -LiteralPath $requestLog | Where-Object { $_ -eq 'REQUEST' }).Count
    if ($requestCount -ne 1) { throw "The one-shot upload received $requestCount requests, expected 1." }
    $automated.helperUpload = $true
    $automated.endpointRequests = $requestCount
    $automated.helperExited = $true

    $clipboardText = (Get-Clipboard -Raw).Trim()
    if ($clipboardText -ne 'https://files.example.test/native-smoke') {
        throw 'File Manager Upload did not copy the Final URL with the preference disabled.'
    }
    $automated.alwaysCopy = $true

    Remove-Item Env:UPIT_FILE_MANAGER_HEADLESS -ErrorAction SilentlyContinue

    # 4. CLI continuity
    $helpResult = Invoke-InstalledCLI @('--help') $fixtureDirectory
    if ($helpResult.exit -ne 0 -or $helpResult.stdout -notmatch 'upit upload' -or $helpResult.stderr.Trim()) { throw 'New terminal cannot invoke stable CLI help by command name.' }
    $automated.cliHelp = $true
    $unicodeFile = Join-Path $fixtureDirectory ('upload ' + [char]0x7A7A + [char]0x95F4 + ' file.txt')
    Copy-Item -LiteralPath $fixtureFile -Destination $unicodeFile
    $uploadResult = Invoke-InstalledCLI @('upload', (Split-Path -Leaf $unicodeFile)) $fixtureDirectory
    if ($uploadResult.exit -ne 0 -or $uploadResult.stdout.Trim() -ne 'https://files.example.test/native-smoke' -or $uploadResult.stderr.Trim()) { throw 'Stable CLI failed relative Unicode-path upload or output-channel preservation.' }
    if (@(Get-Content -LiteralPath $requestLog | Where-Object { $_ -eq 'REQUEST' }).Count -ne 2) { throw 'Stable CLI did not perform exactly one additional upload.' }
    $failureResult = Invoke-InstalledCLI @('upload', 'missing smoke file.txt') $fixtureDirectory
    if ($failureResult.exit -eq 0 -or -not $failureResult.stderr.Trim() -or $failureResult.stdout.Trim()) { throw 'Stable CLI lost nonzero status or error-channel isolation.' }
    $automated.relativeUnicodeArguments = $true
    $automated.nonzeroExitAndStderr = $true
    $automated.cliUpload = $true
    $metadata = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Upit', $true)
    try {
        foreach ($invalid in @($fixtureDirectory, (Join-Path $InstallDirectory 'versions\999.999.999'), (Join-Path $InstallDirectory 'staging\invalid'), (Join-Path $InstallDirectory 'recovery\invalid'))) {
            $metadata.SetValue('PayloadPath', $invalid, [Microsoft.Win32.RegistryValueKind]::String)
            $rejected = Invoke-InstalledCLI @('--help') $fixtureDirectory
            if ($rejected.exit -eq 0 -or -not $rejected.stderr.Trim()) { throw 'Launcher selected a payload despite invalid active metadata.' }
        }
        $metadata.DeleteValue('PayloadPath')
        $rejected = Invoke-InstalledCLI @('--help') $fixtureDirectory
        if ($rejected.exit -eq 0) { throw 'Launcher guessed a version when Active Payload metadata was absent.' }
    } finally { $metadata.SetValue('PayloadPath', $payloadRoot1, [Microsoft.Win32.RegistryValueKind]::String); $metadata.Close() }
    $automated.invalidActivePayloadRejected = $true
    $cancelUploader = $uploaderContent.Replace("${prefix}upload", "${prefix}cancel")
    [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $cancelUploader)
    try {
        $cancelHelper = (Join-Path $PSScriptRoot 'ConsoleCancellationSmoke.cs').Replace("'", "''")
        $cancelCommand = "Add-Type -Path '$cancelHelper'; [ConsoleCancellationSmoke]::CancelUpload('" + $stableLauncher.Replace("'", "''") + "','" + $fixtureFile.Replace("'", "''") + "','" + $fixtureDirectory.Replace("'", "''") + "') | Out-Null"
        $cancelResult = Invoke-ConsoleHelper $cancelCommand
        if ($cancelResult.exit -ne 0) { throw ('Console cancellation harness failed: ' + $cancelResult.stderr) }
        $remainingCLI = @(Get-CimInstance Win32_Process -Filter "Name='upit.exe'" | Where-Object { $_.ExecutablePath -and [string]::Equals($_.ExecutablePath, (Join-Path $payloadRoot1 'upit.exe'), [StringComparison]::OrdinalIgnoreCase) })
        if ($remainingCLI.Count -ne 0) { throw 'Console cancellation orphaned the real payload CLI.' }
        if (@(Get-Content -LiteralPath $requestLog | Where-Object { $_ -eq 'REQUEST' }).Count -ne 3) { throw 'Cancellation did not exercise an in-flight real upload.' }
        $automated.consoleCancellationNoOrphan = $true
    } finally { [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $uploaderContent) }

    # 5. Interactive observations (if not -AutomatedOnly)
    if (-not $AutomatedOnly) {
        Start-Process explorer.exe -ArgumentList "`"$fixtureDirectory`""
        $desktopProc = Start-Process -FilePath (Join-Path $payloadRoot1 'upit-desktop.exe') -PassThru

        Write-Host 'Interactive smoke required on this Windows 11 x64 workstation:'
        Write-Host '  1. In File Explorer, right click upit smoke fixture.txt -> Show more options (Shift+F10).'
        Write-Host '     Verify "Upload with Upit" appears with Upit icon for exactly one regular file.'
        Write-Host '  2. Verify folders and multi-selection do not expose or invoke Upload with Upit.'
        Write-Host '  3. Invoke Upload with Upit: verify direct helper launch without opening Upit Desktop.'
        Write-Host '  4. Confirm clean success closes progress Task Dialog, copies Final URL, and emits silent native Toast "Upload complete".'
        Write-Host '  5. Damage registration (e.g. rename command subkey), verify Desktop reports Needs Repair, click Repair, confirm Registered.'
        Write-Host '  6. Verify manual upload in Desktop and CLI upload operate normally against the same Configuration Set.'

        $interactivePrompts = [ordered]@{
            classicShowMoreOptionsOneFile = 'Upload with Upit appears under Show more options (or Shift+F10) for exactly one regular file'
            foldersAndMultiSelectionHidden = 'Upload with Upit is absent for folders, folder backgrounds, and multi-file selection'
            registeredCommandInvocation = 'invoking Upload with Upit executes the registered command directly without opening Desktop'
            toastCleanSuccess = 'clean upload closes progress Task Dialog, copies Final URL to clipboard, and displays silent native Toast with title "Upload complete" and body "Final URL copied to clipboard."'
            noModalDialogOnCleanSuccess = 'clean success leaves NO modal Task Dialog or Message Box'
            warningRemainsInteractive = 'Shortener or clipboard warning remains interactive and visible (not silent)'
            failureRemainsInteractive = 'network or endpoint failure remains interactive with Retry option'
            cancellationRemainsInteractive = 'cancelling progress Task Dialog stops upload without retry and remains interactive'
            recoveryActionsRemainInteractive = 'Copy Final URL, Retry, Open Upit, and configuration recovery remain interactive'
            privacyNoSensitiveData = 'all native feedback omits file names, paths, endpoints, URLs, credentials, and action tokens'
            noResidentProcess = 'File Manager Upload leaves no resident worker process, tray icon, or background daemon'
            desktopStatesAndExplicitRepair = 'Desktop distinguishes Registered, Needs Repair, and Reinstall Upit; explicit Repair restores damaged registration'
            manualUploadAndCliContinuity = 'Manual Upload in Desktop and CLI upload operate normally against the same Configuration Set'
        }

        foreach ($name in $interactivePrompts.Keys) {
            $answer = Read-Host "Type YES after verifying: $($interactivePrompts[$name])"
            if ($answer -cne 'YES') { throw "Operator did not confirm observation: $name ($($interactivePrompts[$name]))" }
            $operatorObserved[$name] = $true
        }
        $operatorChecksExercised = $true
        $fullAcceptance = $legacyMigrationExercised -and $legacyPackageObserved -and $legacyMigrationRemovedOldPackage

        if ($null -ne $desktopProc -and -not $desktopProc.HasExited) {
            Stop-Process -Id $desktopProc.Id -Force -ErrorAction SilentlyContinue
        }
    } else {
        $operatorChecksExercised = $false
        $fullAcceptance = $false
    }

    if ($PreviousInstallerPath -or $CleanupInstallerPath) {
        if (-not $PreviousInstallerPath -or -not $CleanupInstallerPath) { throw 'Different-version smoke requires both previous and cleanup installer artifacts.' }
        foreach ($artifact in @($PreviousInstallerPath, $CleanupInstallerPath)) {
            if (-not (Test-Path -LiteralPath $artifact -PathType Leaf)) { throw "Update fixture artifact is missing: $artifact" }
            if ((Get-Item -LiteralPath $artifact).VersionInfo.ProductVersion -eq $artifactProductVersion) { throw 'Update fixture versions must differ from the tested product version.' }
        }
        $previousInstall = Start-Process -FilePath $PreviousInstallerPath -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $previousInstall.WaitForExit(60000) -or $previousInstall.ExitCode -ne 0) { throw 'Real prior-version fixture installation failed.' }
        $oldPayload = Assert-ClassicRegistration $InstallDirectory
        $launcherBeforeUpdate = (Get-FileHash -LiteralPath $stableLauncher -Algorithm SHA256).Hash
        if (-not ([System.Management.Automation.PSTypeName]'ConsoleCancellationSmoke').Type) { Add-Type -Path (Join-Path $PSScriptRoot 'ConsoleCancellationSmoke.cs') }
        [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $uploaderContent.Replace("${prefix}upload", "${prefix}hold"))
        $oldLauncherPID = [ConsoleCancellationSmoke]::StartUpload($stableLauncher, $fixtureFile, $fixtureDirectory)
        $oldLauncherProcess = [Diagnostics.Process]::GetProcessById($oldLauncherPID)
        try {
            $oldPayloadProcess = $null
            for ($attempt = 0; $attempt -lt 40; $attempt++) {
                $oldPayloadProcess = @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$oldLauncherPID" | Where-Object { $_.ExecutablePath -and [string]::Equals($_.ExecutablePath, (Join-Path $oldPayload 'upit.exe'), [StringComparison]::OrdinalIgnoreCase) })
                if ($oldPayloadProcess.Count -eq 1) { break }
                Start-Sleep -Milliseconds 100
            }
            if ($oldPayloadProcess.Count -ne 1) { throw 'Prior-version real CLI did not enter the running payload.' }
            $oldPayloadPID = $oldPayloadProcess[0].ProcessId
            $differentUpdate = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
            if (-not $differentUpdate.WaitForExit(60000) -or $differentUpdate.ExitCode -ne 0) { throw 'Different-version update did not commit while the old launcher and payload were running.' }
            $payloadRoot1 = Assert-ClassicRegistration $InstallDirectory
            if ($payloadRoot1 -ne $expectedPayload) { throw 'New invocations were not routed to the tested version.' }
            if ($oldLauncherProcess.HasExited -or -not (Get-Process -Id $oldPayloadPID -ErrorAction SilentlyContinue)) { throw 'Different-version update terminated the original CLI process.' }
            if ((Get-FileHash -LiteralPath $stableLauncher -Algorithm SHA256).Hash -ne $launcherBeforeUpdate) { throw 'Ordinary update replaced an adequate running launcher.' }
            if (-not (Test-Path -LiteralPath $oldPayload) -or @((Get-Item -LiteralPath $productKey).GetValue('InactivePayloads')) -notcontains $oldPayload) { throw 'Locked former payload was not recorded as inactive deferred cleanup.' }
            $newHelp = Invoke-InstalledCLI @('--help') $fixtureDirectory
            if ($newHelp.exit -ne 0) { throw 'New stable CLI invocations failed after committed update.' }
            $automated.committedUpdateWithDeferredCleanup = $true
            $automated.runningOldPayloadSurvives = $true
            $automated.adequateRunningLauncherPreserved = $true
        } finally {
            if (-not $oldLauncherProcess.HasExited) {
                $cancelHelper = (Join-Path $PSScriptRoot 'ConsoleCancellationSmoke.cs').Replace("'", "''")
                $stopScript = "`$ErrorActionPreference='Stop'; Add-Type -Path '$cancelHelper'; [ConsoleCancellationSmoke]::CancelProcess($oldLauncherPID) | Out-Null"
                $stopResult = Invoke-ConsoleHelper $stopScript
                if ($stopResult.exit -ne 0) { throw ('Unable to close old CLI fixture through console cancellation: ' + $stopResult.stderr) }
            }
            $oldLauncherProcess.Dispose()
            [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $uploaderContent)
        }
        $cleanupInstall = Start-Process -FilePath $CleanupInstallerPath -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $cleanupInstall.WaitForExit(60000) -or $cleanupInstall.ExitCode -ne 0) { throw 'Later different-version installation failed.' }
        $payloadRoot1 = Assert-ClassicRegistration $InstallDirectory
        $cleanupVersion = (Get-Item -LiteralPath $CleanupInstallerPath).VersionInfo.ProductVersion
        if ($payloadRoot1 -ne (Join-Path (Join-Path $InstallDirectory 'versions') $cleanupVersion)) { throw 'Cleanup changed the selected Active Payload unexpectedly.' }
        if (Test-Path -LiteralPath $oldPayload) { throw 'Later install did not remove the unlocked inactive payload.' }
        if (@((Get-Item -LiteralPath $productKey).GetValue('InactivePayloads')) -contains $oldPayload) { throw 'Removed inactive payload remains recorded for cleanup.' }
        $automated.laterInstallRetriesDeferredCleanup = $true
        $automated.cleanCommittedDifferentVersionUpdate = $true
        # Re-run installerResolved to establish payloadRoot1 at artifactProductVersion before same-version tests
        $reinstallCurrent = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $reinstallCurrent.WaitForExit(60000) -or $reinstallCurrent.ExitCode -ne 0) { throw 'Resetting to tested version failed.' }
        $payloadRoot1 = Assert-ClassicRegistration $InstallDirectory
    }

    $payloadRoot2 = $payloadRoot1
    if (-not $FreshInstallOnly) {
    # A locked cleanup worker must abort before replacing the callable route.
    $oldCommand = (Get-Item -LiteralPath $classicVerbCommandKey).GetValue('')
    $stateBeforeFailure = Get-CallableState $InstallDirectory
    $uninstallerLock = [IO.File]::Open((Join-Path $InstallDirectory 'Uninstall.exe'), [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
    try {
        $failedUpdate = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $failedUpdate.WaitForExit(60000)) { throw 'Locked-uninstaller update timed out.' }
        if ($failedUpdate.ExitCode -eq 0) { throw 'Update reported success with an unavailable cleanup worker.' }
        $activeAfterFailure = Assert-ClassicRegistration $InstallDirectory
        if ($activeAfterFailure -ne $payloadRoot1 -or (Get-Item -LiteralPath $classicVerbCommandKey).GetValue('') -ne $oldCommand) {
            throw 'Failed uninstaller publication replaced the previous active route.'
        }
        if ((Get-CallableState $InstallDirectory) -cne $stateBeforeFailure) {
            throw 'Failed publication did not restore exact product, verb, uninstall, shortcut, uninstaller, and User PATH state.'
        }
        $automated.failedUninstallerPublicationRetainsRoute = $true
    } finally { $uninstallerLock.Dispose() }

    # 6. Same-version update & launcher format replacement
    # A. Active payload process causes silent same-version install to abort nonzero without state mutation
    if (-not ([System.Management.Automation.PSTypeName]'ConsoleCancellationSmoke').Type) { Add-Type -Path (Join-Path $PSScriptRoot 'ConsoleCancellationSmoke.cs') }
    [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $uploaderContent.Replace("${prefix}upload", "${prefix}hold"))
    $activeUploadPID = [ConsoleCancellationSmoke]::StartUpload($stableLauncher, $fixtureFile, $fixtureDirectory)
    $activeUploadProc = [Diagnostics.Process]::GetProcessById($activeUploadPID)
        for ($attempt = 0; $attempt -lt 40; $attempt++) {
            $childProcs = @(Get-CimInstance Win32_Process -Filter "ParentProcessId=$activeUploadPID" | Where-Object { $_.ExecutablePath -and [string]::Equals($_.ExecutablePath, (Join-Path $payloadRoot1 'upit.exe'), [StringComparison]::OrdinalIgnoreCase) })
            if ($childProcs.Count -eq 1) { break }
            Start-Sleep -Milliseconds 100
        }
    try {
        $sameVerProcBlocked = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $sameVerProcBlocked.WaitForExit(60000)) { throw 'Silent same-version update timed out while process active.' }
        if ($sameVerProcBlocked.ExitCode -eq 0) { throw 'Silent same-version update succeeded despite active payload process.' }
        if ($payloadRoot1 -ne (Assert-ClassicRegistration $InstallDirectory)) { throw 'Blocked same-version update altered active payload route.' }
        $automated.silentSameVersionAbortsWhenInUse = $true
    } finally {
        if (-not $activeUploadProc.HasExited) {
            $cancelHelper = (Join-Path $PSScriptRoot 'ConsoleCancellationSmoke.cs').Replace("'", "''")
            $stopScript = "`$ErrorActionPreference='Stop'; Add-Type -Path '$cancelHelper'; [ConsoleCancellationSmoke]::CancelProcess($activeUploadPID) | Out-Null"
            $stopResult = Invoke-ConsoleHelper $stopScript
        }
        $activeUploadProc.Dispose()
        [IO.File]::WriteAllText((Join-Path $configDirectory 'custom-uploader.json'), $uploaderContent)
    }

    # B. Same-version update after process closes succeeds, retaining the same versions\X.Y.Z path
    $updateProc = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
    if (-not $updateProc.WaitForExit(60000) -or $updateProc.ExitCode -ne 0) { throw 'Same-version update failed after process closure.' }
    $payloadRoot2 = Assert-ClassicRegistration $InstallDirectory
    if ($payloadRoot1 -ne $payloadRoot2) {
        throw "Same-version update changed payload path from $payloadRoot1 to $payloadRoot2; expected unchanged path per ADR 0023."
    }
    if (@(Get-ChildItem (Join-Path $InstallDirectory 'recovery') -ErrorAction SilentlyContinue).Count -ne 0) {
        throw 'Same-version update left recovery directory behind.'
    }
    $automated.sameVersionUpdate = $true

    # C. Launcher-format replacement if Format0InstallerPath provided
    if ($Format0InstallerPath -and (Test-Path -LiteralPath $Format0InstallerPath -PathType Leaf)) {
        $format0Install = Start-Process -FilePath $Format0InstallerPath -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $format0Install.WaitForExit(60000) -or $format0Install.ExitCode -ne 0) { throw 'Format-0 installer failed to install.' }
        $format0Val = (& $stableLauncher --launcher-format).Trim()
        if ($format0Val -ne '0') { throw "Expected format 0 launcher, got '$format0Val'." }
        $format1Update = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $format1Update.WaitForExit(60000) -or $format1Update.ExitCode -ne 0) { throw 'Format-1 update failed.' }
        $format1Val = (& $stableLauncher --launcher-format).Trim()
        if ($format1Val -ne '1') { throw "Expected format 1 launcher after update, got '$format1Val'." }
        $automated.launcherFormatReplacementVerified = $true
    }
    # 7. Real legacy random .tmp layout cutover
    if ($LegacyRandomInstallerPath -and (Test-Path -LiteralPath $LegacyRandomInstallerPath -PathType Leaf)) {
        # Clear test directory and install real legacy random .tmp build
        if (Test-Path -LiteralPath $InstallDirectory) { Remove-Item -LiteralPath $InstallDirectory -Recurse -Force }
        $legacySetupProc = Start-Process -FilePath $LegacyRandomInstallerPath -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $legacySetupProc.WaitForExit(60000) -or $legacySetupProc.ExitCode -ne 0) { throw 'Legacy random setup install failed.' }
        $legacyRecordedPayload = (Get-Item -LiteralPath $productKey).GetValue('PayloadPath')
        if ($legacyRecordedPayload -notmatch '\\versions\\[^\\]+\.tmp$') { throw "Recorded legacy payload does not match expected .tmp layout: $legacyRecordedPayload" }
        # Simulate user manually adding legacy random payload to User PATH
        $userPathWithLegacy = [Environment]::GetEnvironmentVariable('Path', 'User') + ';' + $legacyRecordedPayload
        [Environment]::SetEnvironmentVariable('Path', $userPathWithLegacy, 'User')
        # Run new stable installer over legacy installation
        $cutoverProc = Start-Process -FilePath $installerResolved -ArgumentList "/S /D=$InstallDirectory" -PassThru
        if (-not $cutoverProc.WaitForExit(60000) -or $cutoverProc.ExitCode -ne 0) { throw 'Cutover installer from legacy failed.' }
        $cutoverPayload = Assert-ClassicRegistration $InstallDirectory
        if ($cutoverPayload -ne $expectedPayload) { throw "Cutover did not switch to expected version layout: $cutoverPayload" }
        if (-not (Test-Path -LiteralPath $stableLauncher)) { throw 'Cutover did not publish stable launcher.' }
        # Verify user-owned legacy PATH entry is preserved untouched and stable root is added
        $userPathAfterCutover = [Environment]::GetEnvironmentVariable('Path', 'User')
        if ($userPathAfterCutover -notmatch [regex]::Escape($legacyRecordedPayload)) { throw 'Installer improperly removed or modified user-owned legacy PATH entry.' }
        if ($userPathAfterCutover -notmatch [regex]::Escape($InstallDirectory)) { throw 'Installer failed to add stable root to User PATH during cutover.' }
        if ((Get-Item -LiteralPath $productKey).GetValue('PathEntryOwned') -ne 1) { throw 'Installer did not record ownership for newly added stable root.' }
        # Verify old random payload directory is deleted post-commit
        if (Test-Path -LiteralPath $legacyRecordedPayload) { throw 'Old random .tmp payload was not deleted after successful cutover.' }
        $automated.legacyLayoutCutoverVerified = $true
        $automated.userOwnedLegacyPathPreserved = $true
        # Clean up test-only user PATH entry so final uninstall assertion tests original userPathBefore
        $userPathWithoutLegacy = [Environment]::GetEnvironmentVariable('Path', 'User').Replace(';' + $legacyRecordedPayload, '').Replace($legacyRecordedPayload + ';', '')
        [Environment]::SetEnvironmentVariable('Path', $userPathWithoutLegacy, 'User')
    }
    }

    # 7. Fail-closed uninstall (ACL injection)
    Copy-Item -LiteralPath (Join-Path $InstallDirectory 'Uninstall.exe') -Destination $uninstallerRunner
    $aclKey = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Classes\*\shell\Upit.Upload', [Microsoft.Win32.RegistryKeyPermissionCheck]::ReadWriteSubTree, ([Security.AccessControl.RegistryRights]::ReadKey -bor [Security.AccessControl.RegistryRights]::ChangePermissions))
    if ($null -eq $aclKey) {
        throw 'Classic Verb registry key unavailable for ACL injection.'
    }
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().User
    $denyRule = [Security.AccessControl.RegistryAccessRule]::new($identity, [Security.AccessControl.RegistryRights]::Delete, [Security.AccessControl.InheritanceFlags]::None, [Security.AccessControl.PropagationFlags]::None, [Security.AccessControl.AccessControlType]::Deny)
    $deniedAcl = $aclKey.GetAccessControl()
    $deniedAcl.AddAccessRule($denyRule)
    try {
        $aclKey.SetAccessControl($deniedAcl)
        $uninstallerFailedProc = Start-Process -FilePath $uninstallerRunner -ArgumentList "/S _?=$InstallDirectory" -PassThru
        if (-not $uninstallerFailedProc.WaitForExit(60000)) {
            throw 'Fail-closed uninstaller timed out.'
        }
        if ($uninstallerFailedProc.ExitCode -eq 0) {
            throw 'Fail-closed uninstaller reported success despite injected registry deletion failure.'
        }
        if (-not (Test-Path -LiteralPath $payloadRoot2)) {
            throw 'Fail-closed uninstaller deleted payload despite failure.'
        }
        if (-not (Test-Path -LiteralPath $productKey)) {
            throw 'Fail-closed uninstaller deleted product metadata key despite failure.'
        }
        if (-not (Test-Path -LiteralPath $stableLauncher)) {
            throw 'Fail-closed uninstaller deleted stable launcher despite failure.'
        }
        $automated.uninstallFailureRetainsPayload = $true
    } finally {
        try {
            $freshAcl = $aclKey.GetAccessControl()
            $freshAcl.RemoveAccessRuleSpecific($denyRule)
            $aclKey.SetAccessControl($freshAcl)
        } finally {
            $aclKey.Close()
        }
    }
    # 8. Clean uninstall
    $uninstallerCleanProc = Start-Process -FilePath $uninstallerRunner -ArgumentList "/S _?=$InstallDirectory" -PassThru
    if (-not $uninstallerCleanProc.WaitForExit(60000)) {
        throw 'Clean uninstaller timed out.'
    }
    if ($uninstallerCleanProc.ExitCode -ne 0) {
        throw "Clean uninstaller exited with code $($uninstallerCleanProc.ExitCode)."
    }
    if (Test-Path -LiteralPath $classicVerbKey) {
        throw 'Uninstall left Classic Verb registration behind.'
    }
    if (Test-Path -LiteralPath $productKey) {
        throw 'Uninstall left Software\Upit product key behind.'
    }
    if (Test-Path -LiteralPath $uninstallKey) {
        throw 'Uninstall left Windows Uninstall metadata key behind.'
    }
    if (Test-Path -LiteralPath $payloadRoot2) {
        throw 'Uninstall left payload directory behind.'
    }
    if (Test-Path -LiteralPath $stableLauncher) { throw 'Uninstall left the stable launcher behind.' }
    if ([Environment]::GetEnvironmentVariable('Path', 'User') -cne $userPathBefore) { throw 'Uninstall did not preserve the original User PATH.' }
    $automated.ownedPathRemovedOrUserPathPreserved = $true
    $automated.uninstallClean = $true
    $automated.uninstallRegistrationClean = $true

    $smokeSuccess = $true
    if ($AutomatedOnly) {
        $smokeNotes = 'Automated installer and CLI checks passed; automatedChecks records the exercised scenarios. Operator Explorer UI observations were not exercised.'
    } else {
        $smokeNotes = 'Interactive surface observations and automated lifecycle checks passed. Full acceptance additionally requires exercised real old-package migration.'
    }
} catch {
    $smokeSuccess = $false
    $smokeError = 'Smoke failed; inspect the command error locally. Private paths and values are omitted from evidence.'
    $fullAcceptance = $false
    $smokeNotes = 'Smoke terminated before acceptance; only recorded completed checks are evidence.'
    throw
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
    if ($null -ne $helper -and -not $helper.HasExited) {
        Stop-Process -Id $helper.Id -Force -ErrorAction SilentlyContinue
    }
    if ($null -ne $helper) { $helper.Dispose() }
    if ($null -ne $desktopProc -and -not $desktopProc.HasExited) {
        Stop-Process -Id $desktopProc.Id -Force -ErrorAction SilentlyContinue
    }

    Remove-Item -LiteralPath $fixtureDirectory -Recurse -Force -ErrorAction SilentlyContinue

    if ($configurationChanged -and $backupComplete) {
        foreach ($name in $backedUpFiles.Keys) {
            $destination = Join-Path $configDirectory $name
            if ($originalFiles.ContainsKey($name)) {
                Remove-Item -LiteralPath $destination -Force -ErrorAction SilentlyContinue
                [IO.File]::Move($originalFiles[$name], $destination)
            } elseif (-not $backedUpFiles[$name]) {
                Remove-Item -LiteralPath $destination -Force -ErrorAction SilentlyContinue
            }
        }
        if (-not $configDirectoryExisted -and @(Get-ChildItem -LiteralPath $configDirectory -Force).Count -eq 0) {
            Remove-Item -LiteralPath $configDirectory -Force
        }
    }
    Remove-Item -LiteralPath $configBackupDirectory -Recurse -Force -ErrorAction SilentlyContinue

    if (-not $automated.Contains('uninstallClean') -or -not $automated.uninstallClean) {
        $uninstallerPath = Join-Path $InstallDirectory 'Uninstall.exe'
        if (Test-Path -LiteralPath $uninstallerPath) {
            try {
                $recoveryRunner = Join-Path ([IO.Path]::GetTempPath()) ('upit-smoke-uninstall-' + [guid]::NewGuid().ToString('N') + '.exe')
                Copy-Item -LiteralPath $uninstallerPath -Destination $recoveryRunner
                $recoveryUninstaller = Start-Process -FilePath $recoveryRunner -ArgumentList "/S _?=$InstallDirectory" -PassThru
                $recoveryUninstaller.WaitForExit(60000) | Out-Null
            } catch {} finally {
                if ($null -ne $recoveryRunner) { Remove-Item -LiteralPath $recoveryRunner -Force -ErrorAction SilentlyContinue }
            }
        }
    }

    $evidence = [ordered]@{
        installer = $installerLeaf
        installerSha256 = $installerHash
        artifactVersion = $artifactProductVersion
        signed = $isSigned
        signatureStatus = $signatureStatus
        signatureRequired = [bool]$RequireSignature
        osCaption = $os.Caption
        osProductType = [int]$os.ProductType
        osBuild = [int]$os.BuildNumber
        architecture = $env:PROCESSOR_ARCHITECTURE
        mode = if ($AutomatedOnly) { 'automatedOnly' } else { 'interactive' }
        evidenceType = if ($AutomatedOnly) { 'automatedSubset' } elseif ($isSigned) { 'signedInteractiveSmoke' } else { 'unsignedInteractiveSmoke' }
        automatedChecks = $automated
        operatorObserved = $operatorObserved
        operatorChecksExercised = [bool]$operatorChecksExercised
        legacyMigrationExercised = [bool]$legacyMigrationExercised
        legacyPackageObserved = [bool]$legacyPackageObserved
        fullAcceptance = [bool]$fullAcceptance
        authoritativeProductionProof = [bool]($fullAcceptance -and $operatorChecksExercised -and $isWorkstation)
        success = [bool]$smokeSuccess
        error = $smokeError
        notes = $smokeNotes
        timestampUtc = [DateTime]::UtcNow.ToString('O')
    }

    $evidenceDir = Split-Path -Parent $EvidencePath
    if (-not [string]::IsNullOrWhiteSpace($evidenceDir) -and -not (Test-Path -LiteralPath $evidenceDir)) {
        New-Item -ItemType Directory -Path $evidenceDir -Force | Out-Null
    }
    $evidence | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $EvidencePath -Encoding UTF8
}
