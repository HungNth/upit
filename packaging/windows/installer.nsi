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
 CreateDirectory "$INSTDIR\staging"
 GetTempFileName $PayloadPath "$INSTDIR\staging"
 ${If} ${Errors}
 ${OrIf} $PayloadPath == ""
  MessageBox MB_ICONSTOP "Could not initialize payload staging directory in $INSTDIR\staging." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 Delete "$PayloadPath"
 ClearErrors
 CreateDirectory "$PayloadPath"
 SetOutPath "$PayloadPath"
 File "${UPIT_PAYLOAD}\upit-desktop.exe"
 File "${UPIT_PAYLOAD}\upit-file-manager.exe"
 File "${UPIT_PAYLOAD}\upit.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit payload extraction failed. Staged payload was retained at $PayloadPath. Reinstall Upit." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ; Stage uninstaller, shortcut, worker, and launcher to temporary plugins dir
 InitPluginsDir
 SetOutPath "$PLUGINSDIR"
 ClearErrors
 File /oname=upit-install.exe "${UPIT_PAYLOAD}\upit-install.exe"
 File /oname=upit-launcher.exe "${UPIT_PAYLOAD}\upit-launcher.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit could not extract installation worker. The previous registration and payload were retained." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ClearErrors
 WriteUninstaller "$PLUGINSDIR\Uninstall.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit could not create uninstaller. The previous registration and payload were retained." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ; Stage Start Menu shortcut in temporary plugins dir targeting final versions\version payload
 CreateShortcut "$PLUGINSDIR\Upit.lnk" "$INSTDIR\versions\${UPIT_VERSION}\upit-desktop.exe"
 !insertmacro SetLnkAppUserModelId "$PLUGINSDIR\Upit.lnk" "HungNth.Upit"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit could not stage Start Menu shortcut. The previous registration and payload were retained." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 SetOutPath "$INSTDIR"
 ; Invoke native lifecycle transaction worker from PLUGINSDIR (no timeout to prevent killing compensation)
 ${If} ${Silent}
  nsExec::ExecToStack '"$PLUGINSDIR\upit-install.exe" install --install-dir "$INSTDIR" --staged-payload "$PayloadPath" --staged-launcher "$PLUGINSDIR\upit-launcher.exe" --staged-uninstaller "$PLUGINSDIR\Uninstall.exe" --staged-shortcut "$PLUGINSDIR\Upit.lnk" --version "${UPIT_VERSION}" --silent'
 ${Else}
  nsExec::ExecToStack '"$PLUGINSDIR\upit-install.exe" install --install-dir "$INSTDIR" --staged-payload "$PayloadPath" --staged-launcher "$PLUGINSDIR\upit-launcher.exe" --staged-uninstaller "$PLUGINSDIR\Uninstall.exe" --staged-shortcut "$PLUGINSDIR\Upit.lnk" --version "${UPIT_VERSION}"'
 ${EndIf}
 Pop $ExitCode
 Pop $Output
 ${If} $ExitCode == 10
  ; Success with deferred cleanup of locked former payload (Ticket 03)
 ${ElseIf} $ExitCode != 0
  MessageBox MB_ICONSTOP "Upit installation failed. See transaction diagnostic; recovery material may be retained. $Output" /SD IDOK
  SetErrorLevel $ExitCode
  Abort
 ${EndIf}
 RMDir /r /REBOOTOK "$INSTDIR\staging"
SectionEnd

Section "Uninstall"
 SetShellVarContext current
 InitPluginsDir
 ClearErrors
 SetOutPath "$PLUGINSDIR"
 File /oname=upit-install.exe "${UPIT_PAYLOAD}\upit-install.exe"
 ${If} ${Errors}
  MessageBox MB_ICONSTOP "Upit could not extract uninstaller helper. Uninstall aborted." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 SetOutPath "$INSTDIR"
 ${If} ${Silent}
  nsExec::ExecToStack '"$PLUGINSDIR\upit-install.exe" uninstall --install-dir "$INSTDIR" --silent'
 ${Else}
  nsExec::ExecToStack '"$PLUGINSDIR\upit-install.exe" uninstall --install-dir "$INSTDIR"'
 ${EndIf}
 Pop $ExitCode
 Pop $Output
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "Upit could not unregister File Manager Integration. Payload was retained. Reinstall Upit and retry removal. $Output" /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
SectionEnd
