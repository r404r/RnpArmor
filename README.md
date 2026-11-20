# RnpArmor

最小安全垂直切片：MyGo 原生窗口加上 librnp。这里还不是文件工具的完整界面。

仓库最初只有这一行标题。现有内容保留，没有另建仓库，没有合并，没有发布 release，也没有为项目选定开源许可证。

研究笔记 `p-research-rnp` 的 `0017-gui-feasibility.md`（`12eee8f6de7a4f3031cce254da284fe11a1e27ff`）没有跑过原型，也没有在真实 Windows 上跑过。下面凡是写“通过”的，都是这次在 Linux 上实际跑出来的结果。

## 方案

比较了两条可审计的接法，没有改 UI 栈。

MyGo `ffi` 包不能直接登记 C 回调。`v0.10.0`（`ebde9399baf2734b3361856a53f4036859973d1d`）和较晚的 `e88574c82acca009a971c822048eb4d11d5c049a` 的 `ffi` 树都是 `e557e834c3a698e5cf18959cc430750851ddba5d`。该目录只通过 `purego.RegisterFunc` 把 C 函数绑给 Go。Windows 加载用 `syscall.LoadLibrary`，依赖 DLL 仍走默认搜索，包含当前目录和 `PATH`。因此不用这个包去加载 librnp。

另一条是 cgo/C 桥，或单独的辅助进程。cgo 会让构建依赖 C 编译器，Windows 上还受 MinGW 限制。辅助进程再添一个状态所有者和一套口令 IPC。这一阶段放不进可重复的最小验证。

采用的接法：MyGo 只画原生窗口；librnp 用 MyGo 已经依赖的 purego `v0.11.1` 直接调用。口令回调是进程内一个 `purego.NewCallback` 跳板，`app_ctx` 只传整数槽位，不把 Go 指针交给 C。purego 不会释放这个跳板。`Close` 释放的是槽位、口令副本、密钥 handle 和 `rnp_ffi_t`。Windows 加载代码使用绝对路径和 `LoadLibraryEx` 的 `LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32`。这条加载代码已能交叉编译，没有在 Windows 上执行。

一个 `Session` 持有一把锁，再进进程级的 librnp 锁。口令回调发生在持锁的那个 OS 线程上，回调里不能再进 `Session`。界面线程要用 `Session.Go`，不要直接跑加解密。

## 钉扎

| 组件 | 来源 | 标识 |
| --- | --- | --- |
| Go 工具链 | `https://go.dev/dl/go1.27.4.linux-amd64.tar.gz` | SHA-256 `7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840`。MyGo 的 `go` 行是 1.27.3，这里用当时 JSON 列出的稳定版 1.27.4 |
| MyGo | `github.com/egoist/mygo` `v0.10.0` | `ebde9399baf2734b3361856a53f4036859973d1d`，MIT |
| purego | `v0.11.1` | `15800b5b0a4f3febc099cf663d75ab558a6db8f2`，Apache-2.0 |
| librnp | tag `v0.18.1` | `aad1892e4d8423398a0b973bd0a7e4544c359afc`。不要用已撤回的 `v0.18.0` |
| sexpp | rnp 该提交的 gitlink | `c641a2f36520bab783657a58650d9fda548b9dec` |
| Botan | `https://botan.randombit.net/releases/Botan-3.10.0.tar.xz` | SHA-256 `fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24`，与官网 `sha256sums.txt` 一致 |
| json-c | release `json-c-0.19-20260627` 的 `json-c-0.19.tar.gz` | SHA-256 `37ad0249902e301bd9052bf712e511fcc6acff4ecaad4b5900aad9ce564e26de`。v0.18.1 的 CMake 仍要求 json-c；更晚的 rnp `main` 才把 JSON 编进库里，那不是这个 tag |

完整字段在 `deps/pins.json`。许可文本在 `third_party/NOTICES.md`。项目自己的 LICENSE 未决。

## 构建和测试

```sh
bash scripts/build-librnp.sh
export RNPARMOR_LIBRNP="$PWD/out/lib/librnp.so"   # 必须是绝对路径
go test ./...
CGO_ENABLED=1 go test -race -gcflags=all=-d=checkptr=0 ./...
CGO_ENABLED=0 go build -o rnparmor ./cmd/rnparmor
```

`RNPARMOR_LIBRNP` 若是相对路径会被拒绝。加载前会核对同目录的 `SHA256SUMS`。脚本把 `libbotan-3.so.10` 和 `libjson-c.so.5` 放在 `librnp.so` 旁边，并把 RUNPATH 设为 `$ORIGIN`。不要靠当前目录或 `LD_LIBRARY_PATH`。zlib 和 bzip2 仍用发行版的系统库。

`go test -race` 会打开 checkptr。对 librnp 的 C 缓冲区做 `unsafe.Slice` 会被 checkptr 判为非法指针运算，所以竞态检查要加 `-d=checkptr=0`。这不是关掉竞态检测器。

Windows 打包脚本是 `scripts/build-librnp.ps1` 和 `scripts/package-portable.ps1`。前者在核对钉扎后停止，因为没有在 Windows 上试过编译参数。后者要求绝对路径，并在复制前核对 `SHA256SUMS`。两个脚本都没有在本环境执行。它们不会创建 GitHub release。

## 这次实际跑过的结果

环境：Linux x86_64，Go 1.27.4，gcc/g++ 13.3.0。没有 Windows 机器。

- `bash scripts/build-librnp.sh` 成功。`out/lib/ldd.txt` 里 `libbotan-3.so.10` 和 `libjson-c.so.5` 都解析到 `out/lib/`，没有设置 `LD_LIBRARY_PATH`。`libz` 和 `libbz2` 来自 `/lib/x86_64-linux-gnu`。
- `go test ./internal/rnp/ ./internal/loadlib/` 通过。覆盖版本字符串 `0.18.1`、正确口令、错误口令、取消、重复回调、关闭后回调失效、回调与调用方同一 OS 线程、临时 EDDSA/Curve25519 密钥的文件加密解密、篡改失败且不留下明文、分离签名与验签、篡改签名和篡改原文、验签结果的 `IdentityTrusted` 恒为 false、已有文件不覆盖、空文件和超限拒绝、显式目录保存/加载、不写 `HOME`、`Session.Go` 在 librnp 调用结束前返回、取消后不发布输出。
- `CGO_ENABLED=1 go test -race -gcflags=all=-d=checkptr=0 ./internal/rnp/ ./internal/loadlib/` 通过。
- `go vet ./...` 通过。
- `CGO_ENABLED=0 go build` 编出 Linux 的 `cmd/rnparmor`，也编出 `GOOS=windows` 的 exe。Windows 那个 exe 没有运行。
- 在本机 `DISPLAY=:1` 上启动窗口约 5 秒。进程保持运行，`xwininfo` 看到标题为 `RnpArmor`、约 558×271 的窗口。标准错误是 `mygo: native UI draws without the GPU: there is none`。没有读取窗口像素，所以不能把界面上的版本文字说成已经目视核对。版本文字的核对来自 `Probe` 测试。没有 GPU 的绘制路径也没有在 Windows 上出现过。

测试口令在运行时随机生成，不写入源码、日志或仓库。没有提交私钥或 `out/` 里的二进制。

## 行为边界

- 输入上限 32 MiB，librnp 输出缓冲上限 40 MiB，口令最长 255 字节（librnp 的缓冲区是 256，含结尾 NUL）。
- 解密和验签失败时不把输出提交到目标路径。明文先留在内存里，成功后才发布。
- Linux 上发布用同目录临时文件 `.rnparmor-*`，写完后 `renameat2(RENAME_NOREPLACE)`。目标已存在就失败并保留原文件。崩溃时可能留下临时文件，其中可能有输出字节，需要删除。非 Linux 的发布是 `O_EXCL` 再写入，不是原子替换，而且没有跑过。
- 持久化只有 `SaveTo` / `LoadFrom` 的显式绝对目录。不调用 `rnp_get_default_homedir`，不写 `%APPDATA%` 或家目录 `.rnp`。目录不可写时返回错误，不改去别处。
- 签名数学有效只说明它和本会话里的钥匙对得上。`IdentityTrusted` 固定为 false。
- 已经开始的 librnp 调用不能从中途取消。取消只保证调用返回后不发布输出。
- 校验和打开之间有时间差。校验文件不在仓库里跟踪，由构建脚本写到 `out/lib/SHA256SUMS`。

## 还没验证的事

- 真实 Windows 10/11 上的加载、回调、打包和运行。Linux 交叉编译与 CI 编译都不能代替。
- `.github/workflows/smoke.yml` 已提交，这次没有等待 GitHub Actions。即使以后变绿，也不证明干净的 Win10/11。
- MyGo 在没有 GPU 的机器上的绘制质量、IME 和高 DPI。
- 智能卡、钥匙服务器、信任模型和完整界面。
