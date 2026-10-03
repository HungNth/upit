package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

type windowsIntegrationAdapter struct{ executable func() (string, error) }

func init() { nativeIntegrationAdapter = windowsIntegrationAdapter{executable: os.Executable} }
func (a windowsIntegrationAdapter) run(ctx context.Context, action string) ([]byte, error) {
	executable, err := a.executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", windowsIntegrationScript)
	cmd.Env = append(os.Environ(), "UPIT_INTEGRATION_ROOT="+filepath.Dir(executable), "UPIT_INTEGRATION_ACTION="+action)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New("Windows integration inspection or registration failed; reinstall Upit")
	}
	return out, nil
}
func (a windowsIntegrationAdapter) inspect(ctx context.Context) (FileManagerIntegrationState, error) {
	out, err := a.run(ctx, "inspect")
	if err != nil {
		return FileManagerIntegrationState{}, err
	}
	var state FileManagerIntegrationState
	if err = json.Unmarshal(out, &state); err != nil {
		return state, errors.New("Windows returned invalid integration state")
	}
	return state, nil
}
func (a windowsIntegrationAdapter) act(ctx context.Context, action string) error {
	if action == "open-file-manager" {
		return exec.CommandContext(ctx, "explorer.exe").Start()
	}
	_, err := a.run(ctx, action)
	return err
}

const windowsIntegrationScript = `
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$root = [IO.Path]::GetFullPath($env:UPIT_INTEGRATION_ROOT)
$action = $env:UPIT_INTEGRATION_ACTION
$artifact = Join-Path $root 'repair\Upit.msix'
$state = @{status='Reinstall Upit';guidance='Reinstall Upit using the signed consumer installer. Verify Upload with Upit in File Explorer; registration does not prove menu visibility.';actions=@('open-file-manager')}
Add-Type -AssemblyName System.IO.Compression.FileSystem
function Inspect-TrustedPayload {
 if (-not (Test-Path -LiteralPath $artifact -PathType Leaf)) {return $false}
 $signature = Get-AuthenticodeSignature -LiteralPath $artifact
 if ($signature.Status -ne 'Valid' -or $null -eq $signature.SignerCertificate) {return $false}
 $zip = [IO.Compression.ZipFile]::OpenRead($artifact)
 try {
  $entry=$zip.GetEntry('AppxManifest.xml');if($null -eq $entry){return $false}
  $reader=[IO.StreamReader]::new($entry.Open());try{[xml]$manifest=$reader.ReadToEnd()}finally{$reader.Dispose()}
  $script:identity=$manifest.Package.Identity
  if($identity.Name -ne 'HungNth.Upit' -or $identity.ProcessorArchitecture -ne 'x64' -or $identity.Publisher -ne $signature.SignerCertificate.Subject){return $false}
  $entry=$zip.GetEntry('payload-sha256.json');if($null -eq $entry){return $false}
  $reader=[IO.StreamReader]::new($entry.Open());try{$hashes=$reader.ReadToEnd() | ConvertFrom-Json}finally{$reader.Dispose()}
  foreach($name in @('upit.exe','upit-desktop.exe','upit-file-manager.exe','upit-explorer-command.dll')) {
   $path=Join-Path $root $name
   if(-not (Test-Path -LiteralPath $path -PathType Leaf)){return $false}
   $payloadSignature=Get-AuthenticodeSignature -LiteralPath $path
   if($payloadSignature.Status -ne 'Valid' -or $payloadSignature.SignerCertificate.Thumbprint -ne $signature.SignerCertificate.Thumbprint){return $false}
   $expected=$hashes.PSObject.Properties[$name].Value
   if($expected -notmatch '^[0-9a-fA-F]{64}$' -or (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash -ne $expected){return $false}
  }
  return $true
 } finally {$zip.Dispose()}
}
try {$trusted=Inspect-TrustedPayload} catch {$trusted=$false}
$package=Get-AppxPackage -Name 'HungNth.Upit'
$registered=$false
if($trusted -and $null -ne $package -and $package.Status -eq 'Ok' -and $package.Version.ToString() -eq $identity.Version -and $package.Publisher -eq $identity.Publisher){
 [Windows.Management.Deployment.PackageManager,Windows.Management.Deployment,ContentType=WindowsRuntime] | Out-Null
 $manager=New-Object Windows.Management.Deployment.PackageManager
 $native=$manager.FindPackageForUser('', $package.PackageFullName)
 $registered=$null -ne $native.EffectiveExternalLocation -and [string]::Equals([IO.Path]::GetFullPath($native.EffectiveExternalLocation.Path),$root,[StringComparison]::OrdinalIgnoreCase)
}
if($action -eq 'remove-registration'){
 if($null -ne $package){Remove-AppxPackage -Package $package.PackageFullName}
 if($null -ne (Get-AppxPackage -Name 'HungNth.Upit')){throw 'Package registration remains after uninstall'}
 exit 0
}
if($trusted){
 $state.guidance='Verify Upload with Upit in the primary File Explorer menu. Registration does not prove live menu visibility.'
 if($registered){$state.status='Registered'}else{$state.status='Needs Repair';$state.actions+=@('repair')}
}
if($action -eq 'repair'){
 if(-not $trusted){throw 'Trusted repair inputs are unavailable'}
 Add-AppxPackage -Path $artifact -ExternalLocation $root
 exit 0
}
if($action -ne 'inspect'){throw 'Unknown integration action'}
$state | ConvertTo-Json -Compress
`
