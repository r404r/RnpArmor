# 第三方许可

RnpArmor 本身采用 Apache License, Version 2.0（详见仓库根目录 `LICENSE`，Copyright 2026 r404r）。下面记录本仓库实际链接或导入的上游第三方组件许可文本，不构成法律意见。分发二进制时需要随包带上项目 `LICENSE` 及对应第三方许可全文。`third_party/licenses/` 里的文件是从钉扎来源原样复制的。

| 组件 | 钉扎 | 许可 | 文本 |
| --- | --- | --- | --- |
| librnp | tag `v0.18.1`，提交 `aad1892e4d8423398a0b973bd0a7e4544c359afc` | BSD-2-Clause，另有文件内列出的 NetBSD/NetPGP 等通知 | `third_party/licenses/rnp-LICENSE.md` |
| sexpp | 提交 `c641a2f36520bab783657a58650d9fda548b9dec`（rnp 该提交的 gitlink；版本号 0.9.0） | MIT（Ribose 与原始 MIT 声明都在同一文件） | `third_party/licenses/sexpp-LICENSE.md` |
| Botan | 3.10.0，tarball SHA-256 `fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24` | BSD-2-Clause | `third_party/licenses/botan-license.txt` |
| json-c | 0.19，tag `json-c-0.19-20260627`，tarball SHA-256 `37ad0249902e301bd9052bf712e511fcc6acff4ecaad4b5900aad9ce564e26de` | MIT。rnp v0.18.1 仍链接它；更晚的 rnp `main` 才把 JSON 放进库内，那不是这个 tag | `third_party/licenses/json-c-COPYING` |
| MyGo | `v0.10.0`，提交 `ebde9399baf2734b3361856a53f4036859973d1d` | MIT，Copyright (c) 2026 MyGo contributors | `third_party/licenses/mygo-MIT.txt` |
| purego | `v0.11.1`，提交 `15800b5b0a4f3febc099cf663d75ab558a6db8f2` | Apache-2.0 | `third_party/licenses/purego-Apache-2.0.txt` |

Windows 包把下面两份静态链进 `rnp.dll`，许可文本放进 zip 的 `licenses/`：

- zlib 1.4，tarball SHA-256 `02dd159783677a749c04adbdd02b55b8a2bcc7e054f51b76b549199a916ad584`（GitHub release asset digest，下载后再算过一次）。许可是 zlib 许可。
- bzip2 1.0.8，sourceware `sha512.sum` 中的 SHA-512 `083f5e675d73f3233c7930ebe20425a533feedeaaa9d8cc86831312a6581cefbe6ed0d08d2fa89be81082f2a5abdabca8b3c080bf97218a1bd59dc118a30b9f3`。同一文件的 SHA-256 是 `ab5a03176ee106d3f0fa90e381da478ddae405918153cca248e682cd0c4a2269`。许可是 Julian Seward 的四条款文本。

Windows 构建 `librnp` 还会使用下列头文件兼容依赖，其许可文本同样放入 Windows portable zip 的 `licenses/dirent-LICENSE`：

- `dirent` `1.24`（`tronkko/dirent` v1.24，`https://github.com/tronkko/dirent`），MIT 许可：
  - 头文件来源：`https://raw.githubusercontent.com/tronkko/dirent/1.24/include/dirent.h`
  - `header_sha256`：`7383044a375d481ac8ad7ec2f43151263eca792f085001a8020cc590114a06a6`
  - 许可来源：`https://raw.githubusercontent.com/tronkko/dirent/1.24/LICENSE`
  - `license_sha256`：`2f414f6c48e8c25ffb5d03cddd6043b3b613187f758d0a6964ca2ad7ed749e50`
  - Windows portable 包内路径：`licenses/dirent-LICENSE`

Linux 测试仍动态链接发行版的 zlib 和 bzip2，不是上面这两个钉扎 tarball 编出来的同一份二进制：

- zlib1g `1:1.3.dfsg-3.1ubuntu2.3`，zlib 许可。
- libbz2 `1.0.8-5.1ubuntu0.1`，bzip2 自己的四条款许可。

Windows portable 目录如果带上 `zlib1.dll`、`bz2.dll` 或其他非系统 DLL，必须使用那次构建实际产生的文件，并把对应许可文本放进同一个目录。不要拿这份 Linux 包里的 soname 当作 Windows DLL 文件名。

macOS `.dmg`（`RnpArmor.app`）中的 `RnpArmor.app/Contents/MacOS/librnp.dylib` 静态链入钉扎的 `Botan 3.10.0`、`json-c 0.19`（及 `sexpp`），并动态链接 macOS 系统自带的 `/usr/lib/libz` 与 `/usr/lib/libbz2`（以及系统 `Security` framework）。`RnpArmor.app/Contents/Resources/licenses/` 包含下列第三方许可全文：

- `rnp-LICENSE.md`
- `sexpp-LICENSE.md`
- `botan-license.txt`
- `json-c-COPYING`
- `mygo-MIT.txt`
- `purego-Apache-2.0.txt`

`scripts/build-librnp.sh` 会把 rnp、Botan、sexpp 的许可文件再复制到 `out/licenses/`。`out/` 不提交。
