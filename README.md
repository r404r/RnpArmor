# RnpArmor

[English](#rnparmor) | [简体中文](#rnparmor-中文文档)

RnpArmor is a MyGo-native OpenPGP GUI powered by `librnp v0.18.1`, focused on a Windows 10/11 portable distribution (`rnparmor.exe` + `rnp.dll` in one folder) while keeping Linux build/test support.

> License status: **project license not selected yet**. Third-party licenses are included in portable bundles.

## Highlights

1. **MyGo native GUI with 5 workspaces + live language switch (English / 中文 / 日本語)**
   - Workspace tabs:
     - Keys & generation
     - Text encrypt/decrypt/sign/verify
     - File encrypt/decrypt/sign/verify
     - Packet inspection & feature matrix
     - Portable keyring storage
   - Built-in zero-dependency UI dictionary at `cmd/rnparmor/i18n.go`.
   - Real-time language switch from the top bar (`English | 中文 | 日本語`).
   - Language preset by `--lang en|zh|ja`, `RNPARMOR_LANG`, or system `LANG`.
   - Added a **menu-less standalone About page** (no workspace tabs in About view), showing:
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
   - Windows script smoke path now runs headless smoke-test first, then bounded GUI probe.
   - CI dependency cache added (`out/deps`) for Linux and Windows jobs.

## Current Architecture (at a glance)

- `cmd/rnparmor/`
  - `main.go`: MyGo app entry + 5-workspace UI + async session operations.
  - `i18n.go`: tri-lingual catalog (EN/ZH/JA), language resolution, translation validation.
- `internal/rnp/`
  - `api.go`: purego FFI bindings to librnp APIs.
  - `session.go`: session lifecycle, passphrase callback controls, API models.
  - `ops.go`: key management + text/file crypto/sign/verify + packet/features.
  - `session_test.go`: compatibility tests + expanded integration scenarios.
- `scripts/build-windows.ps1`
  - builds pinned deps and portable output, runs tests and smoke steps.
- `.github/workflows/smoke.yml`
  - Linux + Windows portable jobs with cache.

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
pwsh -File .\scripts\build-windows.ps1
```

The script compiles pinned deps/librnp, builds `rnparmor.exe`, creates `out\dist\rnparmor-windows-amd64`, runs targeted tests, then:

1. executes `rnparmor.exe --smoke-test`
2. runs a bounded GUI process probe

Zip output: `out\rnparmor-windows-amd64.zip`.

## CI

Workflow: `.github/workflows/smoke.yml`

- `linux` job:
  - dependency cache on `out/deps`
  - pinned librnp build + test matrix
- `windows-portable` job:
  - dependency cache on `out/deps`
  - full portable script + artifact upload

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

---

## RnpArmor 中文文档

[English](#rnparmor) | [简体中文](#rnparmor-中文文档)

RnpArmor 是基于 MyGo 原生窗口与 `librnp v0.18.1` 的 OpenPGP 图形工具，当前重点是 Windows 10/11 可解压即用的 portable 形态（同目录下 `rnparmor.exe` + `rnp.dll`），并保留 Linux 的构建与测试链路。

> 许可状态：**项目自身许可证尚未选定**。portable 包中包含第三方许可证文本。

### 功能总览

1. **MyGo 五大工作区 GUI + 中/日/英三语实时切换**
   - 工作区：
     - 钥匙圈与生成
     - 文本加解密与签名验签
     - 文件加解密与签名验签
     - Packet 检视与算法特性
     - 便携钥匙圈存储
   - 三语词典位于 `cmd/rnparmor/i18n.go`，零外部依赖。
   - 顶部栏可即时切换 `English | 中文 | 日本語`。
   - 语言可由 `--lang en|zh|ja`、`RNPARMOR_LANG`、`LANG` 预设。
   - 新增 **无菜单独立 About 页面**（About 视图不显示工作区 Tab 菜单），展示：
     - 作者：`r404r (https://github.com/r404r)`
     - 仓库地址：`https://github.com/r404r/RnpArmor`
     - RNP 上游地址：`https://github.com/rnpgp/rnp` 与当前 `librnp` 版本
     - 本地动态库路径：当前 `rnp.dll` / `librnp.so` 绝对路径
     - 密钥库地址：当前便携工作区路径与 `pubring.gpg`/`secring.gpg` 存在状态
     - 运行时栈：MyGo / purego / Botan 版本

2. **`internal/rnp` 能力扩展**
   - 补齐密钥迭代/定位、导入导出、删除、JSON 导出。
   - 基于 `rnp_generate_key_json` 的多算法密钥生成。
   - 文本能力：加密/解密、clearsign/detached/inline 签名与验签。
   - 文件能力：带选项的加解密与签名验签。
   - Packet 文本/JSON 检视与算法特性查询。
   - 验签结果扩展：状态、哈希、签名者、时间戳等。

3. **安全边界（portable 优先）**
   - 加载动态库前校验 `SHA256SUMS`。
   - Windows 端使用 `LoadLibraryEx` 约束搜索路径。
   - 仅支持显式路径钥匙圈读写，不依赖默认钥匙圈。
   - 文件发布遵循防覆盖语义。

4. **Windows portable 构建与 smoke 流程加固**
   - 新增 `rnparmor.exe --smoke-test`，用于确定性冒烟检查。
   - Windows 脚本先跑 smoke-test，再做有超时保护的 GUI 进程探针。
   - Linux/Windows CI 作业均新增 `out/deps` 缓存。

### 架构与目录

- `cmd/rnparmor/`
  - `main.go`：MyGo 主程序、5 工作区 UI、异步会话操作。
  - `i18n.go`：EN/ZH/JA 三语词典、语言解析与完整性校验。
- `internal/rnp/`
  - `api.go`：purego 对 librnp 的 FFI 绑定。
  - `session.go`：会话生命周期、口令回调控制、数据模型。
  - `ops.go`：密钥管理、文本/文件加解密签名验签、Packet/特性接口。
  - `session_test.go`：兼容测试 + 扩展测试。
- `scripts/build-windows.ps1`：Windows portable 构建、测试、冒烟与打包。
- `.github/workflows/smoke.yml`：Linux/Windows CI。

### 五大工作区说明

#### 1）钥匙圈与生成

- 用内存口令打开会话。
- 按算法参数生成密钥。
- 列表查看、公私钥导出、Key JSON、删除。
- 显式保存/加载 `pubring.gpg` 与 `secring.gpg`。

#### 2）文本工作区

- 文本加密（收件人模式 / 密码模式 / 可选加密同时签名）。
- 文本解密。
- 三种签名模式：
  - 明文签名（clearsign）
  - 分离签名（detached）
  - 内联签名（inline）
- 验签结果展示签名元数据。

#### 3）文件工作区

- 文件加密/解密。
- 分离与内联签名验签。
- 延续防覆盖发布语义。

#### 4）Packet 与特性

- OpenPGP Packet 文本或 JSON 解析。
- 全局或 `type:name` 的特性查询。

#### 5）便携钥匙圈存储

- 显式目录读写钥匙圈。
- 口令仅在会话操作期间以内存形式使用。

#### 6）无菜单 About 页面

- 从顶部 About 按钮进入。
- About 视图中按设计隐藏工作区 Tab 导航菜单。
- 页面仅提供：
  - 返回按钮（`← Back`）与语言切换器
  - 作者/仓库/RNP 上游/本地库路径/密钥库目录/运行时栈等只读信息
- 明确提示不会读取系统默认钥匙圈。

### 验签语义

界面刻意区分两类信号：

- **数学有效**：签名与内容/密钥在密码学上匹配。
- **身份可信**：签名者是否被策略信任。

RnpArmor 默认只报告前者，不会自动推断后者。

### 命令行

```bash
# 探测 librnp + 校验三语词典，不进入 GUI 循环
rnparmor --smoke-test

# 强制界面语言
rnparmor --lang en
rnparmor --lang zh
rnparmor --lang ja
```

语言预设优先级：

1. `--lang`
2. `RNPARMOR_LANG`
3. `LANG`
4. 默认 `en`

### 构建与测试

#### Linux

```bash
bash scripts/build-librnp.sh
export RNPARMOR_LIBRNP="$PWD/out/lib/librnp.so"
go test ./...
CGO_ENABLED=1 go test -race -gcflags=all=-d=checkptr=0 ./...
CGO_ENABLED=0 go build -o rnparmor ./cmd/rnparmor
```

#### Windows（portable）

```powershell
pwsh -File .\scripts\build-windows.ps1
```

脚本会构建钉扎依赖与 `rnp.dll`、打包 `rnparmor.exe`，并执行：

1. `rnparmor.exe --smoke-test`
2. 带超时保护的 GUI 进程探针

Zip 输出：`out\rnparmor-windows-amd64.zip`。

### CI

CI 文件：`.github/workflows/smoke.yml`

- `linux` 作业：
  - `out/deps` 缓存
  - pinned librnp 构建 + 测试矩阵
- `windows-portable` 作业：
  - `out/deps` 缓存
  - portable 全流程 + artifact 上传

### 安全边界与注意事项

- 动态库加载路径受控并做完整性校验。
- 不读写默认钥匙圈，必须显式目录。
- 口令回调重入被设计性阻断。
- 内存路径有输入/输出体积上限。
- 当前阶段仍建议避免在测试环境使用真实生产密钥与口令。

### 仓库状态说明

- 本仓库由最小垂直切片持续扩展而来。
- Windows 桌面人工体验、输入法与高 DPI 仍需在专门手工环境单独确认。

### 第三方依赖与钉扎

详见：

- `deps/pins.json`
- `third_party/NOTICES.md`
- portable 包 `licenses/`
