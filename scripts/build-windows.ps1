# Build the Windows x64 portable directory and zip on a machine with
# Visual Studio, CMake, Python, and Git. Does not publish a GitHub release.
# Requires the pins in deps/pins.json. Does not search PATH for librnp.
$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root
$pins = Get-Content -Raw -Path (Join-Path $root "deps\pins.json") | ConvertFrom-Json
if ($pins.rnp.commit -ne "aad1892e4d8423398a0b973bd0a7e4544c359afc") { throw "rnp pin mismatch" }
if ($pins.rnp.do_not_use -notcontains "v0.18.0") { throw "v0.18.0 must stay rejected" }

$src = Join-Path $root "out\src"
$deps = Join-Path $root "out\deps"
$work = Join-Path $root "out\win-build"
$dist = Join-Path $root "out\dist\rnparmor-windows-amd64"
$zip = Join-Path $root "out\rnparmor-windows-amd64.zip"
$report = Join-Path $root "out\windows-report.txt"
New-Item -ItemType Directory -Force -Path $src, $deps, $work | Out-Null

function Run([string]$File, [string[]]$Args) {
    Write-Host "+ $File $($Args -join ' ')"
    & $File @Args
    if ($LASTEXITCODE -ne 0) { throw "$File exited $LASTEXITCODE" }
}

function Get-Hash([string]$Path, [string]$Algo) {
    return (Get-FileHash -Algorithm $Algo -LiteralPath $Path).Hash.ToLower()
}

function Get-Pinned([string]$Url, [string]$Dest, [string]$Sha256, [string]$Sha512) {
    if (-not (Test-Path -LiteralPath $Dest)) {
        Write-Host "download $Url"
        Run "curl.exe" @("-fsSL", "--retry", "3", "-o", $Dest, $Url)
    }
    if ($Sha256) {
        $got = Get-Hash $Dest "SHA256"
        if ($got -ne $Sha256.ToLower()) { throw "sha256 mismatch for $Dest : $got" }
    }
    if ($Sha512) {
        $got = Get-Hash $Dest "SHA512"
        if ($got -ne $Sha512.ToLower()) { throw "sha512 mismatch for $Dest : $got" }
    }
}

function Import-VcVars {
    if (Get-Command cl.exe -ErrorAction SilentlyContinue) { return }
    $vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
    if (-not (Test-Path $vswhere)) { throw "vswhere.exe is missing" }
    $vs = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (-not $vs) { throw "Visual Studio C++ toolset not found" }
    $bat = Join-Path $vs "VC\Auxiliary\Build\vcvars64.bat"
    Write-Host "vcvars $bat"
    $lines = cmd /c "`"$bat`" >nul && set"
    foreach ($line in $lines) {
        $pair = $line -split "=", 2
        if ($pair.Count -eq 2) {
            Set-Item -Path "Env:$($pair[0])" -Value $pair[1]
        }
    }
    if (-not (Get-Command cl.exe -ErrorAction SilentlyContinue)) { throw "cl.exe is not on PATH after vcvars64" }
}

function Find-Ninja([string]$VsRoot) {
    $cmd = Get-Command ninja.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $known = Join-Path $VsRoot "Common7\IDE\CommonExtensions\Microsoft\CMake\Ninja\ninja.exe"
    if (Test-Path $known) { return $known }
    return $null
}

Import-VcVars
$vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
$vsRoot = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
$ninja = Find-Ninja $vsRoot
$py = $null
foreach ($cand in @("py.exe", "python.exe")) {
    $c = Get-Command $cand -ErrorAction SilentlyContinue
    if ($c) { $py = $c.Source; break }
}
if (-not $py) { throw "Python is required to configure Botan" }
Write-Host "python: $py"
Write-Host "ninja: $ninja"

Get-Pinned $pins.botan.tarball (Join-Path $src "Botan-3.10.0.tar.xz") $pins.botan.sha256 $null
Get-Pinned $pins.json_c.tarball (Join-Path $src "json-c-0.19.tar.gz") $pins.json_c.sha256 $null
Get-Pinned $pins.zlib.tarball (Join-Path $src "zlib-1.4.tar.gz") $pins.zlib.sha256 $null
Get-Pinned $pins.bzip2.tarball (Join-Path $src "bzip2-1.0.8.tar.gz") $pins.bzip2.sha256 $pins.bzip2.sha512

$botanSrc = Join-Path $src "Botan-3.10.0"
if (-not (Test-Path (Join-Path $botanSrc "configure.py"))) {
    Run "tar.exe" @("-xf", (Join-Path $src "Botan-3.10.0.tar.xz"), "-C", $src)
}
$jsonSrc = Join-Path $src "json-c-0.19"
if (-not (Test-Path (Join-Path $jsonSrc "CMakeLists.txt"))) {
    Run "tar.exe" @("-xzf", (Join-Path $src "json-c-0.19.tar.gz"), "-C", $src)
}
$zlibSrc = Join-Path $src "zlib-1.4"
if (-not (Test-Path (Join-Path $zlibSrc "CMakeLists.txt"))) {
    Run "tar.exe" @("-xzf", (Join-Path $src "zlib-1.4.tar.gz"), "-C", $src)
}
$bzSrc = Join-Path $src "bzip2-1.0.8"
if (-not (Test-Path (Join-Path $bzSrc "bzlib.c"))) {
    Run "tar.exe" @("-xzf", (Join-Path $src "bzip2-1.0.8.tar.gz"), "-C", $src)
}

$rnpSrc = Join-Path $src "rnp"
if (-not (Test-Path (Join-Path $rnpSrc ".git"))) {
    Run "git.exe" @("clone", "--depth", "1", "--branch", "v0.18.1", "https://github.com/rnpgp/rnp.git", $rnpSrc)
}
$rnpHead = (& git.exe -C $rnpSrc rev-parse HEAD).Trim()
if ($rnpHead -ne $pins.rnp.commit) { throw "rnp HEAD $rnpHead" }
$sexpp = Join-Path $rnpSrc "src\libsexpp"
if (-not (Test-Path (Join-Path $sexpp ".git"))) {
    if (Test-Path $sexpp) { Remove-Item -Recurse -Force $sexpp }
    Run "git.exe" @("clone", "https://github.com/rnpgp/sexpp.git", $sexpp)
}
Run "git.exe" @("-C", $sexpp, "fetch", "--depth", "1", "origin", $pins.rnp.sexpp_commit)
Run "git.exe" @("-C", $sexpp, "checkout", "--detach", "FETCH_HEAD")
$sexppHead = (& git.exe -C $sexpp rev-parse HEAD).Trim()
if ($sexppHead -ne $pins.rnp.sexpp_commit) { throw "sexpp HEAD $sexppHead" }

$toolchain = Join-Path $root "scripts\windows\msvc-mt.cmake"
$cmakeCommon = @(
    "-DCMAKE_POLICY_DEFAULT_CMP0091=NEW",
    "-DCMAKE_POLICY_VERSION_MINIMUM=3.5",
    "-DCMAKE_TOOLCHAIN_FILE=$toolchain",
    "-DCMAKE_MSVC_RUNTIME_LIBRARY=MultiThreaded",
    "-DCMAKE_BUILD_TYPE=Release"
)
if ($ninja) { $cmakeCommon = @("-G", "Ninja", "-DCMAKE_MAKE_PROGRAM=$ninja") + $cmakeCommon }

$botanPrefix = Join-Path $deps "botan"
if (-not (Test-Path (Join-Path $botanPrefix "include"))) {
    $botanArgs = @(
        (Join-Path $botanSrc "configure.py"),
        "--cc=msvc", "--os=windows", "--cpu=x86_64",
        "--prefix=$botanPrefix",
        "--disable-shared-library",
        "--with-msvc-runtime=MT",
        "--without-documentation",
        "--amalgamation",
        "--with-build-dir=$(Join-Path $work 'botan')"
    )
    if ($ninja) { $botanArgs += @("--build-tool=ninja") } else { $botanArgs += @("--build-tool=nmake") }
    Run $py $botanArgs
    if ($ninja) {
        Run $ninja @("-C", (Join-Path $work "botan"), "install")
    } else {
        Push-Location (Join-Path $work "botan")
        try { Run "nmake.exe" @("install") } finally { Pop-Location }
    }
}
Write-Host "botan prefix:"
Get-ChildItem -Recurse -File $botanPrefix -Include *.lib, *.dll, version.h | ForEach-Object { Write-Host $_.FullName }

$jsonPrefix = Join-Path $deps "json-c"
if (-not (Test-Path (Join-Path $jsonPrefix "lib\json-c.lib"))) {
    Run "cmake.exe" (@(
        "-S", $jsonSrc, "-B", (Join-Path $work "json-c"),
        "-DCMAKE_INSTALL_PREFIX=$jsonPrefix",
        "-DBUILD_SHARED_LIBS=OFF", "-DBUILD_TESTING=OFF", "-DDISABLE_WERROR=ON"
    ) + $cmakeCommon)
    Run "cmake.exe" @("--build", (Join-Path $work "json-c"), "--config", "Release")
    Run "cmake.exe" @("--install", (Join-Path $work "json-c"), "--config", "Release")
}

$zlibPrefix = Join-Path $deps "zlib"
if (-not (Test-Path (Join-Path $zlibPrefix "lib\libz.lib"))) {
    Run "cmake.exe" (@(
        "-S", $zlibSrc, "-B", (Join-Path $work "zlib"),
        "-DCMAKE_INSTALL_PREFIX=$zlibPrefix",
        "-DZLIB_BUILD_SHARED=OFF", "-DZLIB_BUILD_TESTING=OFF"
    ) + $cmakeCommon)
    Run "cmake.exe" @("--build", (Join-Path $work "zlib"), "--config", "Release")
    Run "cmake.exe" @("--install", (Join-Path $work "zlib"), "--config", "Release")
}

$bzPrefix = Join-Path $deps "bzip2"
if (-not (Test-Path (Join-Path $bzPrefix "lib\bz2.lib"))) {
    Run "cmake.exe" (@(
        "-S", (Join-Path $root "scripts\windows\bzip2-static"),
        "-B", (Join-Path $work "bzip2"),
        "-DCMAKE_INSTALL_PREFIX=$bzPrefix",
        "-DBZIP2_SRC=$bzSrc"
    ) + $cmakeCommon)
    Run "cmake.exe" @("--build", (Join-Path $work "bzip2"), "--config", "Release")
    Run "cmake.exe" @("--install", (Join-Path $work "bzip2"), "--config", "Release")
}

$rnpPrefix = Join-Path $deps "rnp"
$rnpBuild = Join-Path $work "rnp"
if (Test-Path $rnpBuild) { Remove-Item -Recurse -Force $rnpBuild }
$link = "ws2_32.lib bcrypt.lib crypt32.lib user32.lib advapi32.lib"
Run "cmake.exe" (@(
    "-S", $rnpSrc, "-B", $rnpBuild,
    "-DCMAKE_INSTALL_PREFIX=$rnpPrefix",
    "-DBUILD_SHARED_LIBS=ON",
    "-DBUILD_TESTING=OFF",
    "-DDOWNLOAD_GTEST=OFF",
    "-DENABLE_DOC=OFF",
    "-DENABLE_CRYPTO_REFRESH=OFF",
    "-DENABLE_PQC=OFF",
    "-DWITH_SEXP_CLI=OFF",
    "-DCRYPTO_BACKEND=botan",
    "-DBOTAN_ROOT_DIR=$botanPrefix",
    "-DCMAKE_PREFIX_PATH=$jsonPrefix;$zlibPrefix;$bzPrefix",
    "-DZLIB_LIBRARY=$(Join-Path $zlibPrefix 'lib\libz.lib')",
    "-DZLIB_INCLUDE_DIR=$(Join-Path $zlibPrefix 'include')",
    "-DZLIB_USE_STATIC_LIBS=ON",
    "-DBZIP2_LIBRARY=$(Join-Path $bzPrefix 'lib\bz2.lib')",
    "-DBZIP2_LIBRARIES=$(Join-Path $bzPrefix 'lib\bz2.lib')",
    "-DBZIP2_INCLUDE_DIR=$(Join-Path $bzPrefix 'include')",
    "-DBZIP2_INCLUDE_DIRS=$(Join-Path $bzPrefix 'include')",
    "-DCMAKE_SHARED_LINKER_FLAGS=$link",
    "-DCMAKE_EXE_LINKER_FLAGS=$link"
) + $cmakeCommon)
Run "cmake.exe" @("--build", $rnpBuild, "--config", "Release")

$dll = Get-ChildItem -Path $rnpBuild -Recurse -Filter "rnp.dll" | Where-Object { $_.FullName -notmatch "examples|tests" } | Select-Object -First 1
if (-not $dll) {
    Get-ChildItem -Path $rnpBuild -Recurse -Filter "*.dll" | ForEach-Object { Write-Host $_.FullName }
    throw "rnp.dll was not produced"
}
Write-Host "rnp.dll $($dll.FullName)"

$go = (Get-Command go.exe).Source
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$exeBuilt = Join-Path $work "rnparmor.exe"
Run $go @("build", "-trimpath", "-o", $exeBuilt, ".\cmd\rnparmor")

if (Test-Path $dist) { Remove-Item -Recurse -Force $dist }
New-Item -ItemType Directory -Force -Path $dist, (Join-Path $dist "licenses") | Out-Null
Copy-Item $exeBuilt (Join-Path $dist "rnparmor.exe")
Copy-Item $dll.FullName (Join-Path $dist "rnp.dll")
Copy-Item (Join-Path $root "third_party\licenses\rnp-LICENSE.md") (Join-Path $dist "licenses\rnp-LICENSE.md")
Copy-Item (Join-Path $root "third_party\licenses\sexpp-LICENSE.md") (Join-Path $dist "licenses\sexpp-LICENSE.md")
Copy-Item (Join-Path $botanSrc "license.txt") (Join-Path $dist "licenses\botan-license.txt")
Copy-Item (Join-Path $jsonSrc "COPYING") (Join-Path $dist "licenses\json-c-COPYING")
Copy-Item (Join-Path $zlibSrc "LICENSE") (Join-Path $dist "licenses\zlib-LICENSE")
Copy-Item (Join-Path $bzSrc "LICENSE") (Join-Path $dist "licenses\bzip2-LICENSE")
Copy-Item (Join-Path $root "third_party\licenses\mygo-MIT.txt") (Join-Path $dist "licenses\mygo-MIT.txt")
Copy-Item (Join-Path $root "third_party\licenses\purego-Apache-2.0.txt") (Join-Path $dist "licenses\purego-Apache-2.0.txt")

$dumpbin = Get-Command dumpbin.exe -ErrorAction SilentlyContinue
if (-not $dumpbin) { throw "dumpbin.exe is missing; refusing to guess DLL dependencies" }
$depReport = New-Object System.Text.StringBuilder
$unknown = @()
$allow = @{
    "kernel32.dll" = $true; "kernelbase.dll" = $true; "ntdll.dll" = $true
    "user32.dll" = $true; "gdi32.dll" = $true; "gdiplus.dll" = $true
    "advapi32.dll" = $true; "shell32.dll" = $true; "shlwapi.dll" = $true
    "ole32.dll" = $true; "oleaut32.dll" = $true; "combase.dll" = $true
    "comdlg32.dll" = $true; "rpcrt4.dll" = $true; "sechost.dll" = $true
    "bcrypt.dll" = $true; "bcryptprimitives.dll" = $true; "crypt32.dll" = $true
    "ncrypt.dll" = $true; "ws2_32.dll" = $true; "wsock32.dll" = $true
    "iphlpapi.dll" = $true; "dnsapi.dll" = $true; "imm32.dll" = $true
    "userenv.dll" = $true; "version.dll" = $true; "winmm.dll" = $true
    "dwmapi.dll" = $true; "uxtheme.dll" = $true; "shcore.dll" = $true
    "d3d11.dll" = $true; "dxgi.dll" = $true; "d3dcompiler_47.dll" = $true
    "dwrite.dll" = $true; "d2d1.dll" = $true; "oleacc.dll" = $true
    "usp10.dll" = $true; "normaliz.dll" = $true; "psapi.dll" = $true
    "powrprof.dll" = $true; "setupapi.dll" = $true; "cfgmgr32.dll" = $true
    "hid.dll" = $true; "winhttp.dll" = $true; "secur32.dll" = $true
    "netapi32.dll" = $true; "mpr.dll" = $true; "wtsapi32.dll" = $true
    "propsys.dll" = $true; "msimg32.dll" = $true; "comctl32.dll" = $true
    "msvcrt.dll" = $true; "wintrust.dll" = $true; "imagehlp.dll" = $true
    "dbghelp.dll" = $true; "nsi.dll" = $true; "mswsock.dll" = $true
    "sspicli.dll" = $true; "profapi.dll" = $true; "cryptbase.dll" = $true
    "ucrtbase.dll" = $true; "winspool.drv" = $true; "avrt.dll" = $true
    "mmdevapi.dll" = $true; "wldp.dll" = $true
}
foreach ($bin in @((Join-Path $dist "rnparmor.exe"), (Join-Path $dist "rnp.dll"))) {
    $text = & dumpbin.exe /nologo /dependents $bin | Out-String
    [void]$depReport.AppendLine("==== $(Split-Path $bin -Leaf) ====")
    [void]$depReport.AppendLine($text.TrimEnd())
    foreach ($line in ($text -split "`r?`n")) {
        $t = $line.Trim()
        if ($t -notmatch '^(?i)[A-Za-z0-9_.-]+\.(dll|drv)$') { continue }
        $name = $t.ToLower()
        if ($allow.ContainsKey($name)) { continue }
        if ($name -like "api-ms-win-*" -or $name -like "ext-ms-*") { continue }
        $unknown += "$name from $(Split-Path $bin -Leaf)"
    }
}
$depPath = Join-Path $dist "DEPENDENCIES.txt"
Set-Content -LiteralPath $depPath -Value $depReport.ToString() -Encoding utf8
if ($unknown.Count -gt 0) {
    throw ("unexpected DLL imports, not bundled and not in the Windows system allowlist:`n" + ($unknown -join "`n"))
}

@"
RnpArmor Windows x64 portable directory.

Start rnparmor.exe from this folder. It loads rnp.dll from the same directory
with an absolute path and LoadLibraryEx flags that do not search the current
directory or PATH.

This copy was built for testing. Do not put real keys or passphrases in it.
The program does not read the default keyring unless a later build adds an
explicit directory, and this build does not.

RnpArmor has no chosen project license. Third-party license texts are in licenses\.
zlib and bzip2 are linked into rnp.dll. Their licenses are still included.

A GitHub-hosted Windows runner is not a manual Windows 10 or 11 desktop test.
Chinese input and high-DPI were not tested. An in-flight librnp call cannot be
cancelled; cancellation only discards output after the call returns.
"@ | Set-Content -LiteralPath (Join-Path $dist "PORTABLE.txt") -Encoding utf8

Push-Location $dist
try {
    $files = Get-ChildItem -File -Recurse | Where-Object { $_.Name -ne "SHA256SUMS" } | ForEach-Object {
        $_.FullName.Substring($dist.Length + 1)
    } | Sort-Object
    $lines = foreach ($rel in $files) {
        $h = Get-Hash (Join-Path $dist $rel) "SHA256"
        "{0}  {1}" -f $h, ($rel -replace '\\', '/')
    }
    Set-Content -LiteralPath (Join-Path $dist "SHA256SUMS") -Value ($lines -join "`n") -Encoding ascii
} finally { Pop-Location }

# Tests use only the packaged DLL. PATH cannot supply a substitute.
$savedPath = $env:PATH
$env:PATH = "$env:SystemRoot\System32"
$env:RNPARMOR_LIBRNP = (Resolve-Path (Join-Path $dist "rnp.dll")).Path
foreach ($name in @("LIB", "INCLUDE", "LIBPATH", "VCINSTALLDIR", "VSINSTALLDIR")) {
    Remove-Item "Env:$name" -ErrorAction SilentlyContinue
}
$testLog = Join-Path $work "go-test.txt"
$testErr = Join-Path $work "go-test-err.txt"
Write-Host "go test with RNPARMOR_LIBRNP=$env:RNPARMOR_LIBRNP"
$testProc = Start-Process -FilePath $go -WorkingDirectory $root -Wait -PassThru -NoNewWindow `
    -ArgumentList "test -count=1 -timeout 20m ./internal/loadlib/ ./internal/rnp/" `
    -RedirectStandardOutput $testLog -RedirectStandardError $testErr
Get-Content $testLog, $testErr -ErrorAction SilentlyContinue
if ($testProc.ExitCode -ne 0) { throw "go test failed: $($testProc.ExitCode)" }

Remove-Item Env:RNPARMOR_LIBRNP -ErrorAction SilentlyContinue
$smokeOut = Join-Path $work "smoke-out.txt"
$smokeErr = Join-Path $work "smoke-err.txt"
$proc = Start-Process -FilePath (Join-Path $dist "rnparmor.exe") -WorkingDirectory $dist -PassThru -WindowStyle Hidden -RedirectStandardOutput $smokeOut -RedirectStandardError $smokeErr
$running = -not $proc.WaitForExit(12000)
$window = 0
try {
    Add-Type @"
using System;
using System.Runtime.InteropServices;
public class RnpArmorWin {
  [DllImport("user32.dll", CharSet = CharSet.Unicode)]
  public static extern IntPtr FindWindow(string cls, string title);
}
"@
    $window = [RnpArmorWin]::FindWindow($null, "RnpArmor")
} catch {
    Write-Host "FindWindow unavailable: $_"
}
if ($running) {
    Stop-Process -Id $proc.Id -Force
    $smokeState = "process still running after 12s; stopped"
} else {
    $smokeState = "process exited $($proc.ExitCode)"
}
$env:PATH = $savedPath

$os = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"
$head = (& git.exe -C $root rev-parse HEAD).Trim()
if (Test-Path $zip) { Remove-Item -Force $zip }
Run "tar.exe" @("-a", "-c", "-f", $zip, "-C", (Split-Path $dist -Parent), (Split-Path $dist -Leaf))
$zipHash = Get-Hash $zip "SHA256"
$listing = (Get-ChildItem -Recurse -File $dist | ForEach-Object { "{0}`t{1}" -f $_.Length, $_.FullName.Substring($dist.Length + 1) }) -join "`n"
@"
commit $head
os $($os.ProductName) build $($os.CurrentBuild) display $($os.DisplayVersion)
go $(& $go version)
dll $($dll.FullName)
RNPARMOR_LIBRNP during tests: packaged rnp.dll, PATH=$env:SystemRoot\System32
go test: passed
smoke: $smokeState
FindWindow RnpArmor: $window
zip sha256 $zipHash
files:
$listing
"@ | Set-Content -LiteralPath $report -Encoding utf8
Write-Host "ZIP $zip"
Write-Host "SHA256 $zipHash"
Get-Content $report
