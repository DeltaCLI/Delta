; Delta CLI Windows Installer Script
; Inno Setup Compiler Script for Delta

#define MyAppName "Delta CLI"
#define MyAppVersion "0.4.5-alpha"
#define MyAppPublisher "Delta Task Force"
#define MyAppURL "https://github.com/deltacli/delta"
#define MyAppExeName "delta.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application.
; Do not use the same AppId value in installers for other applications.
AppId={{E5A3F9B7-2D4C-4B6E-8F9A-1C2D3E4F5A6B}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
;AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\Delta
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile=..\LICENSE.md
; Uncomment the following line to run in non administrative install mode (install for current user only.)
;PrivilegesRequired=lowest
OutputDir=..\build\installer
OutputBaseFilename=delta-setup-{#MyAppVersion}
SetupIconFile=delta.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ChangesEnvironment=yes
MinVersion=0,6.1

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "quicklaunchicon"; Description: "{cm:CreateQuickLaunchIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked; OnlyBelowVersion: 0,6.1
Name: "addtopath"; Description: "Add Delta to PATH environment variable"; GroupDescription: "System Integration:"; Flags: checkedonce

[Files]
Source: "..\build\windows\amd64\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\UserGuide.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\CHANGELOG.md"; DestDir: "{app}"; Flags: ignoreversion
; Include i18n files
Source: "..\i18n\*"; DestDir: "{app}\i18n"; Flags: ignoreversion recursesubdirs createallsubdirs
; Include templates
Source: "..\templates\*"; DestDir: "{app}\templates"; Flags: ignoreversion recursesubdirs createallsubdirs
; Include embedded patterns
Source: "..\embedded_patterns\*"; DestDir: "{app}\embedded_patterns"; Flags: ignoreversion recursesubdirs createallsubdirs
; NOTE: Don't use "Flags: ignoreversion" on any shared system files

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:ProgramOnTheWeb,{#MyAppName}}"; Filename: "{#MyAppURL}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon
Name: "{userappdata}\Microsoft\Internet Explorer\Quick Launch\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: quicklaunchicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
const
  EnvironmentKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';

procedure AddToPath();
var
  Path: string;
  DeltaPath: string;
begin
  DeltaPath := ExpandConstant('{app}');
  
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path) then
  begin
    if Pos(DeltaPath, Path) = 0 then
    begin
      Path := Path + ';' + DeltaPath;
      RegWriteStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path);
    end;
  end;
end;

procedure RemoveFromPath();
var
  Path: string;
  DeltaPath: string;
  P: Integer;
begin
  DeltaPath := ExpandConstant('{app}');
  
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path) then
  begin
    P := Pos(';' + DeltaPath, Path);
    if P = 0 then
    begin
      P := Pos(DeltaPath + ';', Path);
      if P > 0 then
        Delete(Path, P, Length(DeltaPath) + 1);
    end
    else
      Delete(Path, P, Length(DeltaPath) + 1);
      
    if P > 0 then
      RegWriteStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path);
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    if IsTaskSelected('addtopath') then
      AddToPath();
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RemoveFromPath();
end;

[UninstallDelete]
Type: filesandordirs; Name: "{userappdata}\delta"