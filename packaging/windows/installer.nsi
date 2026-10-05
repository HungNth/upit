Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "Win\COM.nsh"
!include "Win\Propkey.nsh"

!macro SetLnkAppUserModelId shortcut appid
  !insertmacro ComHlpr_CreateInProcInstance ${CLSID_ShellLink} ${IID_IShellLink} r0 ""
  ${If} $0 P<> 0
    ${IUnknown::QueryInterface} $0 '("${IID_IPersistFile}",.r1)'
    ${If} $1 P<> 0
      ${IPersistFile::Load} $1 '("${shortcut}", ${STGM_READWRITE})'
      ${IUnknown::QueryInterface} $0 '("${IID_IPropertyStore}",.r2)'
      ${If} $2 P<> 0
        System::Call 'Oleaut32::SysAllocString(w "${appid}") i.r3'
        System::Call '*${SYSSTRUCT_PROPERTYKEY}(${PKEY_AppUserModel_ID})p.r4'
        System::Call '*${SYSSTRUCT_PROPVARIANT}(${VT_BSTR},,&i4 $3)p.r5'
        ${IPropertyStore::SetValue} $2 '($4,$5)'

        System::Call 'Oleaut32::SysFreeString($3)'
        System::Free $4
        System::Free $5
        ${IPropertyStore::Commit} $2 ""
        ${IUnknown::Release} $2 ""
        ${IPersistFile::Save} $1 '("${shortcut}",1)'
      ${EndIf}
      ${IUnknown::Release} $1 ""
    ${EndIf}
    ${IUnknown::Release} $0 ""
  ${EndIf}
!macroend

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
!define MUI_ICON "..\..\assets\branding\app.ico"
!define MUI_UNICON "..\..\assets\branding\app.ico"

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
   MessageBox MB_ICONSTOP "The existing Upit installation path is invalid. Uninstall Upit before reinstalling." /SD IDOK
   SetErrorLevel 1
   Abort
  ${EndIf}
 ${EndIf}
 ClearErrors
 CreateDirectory "$INSTDIR\versions"
 GetTempFileName $PayloadPath "$INSTDIR\versions"
 ${If} ${Errors}
 ${OrIf} $PayloadPath == ""
  MessageBox MB_ICONSTOP "Could not initialize payload staging directory in $INSTDIR\versions." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 Delete "$PayloadPath"
 ClearErrors
 SetOutPath "$PayloadPath"
 File "${UPIT_PAYLOAD}\upit-desktop.exe"
 File "${UPIT_PAYLOAD}\upit-file-manager.exe"
 File "${UPIT_PAYLOAD}\upit.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit payload extraction failed. Staged payload was retained at $PayloadPath. Reinstall Upit." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ; Publish the current cleanup worker before removing the legacy package route.
 ; A failed write must not leave a new Classic Verb with a legacy-only uninstaller.
 SetOutPath "$INSTDIR"
 ClearErrors
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit could not publish its uninstaller. The previous registration and payload were retained." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 nsExec::ExecToStack /TIMEOUT=120000 '"$PayloadPath\upit-file-manager.exe" --install-integration'
 Pop $ExitCode
 Pop $Output
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "Upit registration failed. Staged payload was retained at $PayloadPath. $Output" /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 SetOutPath "$INSTDIR"
 ClearErrors
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "DisplayName" "Upit"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "DisplayVersion" "${UPIT_VERSION}"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "NoModify" 1
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit" "NoRepair" 1
 CreateShortcut "$SMPROGRAMS\Upit.lnk" "$PayloadPath\upit-desktop.exe"
 !insertmacro SetLnkAppUserModelId "$SMPROGRAMS\Upit.lnk" "HungNth.Upit"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit registration succeeded, but installer metadata or shortcut creation failed. Retained payloads at $PayloadPath and previous installation." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ${If} $PreviousPath != ""
  ${If} $PreviousPath != $PayloadPath
   ClearErrors
   RMDir /r /REBOOTOK "$PreviousPath"
   ${If} ${Errors}
    MessageBox MB_ICONSTOP "Upit update succeeded, but previous payload cleanup failed at $PreviousPath." /SD IDOK
    SetErrorLevel 1
    Abort
   ${EndIf}
  ${EndIf}
 ${EndIf}
SectionEnd

Section "Uninstall"
 SetShellVarContext current
 InitPluginsDir
 ClearErrors
 SetOutPath "$PLUGINSDIR"
 File /oname=upit-file-manager.exe "${UPIT_PAYLOAD}\upit-file-manager.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit could not extract uninstaller helper. Uninstall aborted." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 nsExec::ExecToStack /TIMEOUT=120000 '"$PLUGINSDIR\upit-file-manager.exe" --uninstall-integration'
 Pop $ExitCode
 Pop $Output
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "Upit could not unregister File Manager Integration. Payload was retained. Reinstall Upit and retry removal. $Output" /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 Delete "$SMPROGRAMS\Upit.lnk"
 DeleteRegKey HKCU "Software\Upit"
 DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit"
 RMDir /r /REBOOTOK "$INSTDIR\versions"
 Delete /REBOOTOK "$INSTDIR\Uninstall.exe"
 RMDir /REBOOTOK "$INSTDIR"
SectionEnd
