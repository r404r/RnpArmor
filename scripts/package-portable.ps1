# Assemble a single-directory portable layout from an already built exe
# and a directory of DLLs. Every file named by SHA256SUMS is checked
# before it is copied. This does not publish a GitHub release.
# Not executed in the Linux verification environment.
param(
    [Parameter(Mandatory = $true)][string]$Exe,
    [Parameter(Mandatory = $true)][string]$LibDir,
    [Parameter(Mandatory = $true)][string]$OutDir
)

$ErrorActionPreference = "Stop"

foreach ($p in @($Exe, $LibDir, $OutDir)) {
    if (-not [System.IO.Path]::IsPathRooted($p)) {
        throw "path must be absolute: $p"
    }
}
$Exe = [System.IO.Path]::GetFullPath($Exe)
$LibDir = [System.IO.Path]::GetFullPath($LibDir)
$OutDir = [System.IO.Path]::GetFullPath($OutDir)
if (-not (Test-Path -LiteralPath $Exe -PathType Leaf)) { throw "exe missing" }
if (-not (Test-Path -LiteralPath $LibDir -PathType Container)) { throw "lib dir missing" }

$sumsPath = Join-Path $LibDir "SHA256SUMS"
if (-not (Test-Path -LiteralPath $sumsPath -PathType Leaf)) {
    throw "SHA256SUMS missing beside the DLLs"
}

function Test-SumLine([string]$line) {
    $trim = $line.Trim()
    if ($trim.Length -eq 0 -or $trim.StartsWith("#")) { return $null }
    $parts = $trim -split "\s+", 2
    if ($parts.Count -ne 2) { throw "malformed checksum line" }
    $name = $parts[1].Trim()
    if ($name.Contains("\") -or $name.Contains("/") -or $name -eq ".." -or $name -eq ".") {
        throw "checksum name must be a single file name"
    }
    $path = Join-Path $LibDir $name
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash.ToLower()
    if ($hash -ne $parts[0].ToLower()) { throw "checksum mismatch: $name" }
    return $path
}

$files = @()
foreach ($line in Get-Content -LiteralPath $sumsPath) {
    $path = Test-SumLine $line
    if ($null -ne $path) { $files += $path }
}
$haveDll = $false
foreach ($f in $files) {
    if ([System.IO.Path]::GetFileName($f) -eq "rnp.dll") { $haveDll = $true }
}
if (-not $haveDll) { throw "SHA256SUMS does not list rnp.dll" }

if (Test-Path -LiteralPath $OutDir) { throw "output directory already exists" }
New-Item -ItemType Directory -Path $OutDir | Out-Null
Copy-Item -LiteralPath $Exe -Destination (Join-Path $OutDir ([System.IO.Path]::GetFileName($Exe)))
foreach ($f in $files) {
    Copy-Item -LiteralPath $f -Destination (Join-Path $OutDir ([System.IO.Path]::GetFileName($f)))
}
Copy-Item -LiteralPath $sumsPath -Destination (Join-Path $OutDir "SHA256SUMS")

$note = @"
RnpArmor portable directory.
Start rnparmor.exe from this folder. It loads rnp.dll beside the executable.
Do not put real keys in this test build.
RnpArmor itself has no chosen license. Third-party notices are in licenses\ if this copy step is followed by build-windows.ps1.
This script only copies an already built exe and DLL. It does not prove Windows 10 or 11.
"@
Set-Content -LiteralPath (Join-Path $OutDir "PORTABLE.txt") -Value $note -Encoding utf8
Write-Host "wrote $OutDir"
