# RnpArmor

English | [简体中文](README.zh-CN.md)

RnpArmor is a MyGo-native OpenPGP GUI powered by `librnp v0.18.1`, delivering Windows 10/11 portable & installer builds (`rnparmor.exe` + `rnp.dll`), macOS Apple Silicon (`arm64`) `.app`/`.dmg` bundles, and Linux (`amd64`) portable builds/tests.

> License status: **project license not selected yet**. Third-party licenses are included in distributed bundles.

## Highlights

1. **MyGo native GUI with 5 workspaces + native 4-group tri-lingual menu bar + standalone About window**
   - Workspace tabs:
     - Keys & generation
     - Text encrypt/decrypt/sign/verify
     - File encrypt/decrypt/sign/verify
     - Packet inspection & feature matrix
     - Portable keyring storage
   - Built-in zero-dependency UI dictionary at `cmd/rnparmor/i18n.go`.
   - Native top menu bar with 4 tri-lingual groups (`File` / `Workspace` / `Language` / `Help`):
     - `File`: Open session, Load keyring, Save keyring, Quit.
     - `Workspace`: 5 radio items synchronized bi-directionally with the workspace tabs.
     - `Language`: radio items (`English`, `中文`, `日本語`) that switch and re-render the menu bar, main window, and About window live.
     - `Help`: opens the standalone About child window (`About`).
   - Language preset by `--lang en|zh|ja`, `RNPARMOR_LANG`, or system `LANG`.
   - **Standalone menu-less non-modal About child window (`580×520`)**:
     - Single-instance focused, fixed-size, no menu bar (`SetMenu(nil)` on non-macOS), includes a `Close` button.
     - Closing the About window leaves the main window unaffected; closing the main window closes the About window automatically; switching language updates its title and content live.
     - Shows Application Version (`appVersion`, e.g., `v0.0.1` or `dev`), Author (`r404r (https://github.com/r404r)`), Repository URL (`https://github.com/r404r/RnpArmor`), RNP upstream (`https://github.com/rnpgp/rnp` + current `librnp` version), Local library path (`rnp.dll` / `librnp.so`), Keyring directory (`portable workspace` + `pubring.gpg`/`secring.gpg` existence), and UI Framework & Runtime (`MyGo` / `purego` / `Botan` versions).

2. **Expanded RNP feature surface in `internal/rnp`**
   - Key iteration/location, key import/export/remove, key-to-JSON.
   - JSON-driven key generation (`rnp_generate_key_json`) with multiple algorithm families.
   - Text operations: encrypt/decrypt, clearsign/detached/inline sign/verify.
   - File operations with option-rich APIs.
   - Packet dump (text/JSON) and feature querying (`rnp_supported_features`, `rnp_supports_feature`).
   - Extended verification metadata (status/hash/signer/timestamps), with explicit separation of:
     - cryptographic math validity
     - identity trust decision (not asserted by default)

3. **Portable-focused safety boundaries**
   - Absolute-path library loading + `SHA256SUMS` verification before loading.
   - Windows `LoadLibraryEx` restricted search semantics.
   - Explicit keyring load/save only; no implicit default keyring.
   - Non-overwrite output semantics for file publishing.

4. **Windows portable build + GUI subsystem + smoke flow hardening**
   - Windows `rnparmor.exe` is built with `-H=windowsgui` (Windows GUI subsystem) so launching it does not pop up a black console window.
   - `scripts/build-windows.ps1` verifies the PE subsystem is `Windows GUI` via `dumpbin /headers` and records the result in `windows-report.txt`.
   - `rnparmor.exe --smoke-test` supports `--smoke-out <path>` to write smoke output (`status=ok`, `app_version=<value>`, `library=...`, `version=...`, `i18n=ok`) to a file for deterministic CI validation under `-H=windowsgui`.
   - Windows script smoke path runs `--smoke-test --smoke-out` first, then a bounded GUI probe.
   - CI dependency cache (`out/deps`) is configured for Linux and Windows jobs.

5. **Automated multi-platform release workflow**
   - `.github/workflows/release.yml` runs on `push` tag `v*` and `workflow_dispatch`.
   - Builds macOS Apple Silicon (`arm64`), Linux (`amd64`), and Windows (`amd64`) artifacts.
   - Publishes/updates GitHub Release with:
     - `rnparmor-macos-arm64.dmg` (ad-hoc signed `RnpArmor.app`, minimum macOS 13.0)
     - `rnparmor-linux-amd64.tar.gz`
     - `rnparmor-windows-amd64-setup.exe` (Inno Setup 6 per-user installer)
     - `rnparmor-windows-amd64.zip` (Windows portable bundle)
     - `windows-report.txt`
     - `rnparmor-${TAG}-SHA256SUMS.txt`

## Current Architecture (at a glance)

- `cmd/rnparmor/`
  - `main.go`: MyGo app entry, native 4-group menu bar, 5-workspace UI, `580×520` non-modal About child window, async session operations, `--smoke-out`, and build-time `appVersion` display/output.
  - `i18n.go`: tri-lingual catalog (EN/ZH/JA), language resolution, translation validation.
  - `main_test.go`: unit tests for smoke output generation/writing, menu template structure, and state synchronization.
- `internal/loadlib/`
  - `open.go`: absolute-path `librnp` resolution and `SHA256SUMS` verification before loading, including fallback from `Contents/MacOS/SHA256SUMS` to `Contents/Resources/SHA256SUMS` for macOS `.app` bundles.
- `internal/rnp/`
  - `api.go`: purego FFI bindings to librnp APIs.
  - `session.go`: session lifecycle, passphrase callback controls, API models.
  - `ops.go`: key management + text/file crypto/sign/verify + packet/features.
  - `session_test.go`: compatibility tests + expanded integration scenarios.
- `scripts/build-windows.ps1`
  - builds pinned deps, `-H=windowsgui` portable output, and Inno Setup 6 installer (`rnparmor-windows-amd64-setup.exe`), verifies PE subsystem and DLL imports, and runs tests + `--smoke-out` + GUI probe.
- `scripts/build-macos.sh`
  - builds pinned static deps (`Botan 3.10.0` + `json-c 0.19`), `librnp.dylib`, and `RnpArmor.app`, runs ad-hoc `codesign --verify --deep --strict`, `go test`, and `--smoke-test`, and packages `rnparmor-macos-arm64.dmg`.
- `packaging/windows/rnparmor.iss`
  - Inno Setup 6 script for the Windows per-user installer (`rnparmor-windows-amd64-setup.exe`).
- `.github/workflows/smoke.yml`
  - `linux`, `windows-portable`, and `macos-arm64` jobs with dependency caches + per-branch concurrency cancel.
- `.github/workflows/release.yml`
  - tag-driven/manual multi-platform packaging and release publication.

## GUI Workspaces

### 1) Keys & generation

- Open session with in-memory passphrase.
- Generate keys via spec (RSA / ECC variants through JSON-based generation path).
- List key summaries, export public/secret, show key JSON, delete key.
- Save/load explicit keyring files (`pubring.gpg` / `secring.gpg`).

### 2) Text studio

- Encrypt text (recipient and/or password modes, optional sign-on-encrypt).
- Decrypt text.
- Sign text in three modes:
  - cleartext
  - detached
  - inline
- Verify text with signature metadata output.

### 3) File workbench

- Encrypt/decrypt files.
- Sign/verify files (detached and inline flows).
- Keeps non-overwrite publish behavior aligned with internal file safety utilities.

### 4) Packet & features

- Dump OpenPGP packets in text or JSON.
- Query supported algorithms/features globally or by `type:name`.

### 5) Portable keyring storage

- Explicit workspace-based load/save model.
- Keeps passphrase usage in-memory for active operations only.

### 6) Native 4-group menu bar & standalone About child window

- Top menu bar groups: `File` (session/keyring/quit), `Workspace` (5 radio items synced with workspace tabs), `Language` (`English` / `中文` / `日本語` radio items), and `Help` (`About`).
- `Help → About` opens a `580×520` non-modal, fixed-size, menu-less standalone child window (single-instance focused, with a `Close` button).
- Closing the About window leaves the main window running; closing the main window closes the About window automatically; live language switches update both windows immediately.
- Displays read-only details (app version, author, repository, RNP upstream, local library path, portable keyring directory + `pubring.gpg`/`secring.gpg` existence, runtime stack) and explicitly notes that the system default keyring is not read.

## Verification Model

Verification output intentionally distinguishes:

- **Math validity** (signature cryptographically matches content/key)
- **Identity trust** (who that signer is, and whether trusted in policy)

RnpArmor reports the first; it does **not** automatically elevate to the second.

## CLI

```bash
# Probe librnp and validate i18n catalog; no GUI loop
rnparmor --smoke-test

# Write smoke output to a file (used by CI on Windows -H=windowsgui builds)
rnparmor --smoke-test --smoke-out out/smoke-out.txt

# Force UI language
rnparmor --lang en
rnparmor --lang zh
rnparmor --lang ja
```

Expected smoke output includes:

```text
status=ok
app_version=dev
library=...
version=...
i18n=ok
```

Language preset fallback order:

1. `--lang`
2. `RNPARMOR_LANG`
3. `LANG`
4. default `en`

## Build & Test

### Linux

```bash
bash scripts/build-librnp.sh
export RNPARMOR_LIBRNP="$PWD/out/lib/librnp.so"
go test ./...
CGO_ENABLED=1 go test -race -gcflags=all=-d=checkptr=0 ./...
CGO_ENABLED=0 go build -o rnparmor ./cmd/rnparmor
```

### macOS (Apple Silicon / arm64)

```bash
# one-shot: runs deps -> rnp -> package (default --stage all)
bash scripts/build-macos.sh

# staged mode (for local iteration or CI caching)
bash scripts/build-macos.sh --stage deps
bash scripts/build-macos.sh --stage rnp
bash scripts/build-macos.sh --stage package

# inject app version into Info.plist and binary metadata
bash scripts/build-macos.sh --stage package --version v0.0.1
```

`scripts/build-macos.sh` targets `MACOSX_DEPLOYMENT_TARGET=13.0` (`LSMinimumSystemVersion=13.0`) on Apple Silicon (`arm64`):

- `deps`: builds static `Botan 3.10.0` (using the same 57-module list in `scripts/windows/botan3-modules.txt`, stamped with `.modules-sha256`) and static `json-c 0.19` into `out/deps`.
- `rnp`: builds `librnp.dylib` (`@loader_path/librnp.dylib`) linked statically against Botan and `json-c`, and dynamically against macOS system `/usr/lib/libz` and `/usr/lib/libbz2` plus `-framework Security` (verified via `otool -L` to avoid external Homebrew paths).
- `package`: builds `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64` `rnparmor`, assembles `out/dist/RnpArmor.app` (`Contents/MacOS/rnparmor`, `Contents/MacOS/librnp.dylib`, `Contents/Resources/SHA256SUMS`, `Contents/Resources/licenses/`), applies ad-hoc signatures and runs `codesign --verify --deep --strict`, executes `go test` and `--smoke-test`, and packages `out/rnparmor-macos-arm64.dmg` via `hdiutil create`.

> macOS Gatekeeper note: `RnpArmor.app` is ad-hoc signed and **not Apple-notarized**. Opening it for the first time from Finder may trigger a Gatekeeper warning (use Control-click → Open or System Settings → Privacy & Security if prompted).

### Windows (portable & installer)

```powershell
# one-shot (compatible): runs deps -> rnp -> package (default -Stage all)
pwsh -File .\scripts\build-windows.ps1

# coarse staged mode
pwsh -File .\scripts\build-windows.ps1 -Stage deps
pwsh -File .\scripts\build-windows.ps1 -Stage rnp
pwsh -File .\scripts\build-windows.ps1 -Stage package

# granular dependency stages (used by CI for timing observability and cache reuse)
pwsh -File .\scripts\build-windows.ps1 -Stage fetch-deps
pwsh -File .\scripts\build-windows.ps1 -Stage botan
pwsh -File .\scripts\build-windows.ps1 -Stage small-deps

# inject app version into rnparmor.exe metadata shown by --smoke-test/About
pwsh -File .\scripts\build-windows.ps1 -Version v0.0.1

# stage + version together
pwsh -File .\scripts\build-windows.ps1 -Stage package -Version v0.0.1
```

Windows dependency builds pin and bundle:

- `Botan 3.10.0`: built as a minimized static library from the 57-module list in `scripts/windows/botan3-modules.txt` (one module per line, comments allowed), matching `rnpgp/rnp v0.18.1` OpenPGP requirements while skipping unnecessary CLI/tests and non-OpenPGP modules in CI.
- `json-c 0.19`, `zlib 1.4`, and `bzip2 1.0.8`: built as static libraries linked into `rnp.dll`.
- `dirent 1.24` (`tronkko/dirent`, MIT): Windows header compatibility dependency (`dirent.h`) used during the `librnp` build.
- All third-party license texts (`rnp-LICENSE.md`, `sexpp-LICENSE.md`, `botan-license.txt`, `dirent-LICENSE`, `json-c-COPYING`, `zlib-LICENSE`, `bzip2-LICENSE`, `mygo-MIT.txt`, and `purego-Apache-2.0.txt`) are bundled into `licenses/` inside the Windows portable package.

`scripts/build-windows.ps1` supports the following `-Stage` values:

1. `fetch-deps`: download, checksum-verify, and extract `Botan 3.10.0`, `json-c 0.19`, `zlib 1.4`, `bzip2 1.0.8`, and `dirent 1.24` into `out/src` (skipped when `out/deps` hot cache is complete)
2. `botan`: build and install minimized static `Botan 3.10.0` into `out/deps/botan` (stamped with `.modules-sha256`)
3. `small-deps`: build and install static `json-c`, `zlib`, and `bzip2` into `out/deps`, stage `dirent.h`, and cache dependency licenses under `out/deps/licenses`
4. `deps`: execute `fetch-deps -> botan -> small-deps`
5. `rnp`: build `librnp` (`rnp.dll`) using cached dependencies and cache `rnp.dll` at `out/win-build/rnp-dll/rnp.dll`
6. `package`: build `rnparmor.exe` with `-H=windowsgui`, copy `rnp.dll` and `licenses/`, verify `Windows GUI` PE subsystem (`dumpbin /headers`) and DLL imports (`DEPENDENCIES.txt`), generate `PORTABLE.txt` and `SHA256SUMS`, run `go test` against packaged `rnp.dll`, run `--smoke-test --smoke-out` and the bounded GUI probe, and produce the portable zip, Inno Setup 6 installer, and report
7. `all`: execute `deps -> rnp -> package` (default)

The `package` stage creates `out\dist\rnparmor-windows-amd64`, builds the installer via `packaging/windows/rnparmor.iss` + Inno Setup 6 (`ISCC.exe`), and executes:

1. `dumpbin /headers` check confirming `rnparmor.exe` targets the `Windows GUI` subsystem (recorded in `out\windows-report.txt`)
2. `go test -count=1 -timeout 20m ./internal/loadlib/ ./internal/rnp/ ./cmd/rnparmor/` with `RNPARMOR_LIBRNP` pointing to the packaged `rnp.dll` and `PATH` restricted to `System32`
3. `rnparmor.exe --smoke-test --smoke-out <path>` (verifying `status=ok`, `app_version=<Version>`, and `i18n=ok`)
4. a bounded GUI process probe

Outputs: `out\rnparmor-windows-amd64.zip`, `out\rnparmor-windows-amd64-setup.exe` (default per-user non-admin install with `PrivilegesRequired=lowest`, Start Menu entry, optional desktop shortcut, and standard uninstaller), and `out\windows-report.txt`.

## CI

Workflow: `.github/workflows/smoke.yml`

- `linux` job:
  - dependency cache on `out/deps`
  - pinned `librnp` build + `go test -race` + `CGO_ENABLED=0` build
- `windows-portable` job:
  - dependency cache on `out/deps` with `actions/cache/restore@v4` before `fetch-deps` and `actions/cache/save@v4` after `small-deps`
  - staged Windows flow (`fetch-deps -> botan -> small-deps -> save cache -> rnp -> package`) for per-stage timing observability and dependency cache reuse
  - cache key: `${{ runner.os }}-deps-v4-${{ hashFiles('deps/pins.json', 'scripts/windows/**') }}`
  - uploads `out/rnparmor-windows-amd64.zip`, `out/rnparmor-windows-amd64-setup.exe`, and `out/windows-report.txt`
- `macos-arm64` job (`runs-on: macos-15`):
  - dependency cache on `out/deps` with key `${{ runner.os }}-arm64-deps-v1-${{ hashFiles('deps/pins.json', 'scripts/windows/botan3-modules.txt', 'scripts/build-macos.sh') }}`
  - staged macOS flow (`deps -> save cache -> rnp -> package`)
  - uploads `out/rnparmor-macos-arm64.dmg`

Workflow: `.github/workflows/release.yml`

- triggers:
  - `push` tags matching `v*`
  - `workflow_dispatch` (optional `tag` input)
- jobs:
  - `linux-portable`: build/test/package `rnparmor-linux-amd64.tar.gz`
  - `windows-portable`: uses the same `deps-v4` cache key and staged sequence (`fetch-deps -> botan -> small-deps -> save cache -> rnp -> package -Version "${{ steps.tag.outputs.value }}"`) to produce `rnparmor-windows-amd64.zip`, `rnparmor-windows-amd64-setup.exe`, and `windows-report.txt`
  - `macos-arm64` (`runs-on: macos-15`): uses the same `arm64-deps-v1` cache key and staged sequence (`deps -> save cache -> rnp -> package --version "${{ steps.tag.outputs.value }}"`) to produce `rnparmor-macos-arm64.dmg`
  - `publish-release`: on all three platform jobs green, creates/updates GitHub Release and uploads `rnparmor-macos-arm64.dmg`, `rnparmor-linux-amd64.tar.gz`, `rnparmor-windows-amd64-setup.exe`, `rnparmor-windows-amd64.zip`, `windows-report.txt`, and `rnparmor-${TAG}-SHA256SUMS.txt`

## Security & Operational Boundaries

- Dynamic library loading is constrained and integrity-checked.
- No implicit default keyring read/write path.
- Passphrase callback re-entry is blocked by design.
- Large input/output caps are enforced in memory paths.
- Portable test guidance remains: do not use real production keys/passphrases in this stage.

## Repository Status Notes

- On `feat/multiplatform-release` (PR #5), all three CI targets (`linux`, `windows-portable`, and `macos-arm64` in `.github/workflows/smoke.yml`) are passing and `.github/workflows/release.yml` is ready for the `v0.0.1` multi-platform release.
- Desktop manual UX validation—including Windows 10/11 Chinese IME, high-DPI, and Inno Setup installer flow, plus macOS Apple Silicon first-launch Gatekeeper handling—remains environment-dependent and must be confirmed in dedicated manual runs.

## Third-Party and Pins

See:

- `deps/pins.json`
- `third_party/NOTICES.md`
- portable bundle `licenses/`
