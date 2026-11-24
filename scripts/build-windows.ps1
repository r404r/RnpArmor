# Build Windows x64 portable outputs in staged mode:
#   -Stage fetch-deps : download/checksum/extract Botan/json-c/zlib/bzip2 sources
#   -Stage botan      : build/install minimized static Botan into out/deps
#   -Stage small-deps : build/install json-c/zlib/bzip2 + cache dependency licenses
#   -Stage deps       : fetch-deps -> botan -> small-deps
#   -Stage rnp        : build librnp (rnp.dll) into out/win-build/rnp-dll
#   -Stage package    : build rnparmor.exe, run tests/smoke, package zip/report
#   -Stage all        : deps -> rnp -> package (default)
param(
    [string]$Version = "dev",
    [ValidateSet("all", "deps", "fetch-deps", "botan", "small-deps", "rnp", "package")]
    [string]$Stage = "all"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true

function Log([string]$Message) {
    Write-Host ("[{0}] {1}" -f (Get-Date -Format "HH:mm:ss"), $Message)
}

function Sanitize-Annotation([string]$Text) {
    if ($null -eq $Text) { return "" }
    return ($Text -replace '[\r\n]+', ' | ')
}

function Emit-Notice([string]$Title, [string]$Message) {
    if ($env:GITHUB_ACTIONS -eq "true") {
        Write-Host ("::notice title={0}::{1}" -f (Sanitize-Annotation $Title), (Sanitize-Annotation $Message))
    }
}

function Emit-Error([string]$Title, [string]$Message) {
    if ($env:GITHUB_ACTIONS -eq "true") {
        Write-Host ("::error title={0}::{1}" -f (Sanitize-Annotation $Title), (Sanitize-Annotation $Message))
    }
}

function Invoke-TimedStage([string]$Name, [scriptblock]$Body) {
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    Log "stage: $Name"
    & $Body
    $sw.Stop()
    Log (("stage: {0} done ({1}s)" -f $Name, [math]::Round($sw.Elapsed.TotalSeconds, 1)))
    Emit-Notice ("stage-{0}" -f $Name) ("done in {0}s" -f [math]::Round($sw.Elapsed.TotalSeconds, 1))
}

function Run([string]$File, [string[]]$CommandArgs) {
    Write-Host "+ $File $($CommandArgs -join ' ')"
    $prevNative = $PSNativeCommandUseErrorActionPreference
    $PSNativeCommandUseErrorActionPreference = $false
    $tail = New-Object System.Collections.Generic.List[string]
    try {
        & $File @CommandArgs 2>&1 | ForEach-Object {
            $line = $_.ToString()
            Write-Host $line
            $trim = $line.Trim()
            if ($trim.Length -gt 0) {
                $tail.Add($trim)
                if ($tail.Count -gt 15) { $tail.RemoveAt(0) }
            }
        }
        $code = $LASTEXITCODE
    } finally {
        $PSNativeCommandUseErrorActionPreference = $prevNative
    }

    if ($code -ne 0) {
        $take = [Math]::Min(5, $CommandArgs.Count)
        $headArgs = if ($take -gt 0) { ($CommandArgs[0..($take - 1)] -join ' ') } else { "" }
        $ctx = if ($tail.Count -gt 0) { ($tail -join ' | ') } else { "no output" }
        throw "$File $headArgs exited $code :: $ctx"
    }
}

function Get-Hash([string]$Path, [string]$Algo) {
    return (Get-FileHash -Algorithm $Algo -LiteralPath $Path).Hash.ToLower()
}

function To-CMakePath([string]$Path) {
    return ($Path -replace '\\', '/')
}

function Get-Pinned([string]$Url, [string]$Dest, [string]$Sha256, [string]$Sha512) {
    if (-not (Test-Path -LiteralPath $Dest)) {
        Log "download $Url"
        Run "curl.exe" @(
            "-fsSL",
            "--retry", "3",
            "--connect-timeout", "30",
            "--max-time", "300",
            "-o", $Dest,
            $Url
        )
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
    if (Get-Command cl.exe -ErrorAction SilentlyContinue) {
        $vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
        if (Test-Path $vswhere) {
            return (& $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath)
        }
        return ""
    }

    $vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
    if (-not (Test-Path $vswhere)) { throw "vswhere.exe is missing" }
    $vs = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (-not $vs) { throw "Visual Studio C++ toolset not found" }

    $bat = Join-Path $vs "VC\Auxiliary\Build\vcvars64.bat"
    Log "vcvars $bat"
    $lines = cmd /c "`"$bat`" >nul && set"
    foreach ($line in $lines) {
        $pair = $line -split "=", 2
        if ($pair.Count -eq 2) {
            Set-Item -Path "Env:$($pair[0])" -Value $pair[1]
        }
    }
    if (-not (Get-Command cl.exe -ErrorAction SilentlyContinue)) {
        throw "cl.exe is not on PATH after vcvars64"
    }
    return $vs
}

function Find-Ninja([string]$VsRoot) {
    $cmd = Get-Command ninja.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    if ($VsRoot) {
        $known = Join-Path $VsRoot "Common7\IDE\CommonExtensions\Microsoft\CMake\Ninja\ninja.exe"
        if (Test-Path $known) { return $known }
    }
    return $null
}

function Resolve-Python {
    $candidates = @()
    foreach ($name in @("python.exe", "py.exe")) {
        $cmds = Get-Command $name -All -ErrorAction SilentlyContinue
        foreach ($c in $cmds) {
            if ($c.Source) { $candidates += $c.Source }
        }
    }

    $ordered = @()
    foreach ($p in $candidates) {
        if ($ordered -notcontains $p) { $ordered += $p }
    }

    foreach ($p in $ordered) {
        if ($p -match "\\WindowsApps\\") {
            continue
        }

        try {
            $out = & $p "--version" 2>&1
            $code = $LASTEXITCODE
            $text = ($out | Out-String).Trim()
            if ($code -eq 0 -and $text -match '^Python 3\.') {
                return $p
            }
        } catch {
            continue
        }
    }

    return $null
}

function Extract-TarWithPython([string]$ArchivePath, [string]$Destination) {
    $pyCode = @"
import pathlib
import sys
import tarfile

src = pathlib.Path(sys.argv[1])
dst = pathlib.Path(sys.argv[2])
dst.mkdir(parents=True, exist_ok=True)

with tarfile.open(src, 'r:*') as tf:
    kwargs = {}
    if sys.version_info >= (3, 12):
        kwargs['filter'] = 'data'
    try:
        tf.extractall(dst, **kwargs)
    except TypeError:
        tf.extractall(dst)
"@

    Run $py @("-c", $pyCode, $ArchivePath, $Destination)
}

function Parse-BotanModules([string]$ModuleFile) {
    if (-not (Test-Path -LiteralPath $ModuleFile)) {
        throw "Botan modules file missing: $ModuleFile"
    }

    $modules = @()
    foreach ($raw in (Get-Content -LiteralPath $ModuleFile)) {
        $line = ($raw -split "#", 2)[0].Trim()
        if (-not $line) { continue }
        if ($line -notmatch '^[a-z0-9_]+$') {
            throw "invalid Botan module token: '$line'"
        }
        $modules += $line
    }

    if ($modules.Count -eq 0) {
        throw "Botan modules file is empty after stripping comments"
    }

    $dup = $modules | Group-Object | Where-Object { $_.Count -gt 1 }
    if ($dup) {
        $names = ($dup | ForEach-Object { $_.Name }) -join ", "
        throw "duplicate Botan modules: $names"
    }

    return $modules
}

function Get-BotanLib([string]$BotanPrefix) {
    $libDir = Join-Path $BotanPrefix "lib"
    if (-not (Test-Path $libDir)) { return $null }
    return (Get-ChildItem -Path $libDir -Filter "*.lib" -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match "botan" } |
        Select-Object -First 1)
}

function Test-BotanReady([string]$BotanPrefix, [string]$ExpectedModulesHash) {
    $botanHeader = Join-Path $BotanPrefix "include\botan-3\botan\version.h"
    $botanLib = Get-BotanLib $BotanPrefix
    $botanStampPath = Join-Path $BotanPrefix ".modules-sha256"

    if (-not (Test-Path -LiteralPath $botanHeader)) { return $false }
    if (-not $botanLib) { return $false }
    if (-not (Test-Path -LiteralPath $botanStampPath)) { return $false }

    $gotHash = (Get-Content -LiteralPath $botanStampPath -ErrorAction SilentlyContinue | Out-String).Trim()
    return ($gotHash -eq $ExpectedModulesHash)
}

function Get-ZlibLib([string]$ZlibPrefix) {
    $libDir = Join-Path $ZlibPrefix "lib"
    if (-not (Test-Path $libDir)) { return $null }
    $libs = Get-ChildItem -Path $libDir -Filter "*.lib" -ErrorAction SilentlyContinue
    if (-not $libs) { return $null }

    foreach ($pat in @('^libz.*\.lib$', '^zlib.*\.lib$', '^z.*\.lib$')) {
        $hit = $libs | Where-Object { $_.Name -imatch $pat } | Select-Object -First 1
        if ($hit) { return $hit }
    }
    return $null
}

function Get-JsonCLib([string]$JsonPrefix) {
    $libDir = Join-Path $JsonPrefix "lib"
    if (-not (Test-Path $libDir)) { return $null }
    $libs = Get-ChildItem -Path $libDir -Filter "*.lib" -ErrorAction SilentlyContinue
    if (-not $libs) { return $null }

    foreach ($name in @('json-c.lib', 'json-c-static.lib')) {
        $hit = $libs | Where-Object { $_.Name -ieq $name } | Select-Object -First 1
        if ($hit) { return $hit }
    }
    return $null
}

function Test-DepsHot(
    [string]$DepsRoot,
    [string]$BotanPrefix,
    [string]$BotanModulesHash,
    [string]$JsonPrefix,
    [string]$ZlibPrefix,
    [string]$BzPrefix
) {
    $botanReady = Test-BotanReady -BotanPrefix $BotanPrefix -ExpectedModulesHash $BotanModulesHash
    $zlibLib = Get-ZlibLib $ZlibPrefix
    $jsonCLib = Get-JsonCLib $JsonPrefix

    $must = @(
        (Join-Path $BzPrefix "lib\bz2.lib"),
        (Join-Path $DepsRoot "dirent\include\dirent.h"),
        (Join-Path $DepsRoot "licenses\botan-license.txt"),
        (Join-Path $DepsRoot "licenses\dirent-LICENSE"),
        (Join-Path $DepsRoot "licenses\json-c-COPYING"),
        (Join-Path $DepsRoot "licenses\zlib-LICENSE"),
        (Join-Path $DepsRoot "licenses\bzip2-LICENSE")
    )

    foreach ($m in $must) {
        if (-not (Test-Path -LiteralPath $m)) { return $false }
    }
    if ((-not $botanReady) -or (-not $zlibLib) -or (-not $jsonCLib)) { return $false }
    return $true
}

function Resolve-LicensePath([string]$DepsLicensesDir, [string]$Name, [string]$Fallback) {
    $fromCache = Join-Path $DepsLicensesDir $Name
    if (Test-Path -LiteralPath $fromCache) { return $fromCache }
    if (Test-Path -LiteralPath $Fallback) { return $Fallback }
    throw "license file missing: $Name"
}

function Ensure-RnpSources(
    [string]$SrcRoot,
    [object]$Pins
) {
    Log "fetch rnp and sexpp"
    $rnpSrc = Join-Path $SrcRoot "rnp"
    if (-not (Test-Path (Join-Path $rnpSrc ".git"))) {
        Run "git.exe" @("clone", "--depth", "1", "--branch", "v0.18.1", "https://github.com/rnpgp/rnp.git", $rnpSrc)
    }
    $rnpHead = (& git.exe -C $rnpSrc rev-parse HEAD).Trim()
    if ($rnpHead -ne $Pins.rnp.commit) { throw "rnp HEAD mismatch: $rnpHead" }

    $sexpp = Join-Path $rnpSrc "src\libsexpp"
    if (-not (Test-Path (Join-Path $sexpp ".git"))) {
        if (Test-Path $sexpp) { Remove-Item -Recurse -Force $sexpp }
        Run "git.exe" @("clone", "https://github.com/rnpgp/sexpp.git", $sexpp)
    }
    Run "git.exe" @("-C", $sexpp, "fetch", "--depth", "1", "origin", $Pins.rnp.sexpp_commit)
    Run "git.exe" @("-C", $sexpp, "checkout", "--detach", "FETCH_HEAD")
    $sexppHead = (& git.exe -C $sexpp rev-parse HEAD).Trim()
    if ($sexppHead -ne $Pins.rnp.sexpp_commit) { throw "sexpp HEAD mismatch: $sexppHead" }

    return $rnpSrc
}

function Ensure-Extracted([string]$MarkerPath, [string]$ArchivePath) {
    if (-not (Test-Path -LiteralPath $MarkerPath)) {
        Extract-TarWithPython -ArchivePath $ArchivePath -Destination $src
    }
}

Log "build-windows.ps1 start, stage=$Stage, version=$Version"

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root
$pins = Get-Content -Raw -Path (Join-Path $root "deps\pins.json") | ConvertFrom-Json
if ($pins.rnp.commit -ne "aad1892e4d8423398a0b973bd0a7e4544c359afc") { throw "rnp pin mismatch" }
if ($pins.rnp.do_not_use -notcontains "v0.18.0") { throw "v0.18.0 must stay rejected" }

if ($env:GITHUB_ACTIONS -eq "true") {
    try {
        Add-MpPreference -ExclusionPath $root -ErrorAction Stop
        Log "best-effort Defender exclusion added for: $root"
    } catch {
        Log "best-effort Defender exclusion skipped: $($_.Exception.Message)"
    }
}

$src = Join-Path $root "out\src"
$deps = Join-Path $root "out\deps"
$work = Join-Path $root "out\win-build"
$dist = Join-Path $root "out\dist\rnparmor-windows-amd64"
$zip = Join-Path $root "out\rnparmor-windows-amd64.zip"
$report = Join-Path $root "out\windows-report.txt"
$rnpDllDir = Join-Path $work "rnp-dll"
$rnpDllPath = Join-Path $rnpDllDir "rnp.dll"
New-Item -ItemType Directory -Force -Path $src, $deps, $work, $rnpDllDir | Out-Null

$vsRoot = Import-VcVars
$vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
if (-not $vsRoot -and (Test-Path $vswhere)) {
    $vsRoot = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
}
$ninja = Find-Ninja $vsRoot
$py = Resolve-Python
if (-not $py) { throw "Python 3 is required to extract/build dependencies" }
if (-not $ninja) { throw "ninja.exe was not found; it is required for Windows dependency builds" }
Log "python: $py"
Log "ninja: $ninja"

$toolchain = Join-Path $root "scripts\windows\msvc-mt.cmake"
$cmakeCommon = @(
    "-G", "Ninja",
    "-DCMAKE_MAKE_PROGRAM=$(To-CMakePath $ninja)",
    "-DCMAKE_C_COMPILER=cl.exe",
    "-DCMAKE_CXX_COMPILER=cl.exe",
    "-DCMAKE_POLICY_DEFAULT_CMP0091=NEW",
    "-DCMAKE_POLICY_VERSION_MINIMUM=3.5",
    "-DCMAKE_TOOLCHAIN_FILE=$(To-CMakePath $toolchain)",
    "-DCMAKE_MSVC_RUNTIME_LIBRARY=MultiThreaded",
    "-DCMAKE_BUILD_TYPE=Release"
)

$botanTar = Join-Path $src "Botan-3.10.0.tar.xz"
$jsonTar = Join-Path $src "json-c-0.19.tar.gz"
$zlibTar = Join-Path $src "zlib-1.4.tar.gz"
$bzTar = Join-Path $src "bzip2-1.0.8.tar.gz"

$botanSrc = Join-Path $src "Botan-3.10.0"
$jsonSrc = Join-Path $src "json-c-0.19"
$zlibSrc = Join-Path $src "zlib-1.4"
$bzSrc = Join-Path $src "bzip2-1.0.8"

$botanPrefix = Join-Path $deps "botan"
$jsonPrefix = Join-Path $deps "json-c"
$zlibPrefix = Join-Path $deps "zlib"
$bzPrefix = Join-Path $deps "bzip2"
$direntSrcDir = Join-Path $src "dirent-1.24"
$direntHeaderSrc = Join-Path $direntSrcDir "dirent.h"
$direntLicenseSrc = Join-Path $direntSrcDir "LICENSE"
$direntPrefix = Join-Path $deps "dirent"
$direntInclude = Join-Path $direntPrefix "include"
$depsLicenses = Join-Path $deps "licenses"
$botanModulesFile = Join-Path $root "scripts\windows\botan3-modules.txt"
$botanModulesHash = Get-Hash $botanModulesFile "SHA256"
$botanStampPath = Join-Path $botanPrefix ".modules-sha256"

function Build-FetchDeps {
    if (Test-DepsHot -DepsRoot $deps -BotanPrefix $botanPrefix -BotanModulesHash $botanModulesHash -JsonPrefix $jsonPrefix -ZlibPrefix $zlibPrefix -BzPrefix $bzPrefix) {
        Log "deps hot cache hit: skip fetch-deps"
        return
    }

    Get-Pinned $pins.botan.tarball $botanTar $pins.botan.sha256 $null
    Get-Pinned $pins.json_c.tarball $jsonTar $pins.json_c.sha256 $null
    Get-Pinned $pins.zlib.tarball $zlibTar $pins.zlib.sha256 $null
    Get-Pinned $pins.bzip2.tarball $bzTar $pins.bzip2.sha256 $pins.bzip2.sha512
    New-Item -ItemType Directory -Force -Path $direntSrcDir | Out-Null
    Get-Pinned $pins.dirent.header_url $direntHeaderSrc $pins.dirent.header_sha256 $null
    Get-Pinned $pins.dirent.license_url $direntLicenseSrc $pins.dirent.license_sha256 $null

    Ensure-Extracted -MarkerPath (Join-Path $botanSrc "configure.py") -ArchivePath $botanTar
    Ensure-Extracted -MarkerPath (Join-Path $jsonSrc "CMakeLists.txt") -ArchivePath $jsonTar
    Ensure-Extracted -MarkerPath (Join-Path $zlibSrc "CMakeLists.txt") -ArchivePath $zlibTar
    Ensure-Extracted -MarkerPath (Join-Path $bzSrc "bzlib.c") -ArchivePath $bzTar
}

function Build-Botan {
    if (Test-BotanReady -BotanPrefix $botanPrefix -ExpectedModulesHash $botanModulesHash) {
        Log "botan cache hit with matching modules hash: skip botan"
        return
    }

    if (Test-Path -LiteralPath $botanPrefix) { Remove-Item -Recurse -Force $botanPrefix }
    $botanBuildDir = Join-Path $work "botan"
    if (Test-Path -LiteralPath $botanBuildDir) { Remove-Item -Recurse -Force $botanBuildDir }
    New-Item -ItemType Directory -Force -Path $botanPrefix | Out-Null

    Ensure-Extracted -MarkerPath (Join-Path $botanSrc "configure.py") -ArchivePath $botanTar

    $modules = Parse-BotanModules $botanModulesFile
    $botanModules = $modules -join ","
    Log "botan modules count: $($modules.Count)"

    $botanArgs = @(
        (Join-Path $botanSrc "configure.py"),
        "--cc=msvc", "--os=windows", "--cpu=x86_64",
        "--prefix=$botanPrefix",
        "--build-targets=static",
        "--disable-shared-library",
        "--minimized-build",
        "--enable-modules=$botanModules",
        "--no-install-python-module",
        "--without-documentation",
        "--msvc-runtime=MT",
        "--build-tool=ninja",
        "--with-build-dir=$botanBuildDir"
    )
    Run $py $botanArgs
    Run $ninja @("-C", $botanBuildDir, "install")
    Set-Content -LiteralPath $botanStampPath -Value $botanModulesHash -Encoding ascii

    if (-not (Test-BotanReady -BotanPrefix $botanPrefix -ExpectedModulesHash $botanModulesHash)) {
        throw "Botan static install incomplete"
    }
}

function Build-SmallDeps {
    if (Test-DepsHot -DepsRoot $deps -BotanPrefix $botanPrefix -BotanModulesHash $botanModulesHash -JsonPrefix $jsonPrefix -ZlibPrefix $zlibPrefix -BzPrefix $bzPrefix) {
        Log "deps hot cache hit: skip small-deps"
        return
    }

    Ensure-Extracted -MarkerPath (Join-Path $botanSrc "license.txt") -ArchivePath $botanTar
    Ensure-Extracted -MarkerPath (Join-Path $jsonSrc "CMakeLists.txt") -ArchivePath $jsonTar
    Ensure-Extracted -MarkerPath (Join-Path $zlibSrc "CMakeLists.txt") -ArchivePath $zlibTar
    Ensure-Extracted -MarkerPath (Join-Path $bzSrc "bzlib.c") -ArchivePath $bzTar
    New-Item -ItemType Directory -Force -Path $direntSrcDir | Out-Null
    Get-Pinned $pins.dirent.header_url $direntHeaderSrc $pins.dirent.header_sha256 $null
    Get-Pinned $pins.dirent.license_url $direntLicenseSrc $pins.dirent.license_sha256 $null

    if (-not (Get-JsonCLib $jsonPrefix)) {
        Run "cmake.exe" (@(
            "-S", $jsonSrc, "-B", (Join-Path $work "json-c"),
            "-DCMAKE_INSTALL_PREFIX=$(To-CMakePath $jsonPrefix)",
            "-DBUILD_SHARED_LIBS=OFF", "-DBUILD_TESTING=OFF", "-DDISABLE_WERROR=ON"
        ) + $cmakeCommon)
        Run "cmake.exe" @("--build", (Join-Path $work "json-c"), "--config", "Release")
        Run "cmake.exe" @("--install", (Join-Path $work "json-c"), "--config", "Release")
    }

    if (-not (Get-ZlibLib $zlibPrefix)) {
        Run "cmake.exe" (@(
            "-S", $zlibSrc, "-B", (Join-Path $work "zlib"),
            "-DCMAKE_INSTALL_PREFIX=$(To-CMakePath $zlibPrefix)",
            "-DZLIB_BUILD_SHARED=OFF", "-DZLIB_BUILD_TESTING=OFF"
        ) + $cmakeCommon)
        Run "cmake.exe" @("--build", (Join-Path $work "zlib"), "--config", "Release")
        Run "cmake.exe" @("--install", (Join-Path $work "zlib"), "--config", "Release")
    }

    if (-not (Test-Path (Join-Path $bzPrefix "lib\bz2.lib"))) {
        Run "cmake.exe" (@(
            "-S", (Join-Path $root "scripts\windows\bzip2-static"),
            "-B", (Join-Path $work "bzip2"),
            "-DCMAKE_INSTALL_PREFIX=$(To-CMakePath $bzPrefix)",
            "-DBZIP2_SRC:PATH=$(To-CMakePath $bzSrc)"
        ) + $cmakeCommon)
        Run "cmake.exe" @("--build", (Join-Path $work "bzip2"), "--config", "Release")
        Run "cmake.exe" @("--install", (Join-Path $work "bzip2"), "--config", "Release")
    }

    New-Item -ItemType Directory -Force -Path $depsLicenses | Out-Null
    New-Item -ItemType Directory -Force -Path $direntInclude | Out-Null
    Copy-Item $direntHeaderSrc (Join-Path $direntInclude "dirent.h") -Force
    Copy-Item (Join-Path $botanSrc "license.txt") (Join-Path $depsLicenses "botan-license.txt") -Force
    Copy-Item $direntLicenseSrc (Join-Path $depsLicenses "dirent-LICENSE") -Force
    Copy-Item (Join-Path $jsonSrc "COPYING") (Join-Path $depsLicenses "json-c-COPYING") -Force
    Copy-Item (Join-Path $zlibSrc "LICENSE") (Join-Path $depsLicenses "zlib-LICENSE") -Force
    Copy-Item (Join-Path $bzSrc "LICENSE") (Join-Path $depsLicenses "bzip2-LICENSE") -Force

    if (-not (Test-DepsHot -DepsRoot $deps -BotanPrefix $botanPrefix -BotanModulesHash $botanModulesHash -JsonPrefix $jsonPrefix -ZlibPrefix $zlibPrefix -BzPrefix $bzPrefix)) {
        throw "small-deps stage finished but dependency cache is incomplete"
    }
}

function Build-Deps {
    Invoke-TimedStage "fetch-deps" { Build-FetchDeps }
    Invoke-TimedStage "botan" { Build-Botan }
    Invoke-TimedStage "small-deps" { Build-SmallDeps }
}

function Build-Rnp {
    if (-not (Test-DepsHot -DepsRoot $deps -BotanPrefix $botanPrefix -BotanModulesHash $botanModulesHash -JsonPrefix $jsonPrefix -ZlibPrefix $zlibPrefix -BzPrefix $bzPrefix)) {
        throw "dependency cache is incomplete; run -Stage deps first"
    }

    $rnpSrc = Ensure-RnpSources -SrcRoot $src -Pins $pins
    $zlibLib = Get-ZlibLib $zlibPrefix
    if (-not $zlibLib) { throw "zlib static library not found under $zlibPrefix\lib" }
    $jsonCLib = Get-JsonCLib $jsonPrefix
    if (-not $jsonCLib) { throw "json-c static library not found under $jsonPrefix\lib" }

    $rnpBuild = Join-Path $work "rnp"
    if (Test-Path $rnpBuild) { Remove-Item -Recurse -Force $rnpBuild }

    $link = "ws2_32.lib bcrypt.lib crypt32.lib user32.lib advapi32.lib"
    Run "cmake.exe" (@(
        "-S", $rnpSrc, "-B", $rnpBuild,
        "-DCMAKE_INSTALL_PREFIX=$(To-CMakePath (Join-Path $deps 'rnp'))",
        "-DBUILD_SHARED_LIBS=ON",
        "-DBUILD_TESTING=OFF",
        "-DDOWNLOAD_GTEST=OFF",
        "-DENABLE_DOC=OFF",
        "-DENABLE_CRYPTO_REFRESH=OFF",
        "-DENABLE_PQC=OFF",
        "-DWITH_SEXP_CLI=OFF",
        "-DCRYPTO_BACKEND=botan",
        "-DBOTAN_ROOT_DIR=$(To-CMakePath $botanPrefix)",
        "-DCMAKE_PREFIX_PATH=$((To-CMakePath $jsonPrefix) + ';' + (To-CMakePath $zlibPrefix) + ';' + (To-CMakePath $bzPrefix))",
        "-DDIRENT_INCLUDE_DIR=$(To-CMakePath $direntInclude)",
        "-DGETOPT_INCLUDE_DIR=$(To-CMakePath $direntInclude)",
        "-DGETOPT_LIBRARY=kernel32.lib",
        "-DJSON-C_LIBRARY=$(To-CMakePath $jsonCLib.FullName)",
        "-DZLIB_LIBRARY=$(To-CMakePath $zlibLib.FullName)",
        "-DZLIB_INCLUDE_DIR=$(To-CMakePath (Join-Path $zlibPrefix 'include'))",
        "-DZLIB_USE_STATIC_LIBS=ON",
        "-DBZIP2_LIBRARY=$(To-CMakePath (Join-Path $bzPrefix 'lib\bz2.lib'))",
        "-DBZIP2_LIBRARIES=$(To-CMakePath (Join-Path $bzPrefix 'lib\bz2.lib'))",
        "-DBZIP2_INCLUDE_DIR=$(To-CMakePath (Join-Path $bzPrefix 'include'))",
        "-DBZIP2_INCLUDE_DIRS=$(To-CMakePath (Join-Path $bzPrefix 'include'))",
        "-DCMAKE_SHARED_LINKER_FLAGS=$link",
        "-DCMAKE_EXE_LINKER_FLAGS=$link"
    ) + $cmakeCommon)

    Run "cmake.exe" @("--build", $rnpBuild, "--config", "Release", "--target", "librnp")

    $dll = Get-ChildItem -Path $rnpBuild -Recurse -Filter "rnp.dll" |
        Where-Object { $_.FullName -notmatch "examples|tests" } |
        Select-Object -First 1
    if (-not $dll) {
        Get-ChildItem -Path $rnpBuild -Recurse -Filter "*.dll" | ForEach-Object { Write-Host $_.FullName }
        throw "rnp.dll was not produced"
    }

    Copy-Item -Force $dll.FullName $rnpDllPath
    Log "rnp.dll cached at $rnpDllPath"
    Write-Host "rnp.dll sha256 $(Get-Hash $rnpDllPath 'SHA256')"
}

function Build-Package {
    if (-not (Test-Path -LiteralPath $rnpDllPath)) {
        $rnpBuild = Join-Path $work "rnp"
        if (Test-Path $rnpBuild) {
            $dll = Get-ChildItem -Path $rnpBuild -Recurse -Filter "rnp.dll" |
                Where-Object { $_.FullName -notmatch "examples|tests" } |
                Select-Object -First 1
            if ($dll) {
                Copy-Item -Force $dll.FullName $rnpDllPath
            }
        }
    }
    if (-not (Test-Path -LiteralPath $rnpDllPath)) {
        throw "rnp.dll not found at $rnpDllPath; run -Stage rnp first"
    }

    $goCmd = Get-Command go.exe -ErrorAction SilentlyContinue
    if (-not $goCmd) { throw "go.exe is required for package stage" }
    $go = $goCmd.Source

    Log "go build (version $Version)"
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $exeBuilt = Join-Path $work "rnparmor.exe"
    $ldflags = "-s -w -X main.appVersion=$Version"
    Run $go @("build", "-trimpath", "-ldflags", $ldflags, "-o", $exeBuilt, ".\cmd\rnparmor")

    if (Test-Path $dist) { Remove-Item -Recurse -Force $dist }
    New-Item -ItemType Directory -Force -Path $dist, (Join-Path $dist "licenses") | Out-Null

    Copy-Item $exeBuilt (Join-Path $dist "rnparmor.exe")
    Copy-Item $rnpDllPath (Join-Path $dist "rnp.dll")
    Copy-Item (Join-Path $root "third_party\licenses\rnp-LICENSE.md") (Join-Path $dist "licenses\rnp-LICENSE.md")
    Copy-Item (Join-Path $root "third_party\licenses\sexpp-LICENSE.md") (Join-Path $dist "licenses\sexpp-LICENSE.md")
    Copy-Item (Resolve-LicensePath $depsLicenses "botan-license.txt" (Join-Path $botanSrc "license.txt")) (Join-Path $dist "licenses\botan-license.txt")
    Copy-Item (Resolve-LicensePath $depsLicenses "dirent-LICENSE" $direntLicenseSrc) (Join-Path $dist "licenses\dirent-LICENSE")
    Copy-Item (Resolve-LicensePath $depsLicenses "json-c-COPYING" (Join-Path $jsonSrc "COPYING")) (Join-Path $dist "licenses\json-c-COPYING")
    Copy-Item (Resolve-LicensePath $depsLicenses "zlib-LICENSE" (Join-Path $zlibSrc "LICENSE")) (Join-Path $dist "licenses\zlib-LICENSE")
    Copy-Item (Resolve-LicensePath $depsLicenses "bzip2-LICENSE" (Join-Path $bzSrc "LICENSE")) (Join-Path $dist "licenses\bzip2-LICENSE")
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
    } finally {
        Pop-Location
    }

    # Tests use only the packaged DLL. PATH cannot supply a substitute.
    $savedPath = $env:PATH
    try {
        $env:PATH = "$env:SystemRoot\System32"
        $env:RNPARMOR_LIBRNP = (Resolve-Path (Join-Path $dist "rnp.dll")).Path
        foreach ($name in @("LIB", "INCLUDE", "LIBPATH", "VCINSTALLDIR", "VSINSTALLDIR")) {
            Remove-Item "Env:$name" -ErrorAction SilentlyContinue
        }

        Log "go test"
        $testLog = Join-Path $work "go-test.txt"
        $testErr = Join-Path $work "go-test-err.txt"
        Write-Host "go test with RNPARMOR_LIBRNP=$env:RNPARMOR_LIBRNP"
        $testCmd = @("test", "-count=1", "-timeout", "20m", "./internal/loadlib/", "./internal/rnp/")
        & $go @testCmd 1> $testLog 2> $testErr
        Get-Content $testLog, $testErr -ErrorAction SilentlyContinue
        if ($LASTEXITCODE -ne 0) { throw "go test failed: $LASTEXITCODE" }

        Log "smoke"
        Remove-Item Env:RNPARMOR_LIBRNP -ErrorAction SilentlyContinue
        $smokeCheck = & (Join-Path $dist "rnparmor.exe") "--smoke-test" 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "smoke-test failed: $LASTEXITCODE`n$smokeCheck"
        }
        $smokeText = $smokeCheck | Out-String
        if ($smokeText -notmatch ('(?m)^app_version=' + [regex]::Escape($Version) + '\s*$')) {
            throw "smoke-test did not report app_version=$Version`n$smokeText"
        }
        Write-Host $smokeText

        $proc = Start-Process -FilePath (Join-Path $dist "rnparmor.exe") -WorkingDirectory $dist -PassThru -WindowStyle Hidden
        $null = Wait-Process -Id $proc.Id -Timeout 12 -ErrorAction SilentlyContinue
        $running = -not $proc.HasExited
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
    } finally {
        $env:PATH = $savedPath
    }

    Log "zip"
    $os = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"
    $head = (& git.exe -C $root rev-parse HEAD).Trim()
    if (Test-Path $zip) { Remove-Item -Force $zip }
    Push-Location (Split-Path $dist -Parent)
    try {
        Compress-Archive -Path (Split-Path $dist -Leaf) -DestinationPath $zip -Force
    } finally {
        Pop-Location
    }

    $zipHash = Get-Hash $zip "SHA256"
    $listing = (Get-ChildItem -Recurse -File $dist | ForEach-Object {
        "{0}`t{1}" -f $_.Length, $_.FullName.Substring($dist.Length + 1)
    }) -join "`n"

@"
commit $head
version $Version
os $($os.ProductName) build $($os.CurrentBuild) display $($os.DisplayVersion)
go $(& $go version)
dll $rnpDllPath
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
}

try {
    switch ($Stage) {
        "fetch-deps" { Invoke-TimedStage "fetch-deps" { Build-FetchDeps } }
        "botan" { Invoke-TimedStage "botan" { Build-Botan } }
        "small-deps" { Invoke-TimedStage "small-deps" { Build-SmallDeps } }
        "deps" { Invoke-TimedStage "deps" { Build-Deps } }
        "rnp" { Invoke-TimedStage "rnp" { Build-Rnp } }
        "package" { Invoke-TimedStage "package" { Build-Package } }
        "all" {
            Invoke-TimedStage "deps" { Build-Deps }
            Invoke-TimedStage "rnp" { Build-Rnp }
            Invoke-TimedStage "package" { Build-Package }
        }
        default { throw "unsupported stage: $Stage" }
    }

    Log "build-windows.ps1 finished, stage=$Stage"
} catch {
    Emit-Error ("build-windows-$Stage") $_.Exception.Message
    throw
}
