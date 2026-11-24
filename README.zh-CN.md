# RnpArmor

[English](README.md) | 简体中文

RnpArmor 是基于 MyGo 原生窗口与 `librnp v0.18.1` 的 OpenPGP 图形工具，当前重点是 Windows 10/11 可解压即用的 portable 形态（同目录下 `rnparmor.exe` + `rnp.dll`），并保留 Linux 的构建与测试链路。

> 许可状态：**项目自身许可证尚未选定**。portable 包中包含第三方许可证文本。

## 功能总览

1. **MyGo 五大工作区 GUI + 原生 4 组三语菜单栏 + 独立 About 子窗口**
   - 工作区：
     - 钥匙圈与生成
     - 文本加解密与签名验签
     - 文件加解密与签名验签
     - Packet 检视与算法特性
     - 便携钥匙圈存储
   - 三语词典位于 `cmd/rnparmor/i18n.go`，零外部依赖。
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
   - `scripts/build-windows.ps1` 通过 `dumpbin /headers` 校验 PE 子系统为 `Windows GUI` 并记录于 `windows-report.txt`。
   - `rnparmor.exe --smoke-test` 支持 `--smoke-out <path>` 将冒烟结果写入文件，便于 `-H=windowsgui` 产物在 CI 中进行确定性校验。
   - Windows 脚本先执行 `--smoke-test --smoke-out`，再做有超时保护的 GUI 进程探针。
   - Linux/Windows CI 作业均配置 `out/deps` 缓存。

5. **自动打包发布工作流**
   - `.github/workflows/release.yml` 在 `v*` Tag 推送或手动触发时执行。
   - 同时构建 Linux/Windows portable 包并上传 artifact。
   - 双平台成功后创建/更新 GitHub Release，并上传：
     - `rnparmor-windows-amd64.zip`
     - `rnparmor-linux-amd64.tar.gz`
     - `windows-report.txt`
     - `rnparmor-${TAG}-SHA256SUMS.txt`

## 架构与目录

- `cmd/rnparmor/`
  - `main.go`：MyGo 主程序、原生 4 组菜单栏、5 工作区 UI、`580×520` 非模态 About 子窗口、异步会话操作、`--smoke-out` 及构建注入 `appVersion` 展示/输出。
  - `i18n.go`：EN/ZH/JA 三语词典、语言解析与完整性校验。
  - `main_test.go`：冒烟输出与文件写入、菜单模板结构及状态联动单元测试。
- `internal/rnp/`
  - `api.go`：purego 对 librnp 的 FFI 绑定。
  - `session.go`：会话生命周期、口令回调控制、数据模型。
  - `ops.go`：密钥管理、文本/文件加解密签名验签、Packet/特性接口。
  - `session_test.go`：兼容测试 + 扩展测试。
- `scripts/build-windows.ps1`：Windows `-H=windowsgui` portable 构建、PE 子系统与 DLL 导入校验、测试、`--smoke-out` 冒烟与打包。
- `.github/workflows/smoke.yml`：Linux/Windows CI（含同分支并发自动取消）。
- `.github/workflows/release.yml`：Tag/手动触发的自动打包发布。

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

### Windows（portable）

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
6. `package`：使用 `-H=windowsgui` 构建 `rnparmor.exe`、复制 `rnp.dll` 与 `licenses/`、校验 `Windows GUI` PE 子系统（`dumpbin /headers`）与 DLL 导入表（`DEPENDENCIES.txt`）、生成 `PORTABLE.txt` 与 `SHA256SUMS`、对打包后的 `rnp.dll` 执行 `go test`、运行 `--smoke-test --smoke-out` 与 GUI 探针并产出 zip 与报告
7. `all`：依次执行 `deps -> rnp -> package`（默认）

`package` 阶段会生成 `out\dist\rnparmor-windows-amd64`，并执行：

1. `dumpbin /headers` 校验，确认 `rnparmor.exe` 为 `Windows GUI` 子系统（记录至 `out\windows-report.txt`）
2. 将 `RNPARMOR_LIBRNP` 指向打包目录内的 `rnp.dll`、并将 `PATH` 收窄到 `System32` 后运行 `go test -count=1 -timeout 20m ./internal/loadlib/ ./internal/rnp/ ./cmd/rnparmor/`
3. 执行 `rnparmor.exe --smoke-test --smoke-out <path>`（核对 `status=ok`、`app_version=<Version>` 与 `i18n=ok`）
4. 执行带超时保护的 GUI 进程探针

Zip 输出：`out\rnparmor-windows-amd64.zip`（报告：`out\windows-report.txt`）。

## CI

工作流文件：`.github/workflows/smoke.yml`

- `linux` 作业：
  - `out/deps` 缓存
  - pinned `librnp` 构建 + `go test -race` + `CGO_ENABLED=0` 构建
- `windows-portable` 作业：
  - 使用 `actions/cache/restore@v4` 恢复 `out/deps`，并在 `small-deps` 之后执行 `actions/cache/save@v4` 保存缓存
  - CI 拆分为 `fetch-deps -> botan -> small-deps -> save cache -> rnp -> package`，以提升各阶段耗时观测性与依赖缓存复用率
  - 缓存键：`${{ runner.os }}-deps-v4-${{ hashFiles('deps/pins.json', 'scripts/windows/**') }}`

发布文件：`.github/workflows/release.yml`

- 触发条件：
  - 推送 `v*` Tag
  - 手动触发（可选 `tag` 输入）
- 作业：
  - `linux-portable`：构建/测试/打包 `rnparmor-linux-amd64.tar.gz`
  - `windows-portable`：沿用相同的 `deps-v4` 缓存键与分段流水线（`fetch-deps -> botan -> small-deps -> save cache -> rnp -> package -Version "${{ steps.tag.outputs.value }}"`），产出 `rnparmor-windows-amd64.zip` 与 `windows-report.txt`
  - `publish-release`：双平台通过后创建/更新 Release 并上传资产与统一校验文件（`rnparmor-${TAG}-SHA256SUMS.txt`）

## 安全边界与注意事项

- 动态库加载路径受控并做完整性校验。
- 不读写默认钥匙圈，必须显式目录。
- 口令回调重入被设计性阻断。
- 内存路径有输入/输出体积上限。
- 当前阶段仍建议避免在测试环境使用真实生产密钥与口令。

## 仓库状态说明

- `feat/win-gui-menu-and-about` 分支（PR #4）已通过 `.github/workflows/smoke.yml` 双平台 CI，完成 `-H=windowsgui` PE 子系统校验、主窗口原生 4 组三语菜单栏、`580×520` 独立非模态 About 子窗口及 `--smoke-out` 确定性冒烟校验。
- Windows 桌面人工交互体验、中文输入法与高 DPI 表现仍依赖真实环境，需在 Windows 10/11 桌面环境中单独确认。

## 第三方依赖与钉扎

详见：

- `deps/pins.json`
- `third_party/NOTICES.md`
- portable 包 `licenses/`
