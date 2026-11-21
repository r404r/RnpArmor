# Build pinned Botan 3.10.0 and librnp 0.18.1 on Windows.
# This script was not executed in the Linux verification environment.
# A successful run here is a compile of the sources. It is not a test on
# Windows 10 or Windows 11, and it does not publish a release.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$pinsPath = Join-Path $root "deps\pins.json"
$pins = Get-Content -Raw -Path $pinsPath | ConvertFrom-Json
if ($pins.rnp.commit -ne "aad1892e4d8423398a0b973bd0a7e4544c359afc") {
    throw "rnp pin mismatch"
}
if ($pins.rnp.do_not_use -contains "v0.18.0" -eq $false) {
    throw "v0.18.0 must stay rejected"
}
$wantBotan = "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24"

$src = Join-Path $root "out\src"
$deps = Join-Path $root "out\deps"
$lib = Join-Path $root "out\lib"
New-Item -ItemType Directory -Force -Path $src, $deps, $lib | Out-Null

$tarball = Join-Path $src "Botan-3.10.0.tar.xz"
if (-not (Test-Path $tarball)) {
    Invoke-WebRequest -Uri $pins.botan.tarball -OutFile $tarball
}
$got = (Get-FileHash -Algorithm SHA256 -Path $tarball).Hash.ToLower()
if ($got -ne $wantBotan) {
    throw "Botan tarball sha256 $got"
}

if (-not (Get-Command cmake -ErrorAction SilentlyContinue)) {
    throw "cmake is not on PATH. Install a Windows C++ toolchain and cmake, then re-run. Do not copy an unknown DLL into the tree."
}
if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
    throw "python is required to configure Botan"
}

Write-Host "Pinned sources checked. Configure Botan with its own Python script and point rnp at that prefix:"
Write-Host "  BOTAN_ROOT_DIR must be an absolute prefix you just built."
Write-Host "  Do not add the output directory to PATH."
Write-Host "  After the build, put librnp.dll and its non-system dependencies in one directory and run package-portable.ps1."
Write-Host "This script stops before invoking a compiler profile it has not been run against."
exit 2
