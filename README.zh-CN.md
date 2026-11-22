# RnpArmor

[English](README.md) | 简体中文

RnpArmor 是基于 MyGo 原生窗口与 `librnp v0.18.1` 的 OpenPGP 图形工具，当前重点是 Windows 10/11 可解压即用的 portable 形态（同目录下 `rnparmor.exe` + `rnp.dll`），并保留 Linux 的构建与测试链路。

> 许可状态：**项目自身许可证尚未选定**。portable 包中包含第三方许可证文本。

## 功能总览

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
     - 应用版本：构建期注入的应用版本号（`appVersion`，例如 `v0.0.1` 或 `dev`）
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
   - `--smoke-test` 现会输出 `app_version=<值>`，CI 会核对版本注入结果。
   - Windows 脚本先跑 smoke-test，再做有超时保护的 GUI 进程探针。
   - Linux/Windows CI 作业均新增 `out/deps` 缓存。

5. **自动打包发布工作流**
   - 新增 `.github/workflows/release.yml`，在 `v*` Tag 推送或手动触发时执行。
   - 同时构建 Linux/Windows portable 包并上传 artifact。
   - 双平台成功后创建/更新 GitHub Release，并上传：
     - `rnparmor-windows-amd64.zip`
     - `rnparmor-linux-amd64.tar.gz`
     - `windows-report.txt`
     - `rnparmor-${TAG}-SHA256SUMS.txt`

## 架构与目录

- `cmd/rnparmor/`
  - `main.go`：MyGo 主程序、5 工作区 UI、异步会话操作，以及构建注入 `appVersion` 展示/输出。
  - `i18n.go`：EN/ZH/JA 三语词典、语言解析与完整性校验。
- `internal/rnp/`
  - `api.go`：purego 对 librnp 的 FFI 绑定。
  - `session.go`：会话生命周期、口令回调控制、数据模型。
  - `ops.go`：密钥管理、文本/文件加解密签名验签、Packet/特性接口。
  - `session_test.go`：兼容测试 + 扩展测试。
- `scripts/build-windows.ps1`：Windows portable 构建、测试、冒烟与打包。
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

### 6）无菜单 About 页面

- 从顶部 About 按钮进入。
- About 视图中按设计隐藏工作区 Tab 导航菜单。
- 页面仅提供：
  - 返回按钮（`← Back`）与语言切换器
  - 作者/仓库/RNP 上游/本地库路径/密钥库目录/运行时栈等只读信息
- 明确提示不会读取系统默认钥匙圈。

## 验签语义

界面刻意区分两类信号：

- **数学有效**：签名与内容/密钥在密码学上匹配。
- **身份可信**：签名者是否被策略信任。

RnpArmor 默认只报告前者，不会自动推断后者。

## 命令行

```bash
# 探测 librnp + 校验三语词典，不进入 GUI 循环
rnparmor --smoke-test

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
# 一键（兼容旧习惯）：deps + rnp + package
pwsh -File .\scripts\build-windows.ps1

# 分阶段（便于 CI 观测耗时与缓存）
pwsh -File .\scripts\build-windows.ps1 -Stage deps
pwsh -File .\scripts\build-windows.ps1 -Stage rnp
pwsh -File .\scripts\build-windows.ps1 -Stage package

# 注入应用版本（用于 --smoke-test / About 显示）
pwsh -File .\scripts\build-windows.ps1 -Version v1.2.3

# 分阶段 + 版本注入
pwsh -File .\scripts\build-windows.ps1 -Stage package -Version v1.2.3
```

Windows 依赖构建采用 `scripts/windows/botan3-modules.txt` 中的 Botan
最小静态模块清单（每行一个模块，支持注释），对齐 `rnpgp/rnp v0.18.1`
所需能力，避免在 CI 中编译不必要的 CLI/测试与非 OpenPGP 模块。

`scripts/build-windows.ps1` 现支持分阶段执行：

1. `deps`：构建/缓存 Botan + json-c + zlib + bzip2 到 `out/deps`（含许可证缓存）
2. `rnp`：构建 `librnp`，并将 `rnp.dll` 缓存到 `out/win-build/rnp-dll/rnp.dll`
3. `package`：构建 `rnparmor.exe`、执行测试/冒烟/GUI 探针并产出 zip/report

`package` 阶段会生成 `out\dist\rnparmor-windows-amd64`，并执行：

1. `rnparmor.exe --smoke-test`
2. 带超时保护的 GUI 进程探针

Zip 输出：`out\rnparmor-windows-amd64.zip`。

## CI

CI 文件：`.github/workflows/smoke.yml`

- `linux` 作业：
  - `out/deps` 缓存
  - pinned librnp 构建 + 测试矩阵
- `windows-portable` 作业：
  - `out/deps` 缓存（`actions/cache/restore` + deps 阶段后 `actions/cache/save`）
  - Windows 分阶段流水线（`deps` -> `rnp` -> `package`），可直接观测各阶段耗时
  - 缓存键：`${{ runner.os }}-deps-v3-${{ hashFiles('deps/pins.json', 'scripts/windows/**') }}`

发布文件：`.github/workflows/release.yml`

- 触发条件：
  - 推送 `v*` Tag
  - 手动触发（可选 `tag` 输入）
- 作业：
  - `linux-portable`：构建/测试/打包 `rnparmor-linux-amd64.tar.gz`
  - `windows-portable`：构建/测试/打包 `rnparmor-windows-amd64.zip` 与 `windows-report.txt`
  - `publish-release`：双平台通过后创建/更新 Release 并上传资产与统一 SHA256

## 安全边界与注意事项

- 动态库加载路径受控并做完整性校验。
- 不读写默认钥匙圈，必须显式目录。
- 口令回调重入被设计性阻断。
- 内存路径有输入/输出体积上限。
- 当前阶段仍建议避免在测试环境使用真实生产密钥与口令。

## 仓库状态说明

- 本仓库由最小垂直切片持续扩展而来。
- Windows 桌面人工体验、输入法与高 DPI 仍需在专门手工环境单独确认。

## 第三方依赖与钉扎

详见：

- `deps/pins.json`
- `third_party/NOTICES.md`
- portable 包 `licenses/`
