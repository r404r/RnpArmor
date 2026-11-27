# RnpArmor User Guide

English | [简体中文](USER_GUIDE.zh-CN.md) | [README](../README.md)

<p><img src="../assets/icons/rnparmor-256.png" alt="RnpArmor icon" width="88" height="88" /></p>

## 1. Overview & Quick Start

RnpArmor is a native OpenPGP desktop GUI built with MyGo and `librnp v0.18.1`. It is designed for users who want explicit, portable keyring files (`pubring.gpg` and `secring.gpg`) in a chosen folder rather than modifying the system default GnuPG keyring.

Supported release targets:

- **Windows 10/11 (`amd64`)**: Per-user installer (`rnparmor-windows-amd64-setup.exe`) and portable archive (`rnparmor-windows-amd64.zip`).
- **macOS Apple Silicon (`arm64`, macOS 13.0+)**: Application disk image (`rnparmor-macos-arm64.dmg` containing `RnpArmor.app`).
- **Linux (`amd64`)**: Portable archive (`rnparmor-linux-amd64.tar.gz`).

> **Passphrase & Public-Only Keyrings:** RnpArmor starts with an empty in-memory passphrase (no pre-filled default passphrase). You can open a folder, browse, and import a public-only keyring (`pubring.gpg` without `secring.gpg`) without entering a passphrase. In **New Key**, leaving both passphrase fields blank generates an OpenPGP RFC 4880 unprotected secret key (no passphrase required later for signing, decrypting, or subkey operations), whereas entering a passphrase encrypts the secret key; signing, decrypting, or exporting/copying passphrase-protected secret keys uses the masked unlock **Passphrase** in **Portable Keyring** (or the passphrase adopted from **New Key**).

## 2. Download, Installation & First Launch

Release packages are published via `.github/workflows/release.yml` (`v0.0.1`–`v0.0.5` have been released: `v0.0.2` added Apache-2.0 licensing, Linux runtime screenshots, multi-platform icons, and the initial bilingual User Guide; `v0.0.3` fixed the GUI action deadlock and keyring persistence edge cases; `v0.0.4` shipped Phase 1 master-detail Key Manager / native folder picker and Phase 2 keyring pickers / native file browse and save dialogs; `v0.0.5` added PR #14 `+ New Key` algorithm presets/parameter linking/tri-lingual tips and PR #15 hierarchical collapsible subkey tree & full subkey lifecycle management; upcoming empty-passphrase unprotected key generation from `feat/empty-passphrase-keygen` will ship in the next release).

### Verifying Release Checksums

Each release provides a combined checksum file `rnparmor-<TAG>-SHA256SUMS.txt` alongside the platform archives. Verify your download before extracting or installing:

```bash
# Linux / macOS
sha256sum -c rnparmor-<TAG>-SHA256SUMS.txt --ignore-missing
# or on macOS without GNU sha256sum:
shasum -a 256 rnparmor-macos-arm64.dmg
```

```powershell
# Windows (PowerShell)
Get-FileHash .\rnparmor-windows-amd64-setup.exe -Algorithm SHA256
Get-FileHash .\rnparmor-windows-amd64.zip -Algorithm SHA256
```

In addition, every bundle includes an internal `SHA256SUMS` file. Every time RnpArmor starts, `internal/loadlib` verifies the shared library (`rnp.dll`, `librnp.so`, or `librnp.dylib`) against `SHA256SUMS` before loading it.

### Windows Installer (`rnparmor-windows-amd64-setup.exe`)

1. Download `rnparmor-windows-amd64-setup.exe`.
2. Double-click to run the Inno Setup installer:
   - Defaults to **per-user installation without administrator privileges** (`PrivilegesRequired=lowest`, overridable in the dialog).
   - Creates a **Start Menu** entry (`RnpArmor`), an optional **Desktop shortcut** (unchecked by default), and registers standard Windows uninstallation (`RnpArmor`).
3. Launch **RnpArmor** from the Start Menu or Desktop shortcut.

### Windows Portable (`rnparmor-windows-amd64.zip`)

1. Extract `rnparmor-windows-amd64.zip` into a dedicated folder (keep `rnparmor.exe`, `rnp.dll`, `SHA256SUMS`, `LICENSE`, and `licenses\` together).
2. Double-click `rnparmor.exe`:
   - Built with `-H=windowsgui` (Windows GUI subsystem) and an embedded `.ico` resource, so **no black console window pops up**.
   - Loads `rnp.dll` strictly from the same folder using `LoadLibraryEx` restricted search flags (never searches the current working directory or `PATH`).
3. **Windows SmartScreen note**: Because CI artifacts are not Authenticode-signed yet, Windows Defender SmartScreen may show *"Windows protected your PC"*. Verify the file SHA-256 against `rnparmor-<TAG>-SHA256SUMS.txt`, then click **More info** → **Run anyway** if you trust the download.

### macOS Apple Silicon (`rnparmor-macos-arm64.dmg`)

1. Open `rnparmor-macos-arm64.dmg`.
2. Drag `RnpArmor.app` into the `Applications` alias.
3. **First launch & Gatekeeper note**: `RnpArmor.app` is ad-hoc signed and **not Apple-notarized**. Double-clicking it the first time may trigger a Gatekeeper prompt:
   - Right-click (or **Control-click**) `RnpArmor.app` in Finder → choose **Open** → confirm **Open**, or allow it under **System Settings → Privacy & Security**.
4. Inside `RnpArmor.app`, `Contents/MacOS/rnparmor` verifies `Contents/MacOS/librnp.dylib` against `Contents/Resources/SHA256SUMS` automatically on launch.

### Linux Portable (`rnparmor-linux-amd64.tar.gz`)

1. Extract the archive and enter the extracted directory:

```bash
tar -xzf rnparmor-linux-amd64.tar.gz
cd rnparmor-linux-amd64
./rnparmor
```

The bundle contains `rnparmor`, `librnp.so`, `libbotan-3.so.10`, `libjson-c.so.5`, `SHA256SUMS`, `rnparmor.png`, `LICENSE`, and `licenses/`.

### CLI Flags & Environment Variables

| Option / Variable | Purpose | Details |
| --- | --- | --- |
| `--lang en\|zh\|ja` | Preset UI language | Precedence: `--lang` > `RNPARMOR_LANG` > `LANG` > `en` |
| `RNPARMOR_LANG` | Language environment variable | Supports `en`, `zh`, `ja` (and locale prefixes like `zh_CN.UTF-8`) |
| `--smoke-test` | Headless library & i18n check | Probes `librnp`, checks `SHA256SUMS` and translations, prints status, exits `0` without starting GUI |
| `--smoke-out <path>` | Write smoke output to file | Creates parent dirs if needed; useful on Windows `-H=windowsgui` where stdout has no console |
| `RNPARMOR_LIBRNP` | Override shared library path | Must be an **absolute path**; target directory (or `Contents/Resources` on macOS) must still provide a valid `SHA256SUMS` |

Example `--smoke-test` output:

```text
status=ok
app_version=v0.0.4
library=/path/to/librnp.so
version=0.18.1+git...
i18n=ok
```

## 3. Interface Overview & Menu Bar

![RnpArmor main window (English, Linux runtime)](screenshots/rnparmor-linux-en.png)

The main window (`1120×780`) displays the title, loaded `librnp` version, absolute library path, live status bar (`idle` / `busy` + last action or error), the 5 workspace tabs, and a native 4-group menu bar:

1. **`File`**
   - **Choose Folder...**: opens the native system folder picker, sets **Workspace**, and automatically loads keys from that folder.
   - **Open session**: initializes an in-memory `librnp` session (and loads `pubring.gpg` / `secring.gpg` if present in **Workspace**).
   - **Load keyring**: loads `pubring.gpg` (and `secring.gpg`, if present) from the configured **Workspace** directory.
   - **Save keyring**: writes `pubring.gpg` (and `secring.gpg` when secret keys exist) to the **Workspace** directory.
   - **Quit**: closes the application.
2. **`Workspace`**
   - Radio items synchronized bi-directionally with the 5 tabs: **Keys**, **Text Studio**, **File Workbench**, **Packet & Features**, **Portable Keyring**.
3. **`Language`**
   - Live radio switch between **`English`**, **`中文`**, and **`日本語`**. Updates the menu bar, main window title/controls, and open child windows (**New Key**, **Import Key Text**, **About**) immediately without restarting.
4. **`Help`**
   - **About**: opens a standalone, menu-less, non-modal child window (`580×520`, fixed size, single-instance focused) with a **Close** button. Closing About keeps the main window open; closing the main window closes About, New Key, and Import Key Text automatically.
   - Read-only fields shown in About: **Application Version**, **Author**, **Repository URL**, **RNP Upstream & Version**, **Local RNP Library Path**, **Keyring Directory (portable workspace)**, **pubring.gpg exists**, **secring.gpg exists**, **UI Framework & Runtime** (`MyGo v0.10.0 | purego v0.11.1 | Botan 3.10.0`), plus a reminder that the system default keyring is not read.

## 4. Step-by-Step Guide for All 5 Workspaces

> All workspace operations run asynchronously in the background (with a 2-minute timeout). While an operation is running, the status line shows `busy` and ignores duplicate clicks until the call returns.

### Workspace 1: `① Keys`

Use this workspace as a master-detail key manager to browse your keyring folder, create or import keys, inspect primary/subkey metadata, export keys or revocation certificates, and delete keys.

**Toolbar & Filter Bar:**

- **Top Toolbar**:
  - **`+ New Key`**: opens the standalone **`New Key`** window (`560×640`).
  - **`Import File...`**: opens a native file picker filtered for `.asc`, `.pgp`, `.gpg`, and `.key`, imports keys into the active session, and updates disk keyring files if **Workspace** is non-empty.
  - **`Import Text...`**: opens the **`Import Key Text`** child window (`560×420`), where you can paste an ASCII-armored public/secret key or click **`Paste from Clipboard`**, then click **`Import`**.
  - **`Refresh`**: reloads and refreshes the keyring list from **Workspace**.
  - **Workspace chooser**: displays the current **Workspace** path (read-only; defaults to the startup working directory and auto-loads on launch if `pubring.gpg` or `secring.gpg` exists) alongside **`Choose Folder...`**, which opens a native directory chooser and loads the selected folder automatically.
- **Filter, ID Switch & Search Bar**:
  - Ownership filter: **`All`** / **`My Keys`** (keys with a secret key) / **`Public`** (public-only keys).
  - Column ID toggle: **`Key ID (16)`** ↔ **`Fingerprint (40)`**.
  - Search box (**`Search UID, Key ID, fingerprint, algorithm`**): filters primary keys in real time.

**Master Tree Table & Detail Inspector:**

- **Left Pane (Hierarchical Collapsible Key Table)**: top-level rows remain sortable by **`Type`** (`Secret` / `Public`), **`Name / Email`**, **`Key ID`** or **`Fingerprint`**, **`Algorithm`**, and **`Status`** (`Valid` / `Expired` / `Revoked`). Primary keys that have subkeys show a `▾` / `▸` toggle (expanded by default) directly beside their ownership badge:
  - **Expanded Subkey Rows**: appear indented directly under their primary key (`└ Subkey` with usage `Encrypt`, `Sign`, or `Auth`, 16-char Key ID or 40-char Fingerprint synced with the ID toggle, algorithm, and status).
  - **Subkey Search & Selection**: search matches across both primary keys and their subkeys (UID, Key ID, fingerprint, and algorithm). Clicking a subkey row keeps its parent primary key selected in the right pane and highlights that subkey card. When loading or refreshing keys with no prior selection, RnpArmor automatically prefers a certifiable primary secret key.
- **Right Pane (Detail Inspector)**:
  - **Header & Badges**: primary UID, ownership badge (`Secret+Public` or `Public`), **`No Passphrase`** badge (shown when a secret key is stored without passphrase protection), status badge (`Valid`, `Expired`, or `Revoked`), and capability badges (`Certify`, `Sign`, `Encrypt`, and `Auth` when present).
  - **Metadata**: 40-character **Fingerprint** formatted in 4-character groups with **`Copy Fingerprint`**, 16-character **Key ID**, **Algorithm**, **Created**, and **Expires** (`YYYY-MM-DD` UTC or `Never`).
  - **Subkeys & Lifecycle Management**: lists each subkey card with capability badges (`Encrypt` / `Sign` / `Auth`), **`No Passphrase`** badge (when that subkey secret is unprotected), status badge (`Valid` / `Expired` / `Revoked`), algorithm, 16-char Key ID, **Created** / **Expires** dates, and grouped 40-char fingerprint (with **`Copy Fingerprint`** when highlighted). When the primary key includes secret material, you can manage subkeys individually without modifying other subkeys or revoking the primary key:
    - **`+ Add Subkey...`** (`560×680` modal): choose **Usage** (`Encrypt (recommended)`, `Sign`, or `Authenticate (Auth)`), valid usage-restricted **Algorithm** (`ECDH` / `RSA` / `ELGAMAL` for `Encrypt`; `EDDSA` / `RSA` / `ECDSA` for `Sign` and `Authenticate`), curve or bit size, hash/cipher, **Expires** (`Never expires`, `1 year from now`, `2 years from now`, `5 years from now`), and masked **Session passphrase**. On a passphrase-protected primary key, leave blank to reuse the active session passphrase (or enter it if locked); on an unprotected primary key (`No Passphrase`), **Session passphrase** is optional—leaving it blank creates an unprotected subkey, while typing a passphrase still encrypts that new subkey individually.
    - **`Change Expiry...`** (`480×420` modal): updates only the selected subkey's expiration (`Never expires` or `+1` / `+2` / `+5` years from now, stored as seconds from that subkey's creation time) while keeping the primary key unchanged; **Session passphrase** can be left blank when the primary secret key has no passphrase.
    - **`Revoke...`** (`500×460` modal + native confirmation, hidden once revoked): permanently revokes only that subkey with an RFC reason (`Retired`, `Superseded`, `Compromised`, or `No reason`) and optional note; the primary key remains valid, and **Session passphrase** is optional when the primary secret key is unprotected.
    - **`Delete...`** (native warning confirmation): permanently removes only that subkey from the keyring while keeping the primary key (the backend explicitly refuses deleting a primary key via subkey deletion). Note that deleting an encryption subkey may make historical messages encrypted to it undecryptable.
  - **Public Actions**: **`Copy Public Key`** (copies ASCII-armored public key to clipboard) and **`Export Public (.asc)...`** (native Save dialog).
  - **Secret Actions** *(shown only when the selected key includes a secret key)*:
    - **`Export Secret (.asc)...`**: displays a native warning confirmation prompt first, then opens a Save dialog and writes the file with `0600` permissions.
    - **`Copy Secret`**: copies the secret key to the clipboard using concealed clipboard mode (and wipes the temporary buffer).
    - **`Revocation Cert...`**: saves an ASCII-armored revocation certificate (`<last16fp>.rev.asc`) with `0600` permissions via a Save dialog.
  - **`Delete Key...`**: opens a native warning confirmation dialog showing the key fingerprint and whether only the public key or both public and secret keys will be removed; deleting the last remaining key cleans up empty `pubring.gpg` and `secring.gpg` files on disk.
  - **Collapsible `Raw Key JSON`**: click **`Load JSON`** to inspect raw structured key JSON.

**Creating a Key (`+ New Key` Window):**

1. Enter **`Name`** and/or **`Email`** (at least one is required; both combine as `Name <Email>`).
2. Optionally enter and confirm a masked **`Passphrase`** (*Leave blank for no passphrase*) and **`Confirm Passphrase`** (*Confirm passphrase, or leave both blank*):
   - **Passphrase-protected key**: when both fields match and are non-empty, RnpArmor indicates that the secret key will be encrypted with the passphrase, adopts it into the active session upon generation, and clears the form inputs.
   - **Unprotected key (no passphrase)**: leaving **both** fields blank displays a clear live notice below **`Confirm Passphrase`** (*"Both fields are empty. An unprotected secret key will be generated and stored unencrypted."*) and generates an OpenPGP RFC 4880 unprotected secret key without altering any session passphrase held for other keys. Filling only one field is still rejected as a passphrase mismatch.
   - **Security note**: unprotected secret keys are suited to isolated local environments, automated tests, or scenarios where plaintext-at-rest risk is explicitly accepted; anyone with access to the secret key file can use it directly without a passphrase, so a strong passphrase is still recommended for everyday primary keys.
3. Choose **`Expiration`**: `Never`, `1 Year`, `2 Years`, or `5 Years`.
4. Choose **`Algorithm preset`** (`SHA256` + `AES256` defaults):
   - **`Modern (Ed25519 + Curve25519)`** *(default, recommended)*: `EDDSA` (`Ed25519`) **Primary Key** + `ECDH` (`Curve25519`) **Encryption Subkey**.
   - **`RSA 4096 + RSA 4096`**: `RSA 4096` **Primary Key** + `RSA 4096` **Encryption Subkey** (highest legacy compatibility).
   - **`RSA 3072 + RSA 3072`**: `RSA 3072` **Primary Key** + `RSA 3072` **Encryption Subkey**.
   - **`NIST P-256`**: `ECDSA` (`NIST P-256`) **Primary Key** + `ECDH` (`NIST P-256`) **Encryption Subkey**.
   - **`Custom`**: automatically expands **`Advanced algorithms`** (manually editing any algorithm option away from a named preset also switches the preset to **`Custom`** and expands the section).
5. Optionally expand **`Advanced algorithms`** (each field shows a dynamic inline description and hover tooltip in English, Chinese, and Japanese; parameters automatically link to valid curves or key sizes for the selected algorithm):
   - **Primary Key**: `EDDSA` (fixed `Ed25519`), `RSA` (`2048` / `3072` / `4096` bits), `ECDSA` (`NIST P-256` / `NIST P-384` / `NIST P-521`), or `DSA` (`2048` / `3072` bits; noted in UI tips as legacy/not recommended).
   - **Encryption Subkey**: `ECDH` (`Curve25519` / `NIST P-256` / `NIST P-384` / `NIST P-521`), `RSA` (`2048` / `3072` / `4096` bits), or `ELGAMAL` (`2048` / `3072` / `4096` bits; noted in UI tips as legacy/not recommended).
   - **Crypto**: hash (`SHA256`, `SHA384`, `SHA512`, `SHA3-256`; UI tips recommend `SHA256` or `SHA512`) and symmetric cipher (`AES256`, `AES192`, `AES128`, `Camellia256`; UI tips recommend `AES256`).
6. Click **`Generate Key`**.

**Non-clobbering Keyring Persistence:**

- Read-only actions (**Open session**, **Refresh**, **Load keyring**, **Open+Load**, **Query features**, **Load JSON**, **Copy Public Key** / **Export Public (.asc)...**, **Copy Secret** / **Export Secret (.asc)...**, **Revocation Cert...**, text/file crypto & signing/verification, and **Dump packets**) never rewrite or overwrite `pubring.gpg` / `secring.gpg` on disk.
- When **Workspace** is non-empty, **`Generate Key`**, **Import** (`Import File...` / `Import Text...`), and **`Delete Key...`** automatically persist changes to **Workspace**. Public-only keyrings write only `pubring.gpg` (no `secring.gpg`); keyrings with secret keys write both `pubring.gpg` and `0600` `secring.gpg`; deleting the last remaining key removes empty files.
- Explicit **`File → Save keyring`** / **`Portable Keyring → Save`** writes the active session keys; saving an empty session (`0` keys) returns an error and leaves existing files on disk intact.

### Workspace 2: `② Text Studio`

Use this workspace for ASCII-armored text encryption, decryption, signing, and verification.

**Fields:**

- **Plain text**: defaults to `"hello from RnpArmor"`. You can also paste an external ASCII-armored PGP ciphertext (`-----BEGIN PGP MESSAGE-----`) here to run **Decrypt text**, or paste an external `cleartext` / `inline` signed PGP block (`-----BEGIN PGP SIGNED MESSAGE-----` / `-----BEGIN PGP MESSAGE-----`) here to run **Verify text**.
- **Recipients (comma fp)**:
  - **Keyring picker**: select any loaded primary key from the dropdown (`UID [Key ID]`) and click **`+ Add`** to append its fingerprint to the comma-separated list (case- and whitespace-insensitive deduplication preserves existing entries); click **`Clear`** to empty the list.
  - **Manual entry**: paste or edit comma-separated fingerprints directly in the text box (`Comma-separated fingerprints`). If no keys are loaded yet, you can still paste fingerprints directly.
  - **Fallback**: if both **Recipients** and **Password** are empty, RnpArmor automatically falls back to the currently **Selected** key (preferring a certifiable primary secret key by default).
- **Password**: optional masked symmetric encryption password (placeholder `Symmetric passphrase`; can be combined with recipients, or used alone without falling back to the **Selected** key).
- **Cipher / Compression**: cipher (`AES256`, `AES192`, `AES128`, `Camellia256`; default `AES256`), compression (`zlib`, `bzip2`, `zip`, `uncompressed`; default `zlib`), and **Encrypt+Sign** toggle.
- **Sign mode**: `cleartext`, `detached` (default), or `inline` (attached OpenPGP signature).
- **Signer fingerprint**:
  - **Signer dropdown**: lists loaded primary keys that have a secret key, starting with **`Use selected key (default)`**. Switching entries updates the fingerprint override field; choosing **`Use selected key (default)`** clears the override so **Encrypt+Sign** and **Sign text** use the currently **Selected** key.
  - **Manual override**: paste or type a custom fingerprint in the text field below (`Fingerprint override`) to override without altering the dropdown until changed.
- **Detached verify sig**: paste a detached ASCII-armored signature here when verifying a detached signature against the text in **Plain text**.

**Operations:**

- **Encrypt text**: encrypts **Plain text** to **result output** (and stages the armored message internally for immediate one-click testing with **Decrypt text**). If **Recipients** and **Password** are both blank, encrypts to the currently **Selected** key.
- **Decrypt text**: decrypts an ASCII-armored message (`-----BEGIN PGP MESSAGE-----`) pasted directly into **Plain text**; if **Plain text** does not contain a ciphertext block, falls back to the message staged from **Encrypt text** / **Sign text** (or shown in **result output**), then prints the plaintext to **result output**.
- **Sign text**: signs **Plain text** using the selected **Sign mode** (`cleartext`, `detached`, or `inline`, with `SHA256`) and prints the result to **result output** (for `detached`, also copies the signature into **Detached verify sig**).
- **Verify text**:
  - When **Detached verify sig** is non-empty: verifies **Plain text** (or the staged plaintext if **Plain text** is blank) against **Detached verify sig**.
  - When **Detached verify sig** is empty: verifies a `cleartext` or `inline` PGP block pasted directly into **Plain text**, or falls back to the signed message staged from **Sign text** (or in **result output**).
  - Displays:
    - Mathematical validity badge: `[Math: VALID]` or `[Math: INVALID]`
    - Identity trust badge: `[Identity: UNVERIFIED]` or `[Identity: TRUSTED]` *(always `UNVERIFIED` in this release because RnpArmor reports cryptographic validity only and does not assert policy trust)*
    - Metadata lines: `Status`, `KeyID`, `Fingerprint`, `Hash`, `Created`, `Expires`.

### Workspace 3: `③ File Workbench`

Use this workspace to encrypt, decrypt, sign, and verify files on disk safely using native file pickers and self-contained recipient/signer controls.

**Fields & Native File Dialogs:**

- **Source**: file path text box + **`Browse...`** native Open dialog (defaults to showing **`All files`** first, with **`OpenPGP`** filter available).
- **Destination**: file path text box + **`Save As...`** native Save dialog (defaults to **`All files`** first; starts in **Source**'s directory if **Destination** is blank, falling back to **Workspace**).
- **Signature path**: file path text box + both **`Browse...`** (for selecting an existing signature when verifying) and **`Save As...`** (for choosing output signature path when signing); both default to the **`OpenPGP`** filter (`.asc`, `.pgp`, `.gpg`, `.key`, `.sig`) first and **`All files`** second.
- **Detached verify**: toggle **`Detached`** on when verifying a detached signature file.
- **Recipients (comma fp)**, **Password** (masked), **Signer fingerprint**, and **Encrypt+Sign**: embedded directly in **File Workbench** and shared with **Text Studio**, so you can configure recipients, passphrase, signer, and **Encrypt+Sign** without switching tabs.

**Important Path & Safety Rules:**

- All paths (**Source**, **Destination**, **Signature path**) must be **absolute paths** (the native **`Browse...`** / **`Save As...`** dialogs fill absolute paths automatically).
- **Source** and **Destination** cannot be the same path.
- **Non-overwrite protection**: RnpArmor refuses to overwrite an existing **Destination** or **Signature path** (`refusing to overwrite an existing file`). Choose a new output filename each time.
- **Size limit**: regular files only, non-empty, up to **32 MiB** (`MaxInput`) input and **40 MiB** (`MaxOutput`) in-memory output.

**Operations:**

- **Encrypt file**: pick or enter **Source** and **Destination**, configure **Recipients**, **Password**, or **Encrypt+Sign** right in **File Workbench** (or **Cipher / Compression** in **Text Studio**; falls back to the currently **Selected** key if both recipients and password are blank), then click **Encrypt file**. Output uses ASCII armor automatically when **Destination** ends with `.asc` (otherwise binary OpenPGP).
- **Decrypt file**: pick or enter **Source** (encrypted file) and **Destination** (new output path), then click **Decrypt file**.
- **Sign file**: pick or enter **Source** and **Signature path**, optionally pick a **Signer** (or choose **Sign mode** `detached`/`inline` in **Text Studio**; armored with `SHA256`), and click **Sign file**.
- **Verify file**:
  - For detached verification: toggle **Detached** on, pick **Source** (original file) and **Signature path** (`.asc`/`.sig` file via **`Browse...`**), and click **Verify file**.
  - For inline signed files: toggle **Detached** off, pick **Source** (signed file), and click **Verify file**.

### Workspace 4: `④ Packet & Features`

Use this workspace to inspect OpenPGP packet structures and audit supported algorithms in `librnp`.

- **Dump packets**:
  1. Click **`Browse...`** next to **Packet input file** (native Open dialog defaulting to the **`OpenPGP`** filter `.asc`, `.pgp`, `.gpg`, `.key`, `.sig`) or enter an absolute path (up to 32 MiB).
  2. Toggle **JSON** on for JSON output (default) or off for text output.
  3. Click **Dump packets** to inspect the structure in **Packet output**.
- **Query features**:
  - Leave **Feature query** empty to dump all 6 categories (`symmetric algorithm`, `hash algorithm`, `public key algorithm`, `elliptic curve`, `compression algorithm`, `aead algorithm`).
  - Enter a category name (e.g. `elliptic curve`) or a `type:name` pair (e.g. `public key algorithm:EDDSA`) to check a specific feature, then click **Query features** to view the JSON output under **Features**.

### Workspace 5: `⑤ Portable Keyring`

Use this workspace to load, save, and migrate explicit portable keyrings, or to enter your masked unlock passphrase when using secret keys.

- **Workspace**: read-only path display with **`Choose Folder...`** (native folder picker).
- **Passphrase**: masked text input (`Password` mode, initially empty `""`) with placeholder `"Passphrase to unlock secret keys"`. Leave empty to browse or import a public-only keyring (`pubring.gpg`) or to use keys generated without a passphrase (`No Passphrase`); enter your passphrase here whenever you need to decrypt, sign, or export/copy a passphrase-protected secret key (if not already adopted from creating a key in **New Key**).
- **Open+Load**: loads `pubring.gpg` (and `secring.gpg` if present) from **Workspace** into the session without rewriting the files on disk.
- **Save**: saves the current session keys inside **Workspace** (`pubring.gpg` only for public-only sessions, or `pubring.gpg` + `0600` `secring.gpg` when secret keys are present; creates the directory with `0700` permissions if needed). Saving an empty session (`0` keys) returns an error and leaves existing files untouched.
- **Reload keys**: refreshes the loaded key summaries.

**Backup & Migration:**

1. Close RnpArmor or ensure no operation is `busy`.
2. Copy `pubring.gpg` (and `secring.gpg` if your keyring includes secret keys) from your **Workspace** folder to an encrypted backup drive or another machine. Keep `secring.gpg` strictly private.
3. On the destination machine, open RnpArmor, click **`Choose Folder...`** (or go to **Portable Keyring**, select the folder via **`Choose Folder...`**, enter your masked **Passphrase** if needed for secret keys), then click **Open+Load**.

## 5. Security, Verification & Troubleshooting

### Security Reminders

- **Masked passphrases & concealed secret copies**: RnpArmor never pre-fills a default passphrase (`""` at startup). Passphrases in **New Key** and **Portable Keyring** are masked; **Copy Secret** marks clipboard content as concealed; exported secret key files and `.rev.asc` revocation certificates are written with `0600` permissions on Unix. If you intentionally leave both passphrase fields blank to create a **`No Passphrase`** secret key, its private key packets are stored unencrypted (`secring.gpg` and exports still keep `0600` file mode on Unix, but anyone obtaining the file can use the key without a passphrase); use a strong passphrase for everyday primary keys and reserve unprotected keys for isolated local testing or controlled automation.
- **No implicit system keyring**: RnpArmor never reads or modifies `~/.gnupg` or any default OS keyring.
- **Math validity vs. identity trust**: `[Math: VALID]` confirms the cryptographic signature matches the key and data; `[Identity: UNVERIFIED]` reminds you that RnpArmor does not evaluate Web-of-Trust or policy trust. Always verify key fingerprints out-of-band.
- **In-flight call cancellation**: An active `librnp` C call cannot be interrupted mid-flight; timeouts discard the result only after the C call finishes.

### Common Issues & Fixes

| Symptom | Cause | Resolution |
| --- | --- | --- |
| `librnp bundle checksum failed` on startup | Missing `SHA256SUMS`, modified library, or mismatched `RNPARMOR_LIBRNP` | Re-extract the original release archive without modifying `rnp.dll`/`librnp.so`/`librnp.dylib` or `SHA256SUMS`; if using `RNPARMOR_LIBRNP`, use an absolute path alongside its valid `SHA256SUMS` |
| `path must be absolute` | Relative path entered in **File Workbench** or **Packet input file** | Use **`Browse...`** / **`Save As...`** (or enter a full absolute path, e.g., `C:\Users\name\docs\file.txt` on Windows or `/home/name/file.txt` on Linux/macOS), and **`Choose Folder...`** for **Workspace** |
| `refusing to overwrite an existing file` | Output path in **Destination** or **Signature path** already exists | Choose a new output filename that does not already exist |
| `input exceeds the memory limit` | Input file exceeds 32 MiB (`MaxInput`) | RnpArmor currently buffers files up to 32 MiB in memory |
| Operations fail with `passphrase unavailable` or key-not-found | Unlock **Passphrase** is empty, wrong for `secring.gpg`, or keyring folder not loaded | Enter the matching passphrase in **Portable Keyring**, click **`Choose Folder...`** to select the folder containing `pubring.gpg` (and `secring.gpg` for secret operations), and click **Refresh** / **Open+Load** |
| Windows SmartScreen / macOS Gatekeeper warning | Unsigned Windows binary or ad-hoc signed (unnotarized) macOS app | Verify release SHA-256 checksums first; on Windows click **More info → Run anyway**; on macOS **Control-click → Open** |
| Need CLI diagnostics without GUI | Diagnosing library loading or translations | Run `rnparmor --smoke-test --smoke-out smoke.txt` and inspect `status=ok`, `library=`, `version=`, and `i18n=ok` |

### Licenses & Notices

- RnpArmor project license: **Apache License, Version 2.0** ([`../LICENSE`](../LICENSE)).
- Third-party dependency pins and full license notices: [`../third_party/NOTICES.md`](../third_party/NOTICES.md).
- Distributed bundle locations:
  - Windows & Linux: `LICENSE` and `licenses/` at the bundle root.
  - macOS: `RnpArmor.app/Contents/Resources/LICENSE` and `RnpArmor.app/Contents/Resources/licenses/`.
