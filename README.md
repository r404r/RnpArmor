# RnpArmor

MyGo 原生窗口加上 librnp 的最小试验。目标是 Windows 10/11 上解压即可用的单目录程序，macOS 和 Linux 以后再补。当前没有完整的文件管理界面，也没有选定本项目的开源许可证。

仓库最初只有标题。没有合并，没有 GitHub Release。

## 现在能用的

- 一个 MyGo 窗口。它只显示 librnp 是否加载，以及库报告的版本字符串。不在窗口里生成密钥或加密文件。
- 库路径必须是绝对路径，或与 `rnparmor.exe` 放在同一目录的 `rnp.dll`（Windows）/ `librnp.so`（Linux）。加载前核对同目录 `SHA256SUMS`。Windows 使用 `LoadLibraryEx`，不搜索当前目录和 `PATH`。
- 测试代码可以：用随机口令生成临时 EDDSA 与 Curve25519 密钥；加密、解密、分离签名、验签；拒绝错误口令、取消、篡改和覆盖已有文件。这些操作不在窗口里。
- 签名验签成功只表示签名和本会话里的钥匙在数学上对得上。程序不会把这说成身份可信。

## 还没有的

- 密钥列表、导入导出、进度条，以及在窗口里选择文件。
- 从进行中的 librnp 调用里取消。取消只在调用返回后丢弃尚未发布的输出。
- 默认钥匙圈、`%APPDATA%`、智能卡、钥匙服务器和信任模型。测试阶段不要放真实钥匙或口令。
- 中文输入法和高 DPI。没有测过。
- 干净的 Windows 10 或 Windows 11 桌面上由人打开窗口的确认。GitHub 的 Windows runner 不是这个环境。

## 下载 Windows 包

Windows x64 的单目录 zip 由本仓库的 GitHub Actions 作业 `windows-portable` 上传为 artifact，名称是 `rnparmor-windows-amd64`。这不是 GitHub Release。artifact 保留 30 天，下载需要能打开该 Actions 页面的 GitHub 账号。

准确链接和 zip 的 SHA-256 在作业完成后写在这里。作业页：<https://github.com/r404r/RnpArmor/actions>

解压后进入 `rnparmor-windows-amd64`，直接运行 `rnparmor.exe`。不需要安装 Go、编译器或额外的 VC 运行库。`rnp.dll` 必须留在 exe 旁边。zlib 和 bzip2 编进 `rnp.dll`，不另附 DLL。

## 目录

```
rnparmor-windows-amd64/
  rnparmor.exe
  rnp.dll
  SHA256SUMS
  DEPENDENCIES.txt
  PORTABLE.txt
  licenses/
```

`DEPENDENCIES.txt` 是打包时 `dumpbin /dependents` 的原文。系统 DLL（含 UCRT，如果导入表里有）不放进 zip，它们来自 Windows 10 及以上的系统目录。若导入表出现 `vcruntime*.dll` 或 `msvcp*.dll`，构建会失败，而不是从系统目录抄一份进来。

## 依赖和系统要求

| 组件 | 钉扎 |
| --- | --- |
| librnp `v0.18.1` | `aad1892e4d8423398a0b973bd0a7e4544c359afc`。不要用已撤回的 `v0.18.0` |
| sexpp | `c641a2f36520bab783657a58650d9fda548b9dec` |
| Botan 3.10.0 | SHA-256 `fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24` |
| json-c 0.19 | SHA-256 `37ad0249902e301bd9052bf712e511fcc6acff4ecaad4b5900aad9ce564e26de`，静态链接 |
| zlib 1.4 | SHA-256 `02dd159783677a749c04adbdd02b55b8a2bcc7e054f51b76b549199a916ad584`，静态链接 |
| bzip2 1.0.8 | SHA-512 `083f5e675d73f3233c7930ebe20425a533feedeaaa9d8cc86831312a6581cefbe6ed0d08d2fa89be81082f2a5abdabca8b3c080bf97218a1bd59dc118a30b9f3`，静态链接 |
| MyGo `v0.10.0` | `ebde9399baf2734b3361856a53f4036859973d1d`，MIT |
| purego `v0.11.1` | `15800b5b0a4f3febc099cf663d75ab558a6db8f2` |
| Go | 1.27.4。MyGo 的 `go` 行是 1.27.3 |

Windows 包假定系统自带 UCRT，也就是 64 位 Windows 10 或更新的版本。runner 上的实际系统版本写在该次 artifact 旁的 `windows-report.txt`。Linux 上的 `scripts/build-librnp.sh` 仍使用发行版的 zlib 和 bzip2 共享库，和 Windows zip 里静态链进去的不是同一个二进制。

字段和许可全文见 `deps/pins.json` 与 `third_party/NOTICES.md`。

## 自己构建

Linux：

```sh
bash scripts/build-librnp.sh
export RNPARMOR_LIBRNP="$PWD/out/lib/librnp.so"
go test ./...
CGO_ENABLED=1 go test -race -gcflags=all=-d=checkptr=0 ./...
CGO_ENABLED=0 go build -o rnparmor ./cmd/rnparmor
```

`go test -race` 会打开 checkptr。对 librnp 的 C 缓冲区做 `unsafe.Slice` 会被 checkptr 判为非法指针运算，所以竞态检查加上 `-d=checkptr=0`。竞态检测器本身没有关。这是既有限制，不是为了压掉新的告警。

Windows x64，在装有 Visual Studio C++、CMake、Python 和 Git 的环境：

```powershell
pwsh -File .\scripts\build-windows.ps1
```

脚本核对钉扎、用 MSVC 静态运行库编译 Botan、json-c、zlib、bzip2 和 `rnp.dll`，再编 `rnparmor.exe`，把它们放进 `out\dist\rnparmor-windows-amd64`，在 `PATH` 只剩 `System32` 的情况下跑测试，并尝试启动窗口。zip 在 `out\rnparmor-windows-amd64.zip`。CI 用 `windows-2022` 跑同一脚本并上传 artifact。

## 已经跑过的结果

Linux x86_64、Go 1.27.4 上：`scripts/build-librnp.sh` 成功；`go test` 和带 `-d=checkptr=0` 的 `go test -race` 通过；`go vet` 通过。本机曾启动窗口，`xwininfo` 看到标题 `RnpArmor`。标准错误写明没有 GPU。没有读取像素。

Windows 包的命令、runner 系统版本、测试输出和窗口是否出现，以该次 artifact 里的 `windows-report.txt` 为准。在那份报告写上之前，不要把“编译通过”或“作业变绿”说成 Windows 10/11 桌面已经人工验证。

## 安全限制

- 测试口令在运行时随机生成，不进源码、日志或 zip。
- 失败的解密不把明文写成目标文件。已有文件不覆盖。Linux 用 `renameat2(RENAME_NOREPLACE)`。Windows 用 `O_EXCL` 再写入，不是原子替换。
- 输入上限 32 MiB，输出缓冲上限 40 MiB，口令最长 255 字节。
- 不读默认钥匙圈。保存密钥必须给出显式目录。
- 口令回调在持有 librnp 锁的线程上同步执行，回调里不能再进入会话。
- purego 不释放回调跳板。`Close` 释放槽位、口令副本和 C handle。

## 许可

RnpArmor 自己的许可证还没定。zip 里的 `licenses\` 只是第三方文本。分发二进制时要带上它们。这不是法律意见。
