# RnpArmor

English | [简体中文](README.zh-CN.md)

RnpArmor is a MyGo-native OpenPGP GUI powered by `librnp v0.18.1`, focused on a Windows 10/11 portable distribution (`rnparmor.exe` + `rnp.dll` in one folder) while keeping Linux build/test support.

> License status: **project license not selected yet**. Third-party licenses are included in portable bundles.

## Highlights

1. **MyGo native GUI with 5 workspaces + live language switch (English / Simplified Chinese / Japanese)**
   - Workspace tabs:
     - Keys & generation
     - Text encrypt/decrypt/sign/verify
     - File encrypt/decrypt/sign/verify
     - Packet inspection & feature matrix
     - Portable keyring storage
   - Built-in zero-dependency UI dictionary at `cmd/rnparmor/i18n.go`.
   - Real-time language switch from the top bar (`English | Chinese | Japanese`).
   - Language preset by `--lang en|zh|ja`, `RNPARMOR_LANG`, or system `LANG`.
   - Added a **menu-less standalone About page** (no workspace tabs in About view), showing:
     - Application Version: injected at build time (`appVersion`, e.g., `v0.0.1` or `dev`)
     - Author: `r404r (https://github.com/r404r)`
     - Repository URL: `https://github.com/r404r/RnpArmor`
     - RNP upstream: `https://github.com/rnpgp/rnp` + current `librnp` version
     - Local library path: current `rnp.dll` / `librnp.so` absolute path
     - Keyring directory address: current portable workspace path + `pubring.gpg`/`secring.gpg` existence
     - Runtime stack: MyGo / purego / Botan versions

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

4. **Windows portable build + smoke flow hardening**
   - `rnparmor.exe --smoke-test` added for deterministic CI smoke checks.
   - `--smoke-test` now also prints `app_version=<value>` and is validated by CI.
   - Windows script smoke path now runs headless smoke-test first, then bounded GUI probe.
   - CI dependency cache added (`out/deps`) for Linux and Windows jobs.

5. **Automated release workflow for portable artifacts**
   - New `.github/workflows/release.yml` runs on `push` tag `v*` and `workflow_dispatch`.
   - Builds both Linux and Windows portable bundles and uploads them as artifacts.
   - Publishes/updates GitHub Release with:
     - `rnparmor-windows-amd64.zip`
     - `rnparmor-linux-amd64.tar.gz`
     - `windows-report.txt`
     - `rnparmor-${TAG}-SHA256SUMS.txt`

## Current Architecture (at a glance)

- `cmd/rnparmor/`
  - `main.go`: MyGo app entry + 5-workspace UI + async session operations + build-time `appVersion` display/output.
  - `i18n.go`: tri-lingual catalog (EN/ZH/JA), language resolution, translation validation.
- `internal/rnp/`
  - `api.go`: purego FFI bindings to librnp APIs.
  - `session.go`: session lifecycle, passphrase callback controls, API models.
  - `ops.go`: key management + text/file crypto/sign/verify + packet/features.
  - `session_test.go`: compatibility tests + expanded integration scenarios.
- `scripts/build-windows.ps1`
  - builds pinned deps and portable output, runs tests and smoke steps.
- `.github/workflows/smoke.yml`
  - Linux + Windows portable jobs with cache + per-branch concurrency cancel.
- `.github/workflows/release.yml`
  - tag-driven/manual portable packaging and release publication.

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

### 6) Menu-less About page

- Entered from the top-bar About button.
- In About view, workspace navigation tabs are hidden by design.
- Includes only:
  - back button (`← Back`) and language switcher
  - read-only details for author/repo/RNP upstream/local library path/keyring directory/runtime stack
- Explicitly states that system default keyring is not read.

## Verification Model

Verification output intentionally distinguishes:

- **Math validity** (signature cryptographically matches content/key)
- **Identity trust** (who that signer is, and whether trusted in policy)

RnpArmor reports the first; it does **not** automatically elevate to the second.

## CLI

```bash
# Probe librnp and validate i18n catalog; no GUI loop
rnparmor --smoke-test

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

### Windows (portable)

```powershell
# one-shot (compatible): deps + rnp + package
pwsh -File .\scripts\build-windows.ps1

# staged mode (for CI observability / caching)
pwsh -File .\scripts\build-windows.ps1 -Stage deps
pwsh -File .\scripts\build-windows.ps1 -Stage rnp
pwsh -File .\scripts\build-windows.ps1 -Stage package

# inject app version into rnparmor.exe metadata shown by --smoke-test/About
pwsh -File .\scripts\build-windows.ps1 -Version v1.2.3

# stage + version together
pwsh -File .\scripts\build-windows.ps1 -Stage package -Version v1.2.3
```

Windows dependency build uses a minimized Botan static module list from
`scripts/windows/botan3-modules.txt` (one module per line, comments allowed),
matching `rnpgp/rnp v0.18.1` requirements while avoiding unnecessary CLI/tests
and non-OpenPGP modules in CI.

`scripts/build-windows.ps1` now supports stage execution:

1. `deps`: build/cache Botan + json-c + zlib + bzip2 into `out/deps` (with cached licenses)
2. `rnp`: build `librnp` and cache `rnp.dll` to `out/win-build/rnp-dll/rnp.dll`
3. `package`: build `rnparmor.exe`, run tests/smoke/probe, and produce zip/report

The package stage creates `out\dist\rnparmor-windows-amd64` and executes:

1. executes `rnparmor.exe --smoke-test`
2. runs a bounded GUI process probe

Zip output: `out\rnparmor-windows-amd64.zip`.

## CI

Workflow: `.github/workflows/smoke.yml`

- `linux` job:
  - dependency cache on `out/deps`
  - pinned librnp build + test matrix
- `windows-portable` job:
  - dependency cache on `out/deps` with `actions/cache/restore` + post-deps `actions/cache/save`
  - staged Windows flow (`deps` -> `rnp` -> `package`) for per-phase timing visibility
  - cache key: `${{ runner.os }}-deps-v3-${{ hashFiles('deps/pins.json', 'scripts/windows/**') }}`

Workflow: `.github/workflows/release.yml`

- triggers:
  - `push` tags matching `v*`
  - `workflow_dispatch` (optional `tag` input)
- jobs:
  - `linux-portable`: build/test/package `rnparmor-linux-amd64.tar.gz`
  - `windows-portable`: build/test/package `rnparmor-windows-amd64.zip` + `windows-report.txt`
  - `publish-release`: on both jobs green, create/update Release and upload assets with combined checksums

## Security & Operational Boundaries

- Dynamic library loading is constrained and integrity-checked.
- No implicit default keyring read/write path.
- Passphrase callback re-entry is blocked by design.
- Large input/output caps are enforced in memory paths.
- Portable test guidance remains: do not use real production keys/passphrases in this stage.

## Repository Status Notes

- This repository started from a minimal slice and is under active expansion.
- Windows desktop manual UX validation and IME/high-DPI verification are still environment-dependent and must be confirmed in dedicated manual runs.

## Third-Party and Pins

See:

- `deps/pins.json`
- `third_party/NOTICES.md`
- portable bundle `licenses/`
