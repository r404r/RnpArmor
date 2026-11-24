#ifndef AppVersion
  #define AppVersion "dev"
#endif

#ifndef AppVersionInfo
  #define AppVersionInfo "0.0.0.0"
#endif

#ifndef SourceDir
  #define SourceDir "..\\..\\out\\dist\\rnparmor-windows-amd64"
#endif

#ifndef OutputDir
  #define OutputDir "..\\..\\out"
#endif

#ifndef OutputBaseFilename
  #define OutputBaseFilename "rnparmor-windows-amd64-setup"
#endif

[Setup]
AppId={{7D5D2E3B-4B94-4C93-9B2C-8F6D5A1C3E4A}
AppName=RnpArmor
AppVersion={#AppVersion}
AppPublisher=r404r
AppPublisherURL=https://github.com/r404r/RnpArmor
DefaultDirName={autopf}\RnpArmor
DefaultGroupName=RnpArmor
DisableProgramGroupPage=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
OutputDir={#OutputDir}
OutputBaseFilename={#OutputBaseFilename}
VersionInfoVersion={#AppVersionInfo}
UninstallDisplayName=RnpArmor
UninstallDisplayIcon={app}\rnparmor.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Files]
Source: "{#SourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\RnpArmor"; Filename: "{app}\rnparmor.exe"
Name: "{autodesktop}\RnpArmor"; Filename: "{app}\rnparmor.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\rnparmor.exe"; Description: "Launch RnpArmor"; Flags: nowait postinstall skipifsilent
