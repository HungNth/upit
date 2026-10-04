Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

!ifndef UPIT_VERSION
!error "UPIT_VERSION is required"
!endif
!ifndef UPIT_TECHNICAL_VERSION
!error "UPIT_TECHNICAL_VERSION is required"
!endif
!ifndef UPIT_PAYLOAD
!error "UPIT_PAYLOAD is required"
!endif
!ifndef UPIT_OUTPUT
!error "UPIT_OUTPUT is required"
!endif

Name "Upit"
OutFile "${UPIT_OUTPUT}"
InstallDir "$LOCALAPPDATA\Programs\Upit"
RequestExecutionLevel user
VIProductVersion "${UPIT_TECHNICAL_VERSION}"
VIAddVersionKey "ProductName" "Upit"
VIAddVersionKey "FileDescription" "Upit Installer"
VIAddVersionKey "ProductVersion" "${UPIT_VERSION}"
VIAddVersionKey "FileVersion" "${UPIT_TECHNICAL_VERSION}"
VIAddVersionKey "LegalCopyright" "Upit contributors"

Var PreviousPath
Var PayloadPath
Var ExitCode
Var Output

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Upit"
 SetShellVarContext current
 ReadRegStr $PreviousPath HKCU "Software\Upit" "PayloadPath"
 ${If} $PreviousPath != ""
  ${GetParent} "$PreviousPath" $0
  ${If} $0 != "$INSTDIR\versions"
   MessageBox MB_ICONSTOP "The existing Upit installation path is invalid. Uninstall Upit before reinstalling."
   Abort
  ${EndIf}
 ${EndIf}
 CreateDirectory "$INSTDIR\versions"
 GetTempFileName $PayloadPath "$INSTDIR\versions"
 Delete "$PayloadPath"
 SetOutPath "$PayloadPath"
 File "${UPIT_PAYLOAD}\upit-desktop.exe"
 File "${UPIT_PAYLOAD}\upit-file-manager.exe"
 File "${UPIT_PAYLOAD}\upit-explorer-command.dll"
 File "${UPIT_PAYLOAD}\upit.exe"
 SetOutPath "$PayloadPath\repair"
 File /oname=Upit.msix "${UPIT_PAYLOAD}\repair\Upit.msix"
 nsExec::ExecToStack /TIMEOUT=120000 '"$PayloadPath\upit-file-manager.exe" --install-integration'
 Pop $ExitCode
 Pop $Output
 ${If} $ExitCode != 0
  ${If} $PreviousPath != ""
   nsExec::ExecToStack /TIMEOUT=120000 '"$PreviousPath\upit-file-manager.exe" --install-integration'
  ${Else}
   nsExec::ExecToStack /TIMEOUT=120000 '"$PayloadPath\upit-file-manager.exe" --uninstall-integration'
  ${EndIf}
  Pop $0
  Pop $1
  ${If} $0 == 0
   SetOutPath "$INSTDIR"
   RMDir /r "$PayloadPath"
   MessageBox MB_ICONSTOP "Upit registration failed. The previous registration was restored, or the failed first installation was unregistered. $Output"
  ${Else}
   MessageBox MB_ICONSTOP "Upit registration and rollback failed. Payload was retained at $PayloadPath. Reinstall Upit. $Output"
  ${EndIf}
  Abort
 ${EndIf}
 SetOutPath "$INSTDIR"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 WriteRegStr HKCU "Software\Upit" "PayloadPath" "$PayloadPath"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "DisplayName" "Upit"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "DisplayVersion" "${UPIT_VERSION}"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "NoModify" 1
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "NoRepair" 1
 CreateShortcut "$SMPROGRAMS\Upit.lnk" "$PayloadPath\upit-desktop.exe"
 ${If} $PreviousPath != ""
  RMDir /r /REBOOTOK "$PreviousPath"
 ${EndIf}
SectionEnd

Section "Uninstall"
 SetShellVarContext current
 InitPluginsDir
 SetOutPath "$PLUGINSDIR"
 File /oname=upit-file-manager.exe "${UPIT_PAYLOAD}\upit-file-manager.exe"
 nsExec::ExecToStack /TIMEOUT=120000 '"$PLUGINSDIR\upit-file-manager.exe" --uninstall-integration'
 Pop $ExitCode
 Pop $Output
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "Upit could not unregister File Manager Integration. Payload was retained. Reinstall Upit and retry removal. $Output"
  Abort
 ${EndIf}
 Delete "$SMPROGRAMS\Upit.lnk"
 DeleteRegKey HKCU "Software\Upit"
 DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit"
 RMDir /r /REBOOTOK "$INSTDIR\versions"
 Delete /REBOOTOK "$INSTDIR\Uninstall.exe"
 RMDir /REBOOTOK "$INSTDIR"
SectionEnd
