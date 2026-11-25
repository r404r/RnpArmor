# RnpArmor

[English](README.md) | 简体中文

<p><img src="assets/icons/rnparmor-256.png" alt="RnpArmor 图标" width="88" height="88" /></p>

RnpArmor 是基于 MyGo 原生窗口与 `librnp v0.18.1` 的 OpenPGP 图形工具，提供 Windows 10/11 便携版与安装版（`rnparmor.exe` + `rnp.dll`）、macOS Apple Silicon（`arm64`）`.app`/`.dmg` 应用包以及 Linux（`amd64`）便携构建与测试链路。

> 许可证：**Apache License, Version 2.0**（[`LICENSE`](LICENSE)）。第三方声明与许可文本见 [`third_party/NOTICES.md`](third_party/NOTICES.md) 及各平台分发包（Windows 与 Linux 位于 `LICENSE` 与 `licenses/`，macOS 位于 `RnpArmor.app/Contents/Resources/LICENSE` 与 `licenses/`）。

## 界面截图

Linux 运行时主窗口实机截图（`1120×780`，原生菜单栏 `文件 / 工作区 / 语言 / 帮助`，`librnp 0.18.1`），通过 [`.github/workflows/linux-screenshot.yml`](.github/workflows/linux-screenshot.yml) 与 [`scripts/capture-linux-screenshots.sh`](scripts/capture-linux-screenshots.sh) 采集：

![RnpArmor 主窗口（简体中文，Linux 运行时）](docs/screenshots/rnparmor-linux-zh.png)

<details>
<summary>英文界面截图</summary>

![RnpArmor 主窗口（英文，Linux 运行时）](docs/screenshots/rnparmor-linux-en.png)

</details>

<details>
<summary>日文界面截图</summary>

![RnpArmor 主窗口（日文，Linux 运行时）](docs/screenshots/rnparmor-linux-ja.png)

</details>

## 功能总览

1. **MyGo 五大工作区 GUI + 原生 4 组三语菜单栏 + 独立 About 子窗口**
   - 工作区：
     - 钥匙圈与生成
     - 文本加解密与签名验签
     - 文件加解密与签名验签
     - Packet 检视与算法特性
     - 便携钥匙圈存储
   - 三语词典位于 `cmd/rnparmor/i18n.go`，零外部依赖。
   - 应用图标（通过 `assets/icons/icons.go` 内嵌透明 `256×256` PNG `assets/icons/rnparmor-256.png`，即 `icons.AppIconPNG`）在运行时通过 `w.SetIcon(...)` 设置到主窗口与 About 窗口，并在 macOS 上通过 `mygo.App.Dock.SetIcon(...)` 设置 Dock 图标。
   - 主窗口挂载原生 4 组三语菜单栏（文件 / 工作区 / 语言 / 帮助）：
     - 文件：打开会话、加载钥匙圈、保存钥匙圈、退出。
     - 工作区：5 个单选菜单项，与工作区 Tab 双向联动。
     - 语言：三语单选项（`English`、`中文`、`日本語`），切换后实时刷新菜单栏、主窗口与 About 子窗口。
     - 帮助：打开独立 About 子窗口（关于）。
   - 语言可由 `--lang en|zh|ja`、`RNPARMOR_LANG`、`LANG` 预设。
   - **独立无菜单非模态 About 子窗口（`580×520`）**：
     - 单实例聚焦、固定尺寸、无菜单栏（非 macOS 平台调用 `SetMenu(nil)`）、带关闭按钮。
     - 关闭 About 窗口不影响主窗口；主窗口关闭时联动关闭 About 窗口；语言切换时窗口标题与内容同步刷新。
     - 展示应用版本（`appVersion`，如 `v0.0.1` 或 `dev`）、作者（`r404r (https://github.com/r404r)`）、仓库地址（`https://github.com/r404r/RnpArmor`）、RNP 上游地址与 `librnp` 版本、本地动态库路径（`rnp.dll` / `librnp.so`）、密钥库目录（便携工作区路径及 `pubring.gpg`/`secring.gpg` 存在状态）与 GUI 框架/运行时栈（MyGo / purego / Botan 版本）。

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

4. **Windows portable 构建、GUI 子系统与 smoke 流程加固**
   - Windows 构建使用 `-H=windowsgui`（Windows GUI 子系统），双击启动不弹出黑色控制台窗口。
   - `cmd/genwinres` 在构建期基于 `assets/icons/rnparmor.ico` 生成临时 AMD64 COFF `.syso` 资源文件，使 `rnparmor.exe` 内嵌 `.rsrc` 图标资源；`packaging/windows/rnparmor.iss` 配置 `SetupIconFile` 为安装程序设置图标。
   - `scripts/build-windows.ps1` 通过 `dumpbin /headers` 同时校验 `Windows GUI` PE 子系统与 `.rsrc` 资源节，并将子系统信息记录于 `windows-report.txt`。
   - `rnparmor.exe --smoke-test` 支持 `--smoke-out <path>` 将冒烟结果写入文件，便于 `-H=windowsgui` 产物在 CI 中进行确定性校验。
   - Windows 脚本先执行 `--smoke-test --smoke-out`，再做有超时保护的 GUI 进程探针。
   - Linux/Windows CI 作业均配置 `out/deps` 缓存。

5. **自动多平台打包发布工作流**
   - `.github/workflows/release.yml` 在 `v*` Tag 推送或手动触发时执行。
   - 同时构建 macOS Apple Silicon（`arm64`）、Linux（`amd64`）与 Windows（`amd64`）产物并附带各平台图标（macOS `RnpArmor.app` 内置 `RnpArmor.icns`、Linux 便携包附带 `rnparmor.png`、Windows 含 `.rsrc` 与 `SetupIconFile`）。
   - 三平台通过后创建/更新 GitHub Release，并上传：
     - `rnparmor-macos-arm64.dmg`（内含 ad-hoc 签名的 `RnpArmor.app`，最低 macOS 13.0）
     - `rnparmor-linux-amd64.tar.gz`
     - `rnparmor-windows-amd64-setup.exe`（Inno Setup 6 当前用户免管理员安装程序）
     - `rnparmor-windows-amd64.zip`（Windows 便携包）
     - `windows-report.txt`
     - `rnparmor-${TAG}-SHA256SUMS.txt`

## 架构与目录

- `assets/icons/`
  - `rnparmor.png`、`rnparmor-256.png`、`rnparmor.ico`、`RnpArmor.icns`：多平台透明应用图标资产。
  - `icons.go`、`icons_test.go`：内嵌 `rnparmor-256.png`（`icons.AppIconPNG`）供运行时窗口与 Dock 图标使用。
- `cmd/rnparmor/`
  - `main.go`：MyGo 主程序、运行时窗口/Dock 图标设置、原生 4 组菜单栏、5 工作区 UI、`580×520` 非模态 About 子窗口、异步会话操作、`--smoke-out` 及构建注入 `appVersion` 展示/输出。
  - `i18n.go`：EN/ZH/JA 三语词典、语言解析与完整性校验。
  - `main_test.go`：冒烟输出与文件写入、菜单模板结构及状态联动单元测试。
- `cmd/genwinres/`
  - `main.go`：零第三方依赖的纯 Go AMD64 COFF `.syso` 资源生成器，构建时将 `rnparmor.ico` 临时嵌入 `rnparmor.exe`。
- `internal/loadlib/`
  - `open.go`：绝对路径解析 `librnp` 并于加载前校验 `SHA256SUMS`，针对 macOS `.app` 包结构支持从 `Contents/MacOS/SHA256SUMS` 回退至 `Contents/Resources/SHA256SUMS`。
- `internal/rnp/`
  - `api.go`：purego 对 librnp 的 FFI 绑定。
  - `session.go`：会话生命周期、口令回调控制、数据模型。
  - `ops.go`：密钥管理、文本/文件加解密签名验签、Packet/特性接口。
  - `session_test.go`：兼容测试 + 扩展测试。
- `scripts/build-windows.ps1`：Windows 构建脚本，调用 `cmd/genwinres` 嵌入临时 `.syso` 图标、编译 `-H=windowsgui` 便携包与 Inno Setup 6 安装包（`rnparmor-windows-amd64-setup.exe`）、校验 PE 子系统、`.rsrc` 资源节与 DLL 导入表，并运行测试、`--smoke-out` 冒烟与打包。
- `scripts/build-macos.sh`：macOS Apple Silicon（`arm64`）构建脚本，编译静态依赖（`Botan 3.10.0` + `json-c 0.19`）与 `librnp.dylib`，组装 `RnpArmor.app`（写入 `Contents/Resources/RnpArmor.icns` 与 `Info.plist` 的 `CFBundleIconFile`），执行 ad-hoc `codesign --verify --deep --strict`、`go test`、`--smoke-test`，最终打包 `rnparmor-macos-arm64.dmg`。
- `scripts/capture-linux-screenshots.sh`：在无头 X11（`Xvfb` + `openbox`）下以 `--lang en|zh|ja` 启动 `rnparmor`，通过 `scrot` 抓取聚焦窗口截图并校验分辨率与非空白像素方差。
- `packaging/windows/rnparmor.iss`：Windows 安装程序（`rnparmor-windows-amd64-setup.exe`）的 Inno Setup 6 脚本（含 `SetupIconFile`）。
- `.github/workflows/smoke.yml`：`linux`、`windows-portable` 与 `macos-arm64` 三平台 CI（含缓存与同分支并发自动取消）。
- `.github/workflows/release.yml`：Tag/手动触发的多平台自动打包发布。
- `.github/workflows/linux-screenshot.yml`：调用 `scripts/capture-linux-screenshots.sh` 采集 Linux 三语实机截图（含 `assets/icons/**` 变更触发），上传 `rnparmor-linux-screenshots` artifact，并可在非 `main` 分支将更新后的 PNG 提交至 `docs/screenshots/`。

## 五大工作区说明

### 1）钥匙圈与生成

- 用内存口令打开会话。
- 按算法参数生成密钥。
- 列表查看、公私钥导出、Key JSON、删除。
- 显式保存/加载 `pubring.gpg` 与 `secring.gpg`。

### 2）文本工作区

- 文本加密（收件人模式 / 密码模式 / 可选加密同时签名）。
- 文本解密。
- 三种签名模式：
  - 明文签名（clearsign）
  - 分离签名（detached）
  - 内联签名（inline）
- 验签结果展示签名元数据。

### 3）文件工作区

- 文件加密/解密。
- 分离与内联签名验签。
- 延续防覆盖发布语义。

### 4）Packet 与特性

- OpenPGP Packet 文本或 JSON 解析。
- 全局或 `type:name` 的特性查询。

### 5）便携钥匙圈存储

- 显式目录读写钥匙圈。
- 口令仅在会话操作期间以内存形式使用。

### 6）原生 4 组菜单栏与独立 About 子窗口

- 主窗口顶部原生菜单栏包含 4 组：文件（打开会话 / 加载与保存钥匙圈 / 退出）、工作区（5 个工作区单选联动）、语言（`English` / `中文` / `日本語` 单选实时刷新菜单与窗口）与帮助（关于）。
- 点击「帮助 → 关于」打开 `580×520` 固定尺寸、非模态的独立无菜单子窗口（单实例聚焦、无菜单栏、带关闭按钮）。
- 关闭 About 子窗口不影响主窗口；主窗口关闭时联动关闭 About 子窗口；切换语言时其标题与内容同步更新。
- 仅展示只读信息（应用版本、作者、仓库、RNP 上游、本地动态库路径、便携密钥库目录及 `pubring.gpg`/`secring.gpg` 存在状态、运行时栈），并明确提示不读取系统默认钥匙圈。

## 验签语义

界面刻意区分两类信号：

- **数学有效**：签名与内容/密钥在密码学上匹配。
- **身份可信**：签名者是否被策略信任。

RnpArmor 默认只报告前者，不会自动推断后者。

## 命令行

```bash
# 探测 librnp + 校验三语词典，不进入 GUI 循环
rnparmor --smoke-test

# 将冒烟输出写入指定文件（用于 Windows -H=windowsgui 构建的 CI 确定性校验）
rnparmor --smoke-test --smoke-out out/smoke-out.txt

# 强制界面语言
rnparmor --lang en
rnparmor --lang zh
rnparmor --lang ja
```

预期 smoke 输出含：

```text
status=ok
app_version=dev
library=...
version=...
i18n=ok
```

语言预设优先级：

1. `--lang`
2. `RNPARMOR_LANG`
3. `LANG`
4. 默认 `en`

## 构建与测试

### Linux

```bash
bash scripts/build-librnp.sh
export RNPARMOR_LIBRNP="$PWD/out/lib/librnp.so"
go test ./...
CGO_ENABLED=1 go test -race -gcflags=all=-d=checkptr=0 ./...
CGO_ENABLED=0 go build -o rnparmor ./cmd/rnparmor
```

### macOS（Apple Silicon / arm64）

```bash
# 一键执行：按序执行 deps -> rnp -> package（默认 --stage all）
bash scripts/build-macos.sh

# 分阶段模式（便于本地迭代与 CI 缓存）
bash scripts/build-macos.sh --stage deps
bash scripts/build-macos.sh --stage rnp
bash scripts/build-macos.sh --stage package

# 注入版本号到 Info.plist 与二进制元数据
bash scripts/build-macos.sh --stage package --version v0.0.1
```

`scripts/build-macos.sh` 面向 Apple Silicon（`arm64`），最低系统要求 `MACOSX_DEPLOYMENT_TARGET=13.0`（`LSMinimumSystemVersion=13.0`）：

- `deps`：按 `scripts/windows/botan3-modules.txt` 的 57 个模块清单编译静态 `Botan 3.10.0`（写入 `.modules-sha256` 校验印记）与静态 `json-c 0.19` 到 `out/deps`。
- `rnp`：构建 `librnp.dylib`（install name 为 `@loader_path/librnp.dylib`），静态链接 Botan 与 `json-c`，动态链接 macOS 系统自带 `/usr/lib/libz` 与 `/usr/lib/libbz2` 以及 `-framework Security`（通过 `otool -L` 校验无外部 Homebrew 等非系统路径依赖）。
- `package`：构建 `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64` 的 `rnparmor`，组装 `out/dist/RnpArmor.app`（`Contents/MacOS/rnparmor`、`Contents/MacOS/librnp.dylib`、`Contents/Resources/RnpArmor.icns`、`Contents/Resources/LICENSE`、`Contents/Resources/SHA256SUMS` 与 `Contents/Resources/licenses/`，并在 `Info.plist` 写入 `CFBundleIconFile`），在严格签名校验前检查图标完备性，执行 ad-hoc 签名与 `codesign --verify --deep --strict`，运行 `go test` 与 `--smoke-test`，最终通过 `hdiutil create` 生成 `out/rnparmor-macos-arm64.dmg`。

> macOS Gatekeeper 提示：`RnpArmor.app` 仅作 ad-hoc 签名且**未经过 Apple 公证**。首次在访达中打开可能触发 Gatekeeper 安全提示（如遇提示，可通过右键 → 打开或在「系统设置 → 隐私与安全性」中放行）。

### Windows（portable 与安装版）

```powershell
# 一键（兼容旧习惯）：按序执行 deps -> rnp -> package（默认 -Stage all）
pwsh -File .\scripts\build-windows.ps1

# 粗粒度分阶段模式
pwsh -File .\scripts\build-windows.ps1 -Stage deps
pwsh -File .\scripts\build-windows.ps1 -Stage rnp
pwsh -File .\scripts\build-windows.ps1 -Stage package

# 细粒度依赖阶段（CI 用于分段观测耗时与缓存复用）
pwsh -File .\scripts\build-windows.ps1 -Stage fetch-deps
pwsh -File .\scripts\build-windows.ps1 -Stage botan
pwsh -File .\scripts\build-windows.ps1 -Stage small-deps

# 注入应用版本（用于 --smoke-test / About 显示）
pwsh -File .\scripts\build-windows.ps1 -Version v0.0.1

# 分阶段 + 版本注入
pwsh -File .\scripts\build-windows.ps1 -Stage package -Version v0.0.1
```

Windows 依赖构建钉扎并打包以下组件：

- `Botan 3.10.0`：按 `scripts/windows/botan3-modules.txt` 中的精简静态模块清单（共 57 个模块，每行一个模块，支持注释）编译静态库，对齐 `rnpgp/rnp v0.18.1` 的 OpenPGP 能力需求，避免在 CI 中编译不必要的 CLI/测试与非 OpenPGP 模块。
- `json-c 0.19`、`zlib 1.4`、`bzip2 1.0.8`：编译为静态库并链接进 `rnp.dll`。
- `dirent 1.24`（`tronkko/dirent`，MIT 许可）：Windows 构建 `librnp` 所需的头文件兼容依赖（`dirent.h`）。
- 上述及 Go/RNP 第三方许可证文本（`rnp-LICENSE.md`、`sexpp-LICENSE.md`、`botan-license.txt`、`dirent-LICENSE`、`json-c-COPYING`、`zlib-LICENSE`、`bzip2-LICENSE`、`mygo-MIT.txt`、`purego-Apache-2.0.txt`）均随 Windows portable 包打入 `licenses/` 目录。

`scripts/build-windows.ps1` 支持以下 `-Stage` 阶段参数：

1. `fetch-deps`：下载、校验并解压 `Botan 3.10.0`、`json-c 0.19`、`zlib 1.4`、`bzip2 1.0.8` 与 `dirent 1.24` 源码到 `out/src`（若 `out/deps` 热缓存已完整则自动跳过）
2. `botan`：编译并安装精简静态 `Botan 3.10.0` 到 `out/deps/botan`（写入 `.modules-sha256` 校验印记）
3. `small-deps`：编译并安装静态 `json-c`、`zlib`、`bzip2` 到 `out/deps`，准备 `dirent.h` 并将依赖许可证缓存至 `out/deps/licenses`
4. `deps`：依次执行 `fetch-deps -> botan -> small-deps`
5. `rnp`：基于缓存依赖编译 `librnp`（`rnp.dll`），并缓存至 `out/win-build/rnp-dll/rnp.dll`
6. `package`：调用 `cmd/genwinres` 基于 `assets/icons/rnparmor.ico` 生成临时 `cmd/rnparmor/rsrc_windows_amd64.syso`，使用 `-H=windowsgui` 构建 `rnparmor.exe` 并清理临时 `.syso`，复制 `rnp.dll`、`LICENSE` 与 `licenses/`，校验 `Windows GUI` PE 子系统、`.rsrc` 资源节（`dumpbin /headers`）与 DLL 导入表（`DEPENDENCIES.txt`），生成 `PORTABLE.txt` 与 `SHA256SUMS`，对打包后的 `rnp.dll` 执行 `go test`，运行 `--smoke-test --smoke-out` 与 GUI 探针，并产出便携 zip、Inno Setup 6 安装包（`SetupIconFile=..\..\assets\icons\rnparmor.ico`）与报告
7. `all`：依次执行 `deps -> rnp -> package`（默认）

`package` 阶段会生成 `out\dist\rnparmor-windows-amd64`，使用 `packaging/windows/rnparmor.iss` 与 Inno Setup 6（`ISCC.exe`）打包安装程序，并执行：

1. `dumpbin /headers` 校验，确认 `rnparmor.exe` 为 `Windows GUI` 子系统（记录至 `out\windows-report.txt`）且包含 `.rsrc` 图标资源节
2. 将 `RNPARMOR_LIBRNP` 指向打包目录内的 `rnp.dll`、并将 `PATH` 收窄到 `System32` 后运行 `go test -count=1 -timeout 20m ./internal/loadlib/ ./internal/rnp/ ./cmd/rnparmor/`
3. 执行 `rnparmor.exe --smoke-test --smoke-out <path>`（核对 `status=ok`、`app_version=<Version>` 与 `i18n=ok`）
4. 执行带超时保护的 GUI 进程探针

输出产物：`out\rnparmor-windows-amd64.zip`、`out\rnparmor-windows-amd64-setup.exe`（默认当前用户免管理员安装 `PrivilegesRequired=lowest`，支持开始菜单快捷方式、可选桌面快捷方式与标准卸载）与 `out\windows-report.txt`。

## CI

工作流文件：`.github/workflows/smoke.yml`

- `linux` 作业：
  - `out/deps` 缓存
  - pinned `librnp` 构建 + `go test -race` + `CGO_ENABLED=0` 构建
- `windows-portable` 作业：
  - 使用 `actions/cache/restore@v4` 恢复 `out/deps`，并在 `small-deps` 之后执行 `actions/cache/save@v4` 保存缓存
  - CI 拆分为 `fetch-deps -> botan -> small-deps -> save cache -> rnp -> package`，以提升各阶段耗时观测性与依赖缓存复用率
  - 缓存键：`${{ runner.os }}-deps-v4-${{ hashFiles('deps/pins.json', 'scripts/windows/**') }}`
  - 上传 `out/rnparmor-windows-amd64.zip`、`out/rnparmor-windows-amd64-setup.exe` 与 `out/windows-report.txt`
- `macos-arm64` 作业（`runs-on: macos-15`）：
  - 使用 `out/deps` 缓存，缓存键 `${{ runner.os }}-arm64-deps-v1-${{ hashFiles('deps/pins.json', 'scripts/windows/botan3-modules.txt', 'scripts/build-macos.sh') }}`
  - 分阶段流程：`deps -> save cache -> rnp -> package`
  - 上传 `out/rnparmor-macos-arm64.dmg`

发布文件：`.github/workflows/release.yml`

- 触发条件：
  - 推送 `v*` Tag
  - 手动触发（可选 `tag` 输入）
- 作业：
  - `linux-portable`：构建/测试/打包 `rnparmor-linux-amd64.tar.gz`（随包附带 `assets/icons/rnparmor.png` 即 `rnparmor.png`）
  - `windows-portable`：沿用相同的 `deps-v4` 缓存键与分段流水线（`fetch-deps -> botan -> small-deps -> save cache -> rnp -> package -Version "${{ steps.tag.outputs.value }}"`），产出 `rnparmor-windows-amd64.zip`、`rnparmor-windows-amd64-setup.exe` 与 `windows-report.txt`
  - `macos-arm64`（`runs-on: macos-15`）：沿用相同的 `arm64-deps-v1` 缓存键与分段流水线（`deps -> save cache -> rnp -> package --version "${{ steps.tag.outputs.value }}"`），产出 `rnparmor-macos-arm64.dmg`
  - `publish-release`：三平台均通过后创建/更新 GitHub Release，上传 `rnparmor-macos-arm64.dmg`、`rnparmor-linux-amd64.tar.gz`、`rnparmor-windows-amd64-setup.exe`、`rnparmor-windows-amd64.zip`、`windows-report.txt` 与统一校验文件 `rnparmor-${TAG}-SHA256SUMS.txt`

截图工作流：`.github/workflows/linux-screenshot.yml`

- 触发条件：
  - `workflow_dispatch`（可选 `commit_screenshots` 布尔输入，仅限非 `main` 分支）
  - 修改 `scripts/capture-linux-screenshots.sh`、`.github/workflows/linux-screenshot.yml` 或 `assets/icons/**` 的 `pull_request`
- `linux-screenshot` 作业（`runs-on: ubuntu-latest`）：
  - 构建 pinned `librnp.so` 与 `rnparmor`，在 `Xvfb` + `openbox` 下运行 `scripts/capture-linux-screenshots.sh`，上传 `rnparmor-linux-screenshots`（`out/screenshots/*.png`），并按条件在非 `main` 分支提交 `docs/screenshots/rnparmor-linux-{en,zh,ja}.png`

## 安全边界与注意事项

- 动态库加载路径受控并做完整性校验。
- 不读写默认钥匙圈，必须显式目录。
- 口令回调重入被设计性阻断。
- 内存路径有输入/输出体积上限。
- 当前阶段仍建议避免在测试环境使用真实生产密钥与口令。

## 仓库状态说明

- `v0.0.1` 多平台版本已发布（覆盖 macOS `arm64`、Linux `amd64` 与 Windows `amd64`）；项目现已正式采用 Apache License, Version 2.0（[`LICENSE`](LICENSE)），后续发布版本将随包分发该许可证。
- 桌面端人工交互体验（Windows 10/11 中文输入法、高 DPI 与安装程序流程，以及 macOS Apple Silicon 首次启动未公证 Gatekeeper 提示）仍依赖真实桌面环境，需在专门手工测试中确认。

## 许可证

- RnpArmor 本身采用 **Apache License, Version 2.0**（`Copyright 2026 r404r`），全文见 [`LICENSE`](LICENSE)。
- 各平台分发包内许可证路径：
  - Windows（`rnparmor-windows-amd64.zip`、`rnparmor-windows-amd64-setup.exe`）与 Linux（`rnparmor-linux-amd64.tar.gz`）：包根目录下的 `LICENSE` 与 `licenses/`。
  - macOS（`rnparmor-macos-arm64.dmg` / `RnpArmor.app`）：`RnpArmor.app/Contents/Resources/LICENSE` 与 `RnpArmor.app/Contents/Resources/licenses/`。

## 第三方依赖与钉扎

详见：

- `deps/pins.json`
- `third_party/NOTICES.md`
- 分发包内 `LICENSE` 与 `licenses/`
