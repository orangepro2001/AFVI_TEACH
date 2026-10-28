# 程序内自动更新（应用内接受更新推送）

程序自己问更新服务：「有没有比我现在更新的版本？」有就**在后台下载并校验**，下载好了才提示，用户点「安装并重启」才真正安装。安装包换掉文件后自动启动新版本，用户不需要去找安装包、也不需要 ssh 或共享目录。

服务端是 `release-publish`（命令名 `relhub`），发布流程见 [BUILD.md](BUILD.md#更新服务release-publish--relhub)。本文是**客户端**这一侧：怎么工作、怎么关、怎么换服务器、怎么排错。

## 1. 用户看到什么

1. 程序启动 20 秒后检查一次，之后每 6 小时一次（**后台**，界面不卡）。
2. 有新版本：自动下载（十几 MB，带进度），对话框同时出现，说明版本号、大小、发布说明。
3. 下载并**校验签名和 sha256** 之后，按钮才变成可用的「安装并重启」。
4. 点「安装并重启」：安装包以 `-silent -dir <本程序所在目录> -launch` 运行，本程序退出，装完后新版本自动启动。
5. 也可以选「稍后」（对话框关掉，安装包留着，下次提问）、「忽略此版本」（这个版本不再提示，直到有更新的版本）。

只提示、不下载的对手方是没有的：**策略就是自动下载、用户确认后安装**，工厂现场不会被突然重启打断。安装包替换文件时正在运行的 exe 会被改名成 `.old`，所以「稍后」是安全的。

菜单 **帮助 ▸ 检查更新…** 可以随时手动检查；手动打开时也会明确告诉你「已是最新版本」。

## 2. 它信任什么

- **签名**：更新服务的清单用 Ed25519 签名，程序里固定了公钥（`internal/app/update.go` 的 `updateKeys`，当前 `keyId f5faa9f9858b789f`）。签名对不上、清单属于别的 app/平台/渠道、字段不合规、下载到的字节 sha256 不符——任何一种情况都拒绝，装都不会装。
- **版本号**：程序自己的版本号由构建时通过 `-ldflags` 写进 `internal/version`（规则见 `scripts/version.sh`，与发布脚本同一份）。比较的是单调递增的整数 `versionCode`，不是字符串。
- **不信任的目标也包含「自证」**：没有版本戳的构建（`go run`、`go test`、开发者自己 `go build`）是 `dev`，**不参与更新**，见第 4 节。

**密钥轮换**：服务端换签名密钥之前，必须先把新公钥发到客户端（加进 `updateKeys`），否则换了之后老客户端会拒绝一切更新。临时换服务器测试可以用 `AFVI_UPDATE_KEYS` 追加一把公钥。

## 3. 开关与配置

| 开关 | 作用 |
| --- | --- |
| `-update <url>` | 更新服务地址（含前缀，如 `https://host/release`）；`-update off` 关闭 |
| `-update off` | 完全关闭检查（对话框只说"已关闭"） |
| `AFVI_UPDATE_URL` | 同 `-update`，命令行优先 |
| `AFVI_UPDATE_KEYS` | 追加受信公钥（base64，逗号/空格分隔），用于轮换或测试 |

默认地址是构建服务器的 Tailscale Serve 前端 `https://tony-ai-pc-1.tail9f7f1f.ts.net/release`；应用编号 `afvi-teach`、平台 `win-x64`、渠道 `stable`。

本地验证（不需要网络，也不需要打 tag）：

```bash
# 已经把 0.1.12 发到本机 relhub 的情况下：
./bin/teach-linux -update http://127.0.0.1:18790/release   # 或 Windows 上的 bin\teach-imgui.exe
```

安装包下载到**程序所在目录**的 `updates/`（例如 `%LOCALAPPDATA%\Programs\AFVI Teach\updates\`）。安装到 Program Files 这类不可写的位置时下载会失败并提示——安装程序默认装在每用户可写的位置，就是为了这个。

## 4. 版本号怎么来的

`scripts/version.sh` 是唯一实现，构建和发布都调用它：

```
base  = 最近的 v* tag 的 MAJOR.MINOR.PATCH（预发布后缀只属于 tag 名）
N     = 该 tag 之后的提交数
patch = tagPatch*100 + N        code = MAJOR*10000 + MINOR*100 + patch
```

`build-wsl.sh` 用 `-X afvi/teach-imgui/internal/version.Name=… -X …Code=…` 写进两个 GUI，`publish-relhub.sh` 用同一规则发布，所以「程序认为自己是几号」和「服务端认为这个包是几号」永远一致。发布用的**文件名**仍然是 `git describe`（如 `AFVI-Teach-Setup-nightly-7-ge2fedb2.exe`），便于追提交。

`go run` 和 `go test` 没有版本戳（`dev` / 0），更新检查自动关闭——否则开发机每次都会"发现新版本"。

## 5. 自测

`-selftest` 里有一个 `update-check` 步骤：它在进程内起一个**临时 HTTP 服务**，用临时生成的密钥签一份清单，然后完整走一遍检查→验签→下载→校验→"可以安装"的状态，断言版本号、字节数和文件确实落盘，最后把状态恢复成关闭。它**从不点「安装并重启」**（那会真的运行安装器）。截图见 `native-update.png`。

`AFVI_UPDATE_LIVE` 让 `internal/updater` 的单元测试打真服务器（默认跳过）：

```bash
AFVI_UPDATE_LIVE=http://127.0.0.1:18790/release CGO_ENABLED=0 go test ./internal/updater -run Live -v
```

它验证的是最要紧的一件事：本程序算出来的签名字节和真服务器的**逐字节一致**（用的是真服务器的公钥，签名对不上就会失败）。

## 6. 排错

| 现象 | 原因 / 处理 |
| --- | --- |
| 对话框说"开发构建，不参与自动更新" | 这个 exe 没有版本戳：用 `./scripts/build-wsl.sh` 构建的版本才会有 |
| "检查更新失败：cannot reach …" | 机器不在 tailnet 上，或者服务没跑（`systemctl --user status` / `podman ps`，`ci status` 里也有 relhub 那一行） |
| "签名不匹配任何固定密钥" | 服务端换过密钥而客户端还没更新；把新公钥加进 `updateKeys` 重新构建 |
| 一直"有新版本"但装不上 | 看 `updates/` 里是否有 `.part` 文件（断了会续传）；sha256 不符时 `.part` 会被删掉重下 |
| 装了以后版本号没变 | 安装器可能装到了别的目录：它带的是 `-dir <程序所在目录>`，检查 `%LOCALAPPDATA%\Programs\AFVI Teach` 与快捷方式指向的位置 |

日志：没有单独的日志文件，失败原因直接显示在对话框里（`update.err.*`），服务端那边看 `~/ci/afvi-teach/bin/ci status` 和 `relhub stats`。
