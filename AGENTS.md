# RnpArmor - Project Agent Rules & Constraints

## 1. 仓库与分支红线
- **目标仓库**：`https://github.com/r404r/RnpArmor`
- **工作目录**：`/data/OpenBot/Shared/RnpArmor`（多分支并行开发必须使用 `/data/OpenBot/Shared/RnpArmor/worktrees/<branch>` 下的 `git worktree`）。
- **分支红线**：严禁直接向受保护的 `main` 分支提交未审查代码；所有开发必须在独立功能分支（`feat/*` / `fix/*` `worktree` 分支）上进行。
- **跨项目隔离**：严禁读写其他项目目录（如 `/data/OpenBot/Shared/p-w-C-UI-MOCK-002`）。

## 1.1 提交、推送与 PR 合并门禁规范
- **原子提交与即时推送**：每个小功能（Feature/修复）须为可编译、逻辑自洽的最小闭环（代码 + 对应测试 + 双语 `README.md` 同步检查）；涉及 UI 须先完成独立 QA 实测与截图验收，随后立即本地 Commit 并推送至远端功能分支。
- **PR 生命周期与 CI 并发节流**：新分支首次 Push 即创建 PR（多步任务先建 Draft PR）持续触发 CI 验证；`.github/workflows/smoke.yml` 配置 `concurrency`（`cancel-in-progress: true`）自动取消同分支旧构建。
- **硬性合并门禁**：Merge PR 前必须确认当前 `HEAD` 的 GitHub Actions 检查项全部通过（`success`）且 UI 已通过独立 QA；若 CI 失败，先在原分支修复并重推验证。
- **单分支单周期**：PR 合入 `main` 后即封存关闭该功能分支，严禁向已合并旧分支追加新功能；后续需求必须从最新 `origin/main` 切出新 `feat/*` / `fix/*` 分支与 `worktree`。
- **单一推送出口**：专职子 Agent 在各自 `worktree` 完成开发与本地 Commit；由 Master 总管核实交付物、`README.md` 双语同步及 QA 结果后统一 Push 远端与管理 PR。

## 2. 本地端口与环境清理规范
- **专属端口段**：`8200–8209`（仅用于 RnpArmor 的 UI 原型预览或本地测试服务）。
- **清理要求**：测试或预览结束后必须立即停止本地服务进程，并清理仓库外或临时产物，保持 `git status` 干净。

## 3. 技术栈与打包目标
- **GUI 框架**：基于 `github.com/egoist/mygo` (`v0.10.0`) 构建原生图形界面。
- **底层密码学库**：基于 `rnpgp/rnp` (`librnp v0.18.1`) + `Botan 3.10.0` + `json-c 0.19`，通过 `purego` (`CGO_ENABLED=0`) 动态加载同目录绝对路径并校验 `SHA256SUMS`。
- **Windows Portable 目标**：构建 Windows x64 单目录解压即用（portable）版本（`rnparmor.exe` + 静态链接依赖的 `rnp.dll` + `SHA256SUMS` + `licenses/`），不依赖额外 VC 运行库或外部 PATH。

## 4. 隐私与安全约束
- **口令与私钥保护**：严禁将任何真实口令、测试口令或私钥写入日志、源码或 Git 提交；内存中的敏感字节使用后必须显式清零（`wipe`）。
- **钥匙圈隔离**：不擅自隐式读写系统默认 GPG 钥匙圈，所有密钥库导入/导出/保存均须通过显式路径或用户指定目录执行，且默认防覆盖。
- **线程安全**：所有 `librnp` 调用必须通过 `Session.Do` / `Session.Go` 在锁定的工作线程上串行执行，严禁阻塞 MyGo UI 线程或在口令回调中重入会话。

## 5. 多语言（i18n）与 README 同步铁律
- **三语界面切换**：GUI 界面（MyGo 桌面端及 HTML 交互原型）必须内置支持 **English / 简体中文 / 日本語** 三种语言，并提供显式的语言切换器（Language Switcher），支持在多语言间即时无损切换。
- **README 分文件规范与必检机制**：仓库文档必须拆分为**英文 `README.md`** 与**简体中文 `README.zh-CN.md`** 两个独立文件，严禁中英文混写在同一文件；两份文件在最开头（标题下）必须提供互链语言切换（`English | [简体中文](README.zh-CN.md)` / `[English](README.md) | 简体中文`）。**每个 Feature 追加完成后，以及每次发布新版本（打 Tag / 发布 Release）前，必须同步检查并更新两份文档**，确保新增特性、使用方法、版本变更与安全边界在英/中文档中准确一致。
