#define MyAppName "CC Switch CLI (ccs)"
#define DefaultAppVersion "2.0.0" ; keep in sync with internal/version/version.go
; MyAppVersion can be overridden by /DMyAppVersion or CCS_VERSION env var
#ifndef MyAppVersion
  #define MyAppVersion GetEnv("CCS_VERSION")
#endif

#if MyAppVersion == ""
  #undef MyAppVersion
  #define MyAppVersion DefaultAppVersion
#endif
#define MyAppPublisher "YangQing-Lin"
#define MyAppURL "https://github.com/YangQing-Lin/cc-switch-cli"
#define MyAppExeName "ccs-windows-amd64.exe"

[Setup]
AppId={{12B175B2-8904-4628-BE3E-F2CF6778837B}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={localappdata}\Programs\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
OutputDir=build
OutputBaseFilename=ccs-{#MyAppVersion}-Setup
SetupIconFile=build\windows\icon.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "build\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent
