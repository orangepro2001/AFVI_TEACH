# 构建服务器：自动构建与发布

开发和测试在本机做；推送到 GitHub 之后，拉取源码、编译、打安装包、发布 Release 都由一台 Linux 服务器（ssh 别名 `AI`）完成。本机不需要为发布做任何事。

```
本机 git push ──► GitHub ◄── 服务器每分钟 fetch（systemd 用户定时器 afvi-ci.timer）
                    ▲                │ 有新提交 / 新 tag
                    │                ▼
                    │        scripts/build-wsl.sh（gofmt、vet、纯 Go 测试、三个 exe、安装包）
                    │                │
                    └── GitHub API ◄─┘ 发布 Release（main → 滚动预发布 nightly，v* → 同名 Release）
```

- **为什么轮询而不是 webhook**：服务器在局域网内，GitHub 连不进来。服务器只需要主动出站访问 GitHub；仓库是公开的，拉源码不需要任何凭据，只有“发布 Release”需要一个 token。
- **延迟**：推送后最多约 1 分钟发现，再加构建时间。冷缓存的第一次构建要几分钟（下载 Go 模块、编译 cgo 的 ImGui 包装层）；缓存暖了之后，实测同一提交重建只要 8 秒，改了代码则要重新编译受影响的包。
- **构建本身没有变**：服务器跑的就是仓库里的 `scripts/build-wsl.sh --version <版本>`，和以前 GitHub Actions 跑的一样；版本号规则也一样（tag 构建用 tag 名，main 构建用 `git describe --tags --always`）。
- GitHub Actions 里的 `release.yml` 现在只剩手动 Run workflow（云端试跑，产物只留 artifact），不再自动触发，也不发布。

## 服务器上的文件布局

全部在 `~/ci/afvi-teach/`（登录用户 `tony` 的家目录下），一眼能看出每样东西的用途：

```
~/ci/afvi-teach/
├── README.txt             这棵树的说明（install.sh 写入）
├── bin/ci                 驱动脚本，唯一的程序
├── etc/
│   ├── ci.conf            设置：仓库、保留数量、超时、重试
│   └── github-token       发布用的 GitHub token，权限 600（ci set-token 写入）
├── out/                   结果。一个构建一个文件夹，别的什么都不放
│   ├── main/<日期>-<sha>/     main 的每次构建，保留最新 10 个
│   ├── tags/<tag>/            每个 v* tag 的构建，永久保留
│   ├── latest-main  →         最新的 main 构建
│   └── latest-release →       最新的 tag 构建
│        每个构建文件夹里：AFVI-Teach-Setup-<版本>.exe、SHA256SUMS.txt、build.log、meta.json
├── src.git/               GitHub 仓库的裸镜像（只 fetch，可随时删除，会重新克隆）
├── cache/{gomod,gobuild}/ Go 模块缓存和编译缓存（可随时删除，只是下次构建变慢）
├── work/                  正在进行的构建的临时检出，构建结束即删除
└── state/lock             同一时间只跑一个构建
```

`meta.json` 记录这次构建的一切：提交、版本号、状态（`building` / `built` / `failed` / `interrupted` / `baseline`）、尝试次数、开始和结束时间、安装包的 sha256、工具链版本、是否已发布和 Release 地址。服务器重启后靠它接着干，没有别的状态文件。

定时器和服务的单元在 `~/.config/systemd/user/afvi-ci.{service,timer}`（用户级，不需要 root；该账户已开启 linger，没人登录也照常运行）。

## 一次性准备

1. **安装**（在本机仓库根目录；幂等，可重复运行来更新 `bin/ci` 和单元，不会动 `etc/`、token、构建结果和缓存）：

   ```bash
   deploy/server/install.sh              # --host AI --home ~/ci/afvi-teach 是默认值，--no-enable 只安装不启动定时器
   ```

   它先检查服务器上有 git、go、jq、curl、flock、rsync、mingw-w64（含 windres）和 systemd 用户会话，建好上面的目录，安装 `bin/ci` 与两个 systemd 单元，把此刻已存在的 tag 登记为“已处理”（`baseline`，免得它们被当成新 tag 重新构建、发布），然后启动定时器。

2. **给服务器发布权限**。在 GitHub：Settings → Developer settings → Personal access tokens → Fine-grained tokens → Generate new token：

   - Resource owner：仓库所有者；Repository access：*Only select repositories* → `AFVI_TEACH`
   - Permissions：**Contents: Read and write**（创建 / 删除 Release 与 tag）；**Commit statuses: Read and write**（在提交旁边显示构建成败，可选，没有它只是少一个对勾）
   - Expiration：自己定；过期后服务器的构建照常进行，只是发布失败，换新 token 再执行下一步即可

   **Repository access 不能选 Public repositories**（那个选项是只读的，哪怕仓库是公开的，也不能创建 Release）。然后在服务器上（token 只在终端里输入，不回显、不进入命令行和日志；`set-token` 会真实探测写权限，权限不够就拒绝保存并说明缺什么）：

   ```bash
   ssh -t AI '~/ci/afvi-teach/bin/ci set-token'
   ```

   没有 token 时服务器照样构建，产物留在 `out/`，只是不发布；放上 token 后下一轮会把当前 main 的构建补发出去。

3. **关掉云端的自动触发**：把 `.github/workflows/release.yml`（已改成只有手动触发）提交并推送。**在确认服务器发布成功之后再推**，否则中间会有一段时间没有任何东西在发布。

## 日常操作

下文的 `ci` 就是服务器上的 `~/ci/afvi-teach/bin/ci`，没有放进 PATH；在本机用 `ssh AI '~/ci/afvi-teach/bin/ci <命令>'` 调用（单引号不能省，否则 `~` 会被本机的 shell 展开成本机的家目录）。

```bash
ssh AI '~/ci/afvi-teach/bin/ci status'        # 提交、各构建的状态、定时器、token、占用的磁盘
ssh AI '~/ci/afvi-teach/bin/ci'               # 列出所有命令
```

| 想做的事 | 命令（在服务器上，或前面加 `ssh AI`） |
| --- | --- |
| 看最近几次运行做了什么 | `journalctl --user -u afvi-ci -n 50` |
| 看某次构建的完整日志 | `less ~/ci/afvi-teach/out/main/<文件夹>/build.log` |
| 取最新安装包 | `~/ci/afvi-teach/out/latest-main/AFVI-Teach-Setup-*.exe` |
| 立刻构建 main，不等定时器 | `ci build main` |
| 失败后重新构建（忽略重试次数上限） | `ci build main --force` |
| 重建某个 tag，不发布 | `ci build v1.0.0 --force --no-publish` |
| 把已构建的重新发布 | `ci publish main` / `ci publish v1.0.0` |
| 暂停 / 恢复自动构建 | `systemctl --user stop afvi-ci.timer` / `systemctl --user start afvi-ci.timer` |
| 清理旧构建 | `ci gc`（每次轮询结束也会自动做） |
| 改设置 | 编辑 `etc/ci.conf`，下次运行生效 |

发版：

```bash
git tag v1.0.0
git push origin v1.0.0       # 服务器构建后发布 Release「v1.0.0」；tag 名含 - （如 v1.0.0-rc1）标为预发布
```

## 行为细节

- **main**：每次发现新的 HEAD 就构建，成功后删除旧的 `nightly`（连 tag）并在这次提交上重建，所以 Release 页上只有最新一次 main。构建进行中又有新提交，旧的那个不会发布，随后构建新的。构建失败不会动现有的 `nightly`。
- **v\* tag**：每个 tag 构建一次并发布成自己的 Release（自动生成发布说明）。**tag 被当作不可变的**：已经构建过的 tag 之后被改指别处，服务器不会重新构建；已存在的 Release 不会被覆盖，只补上缺的文件。需要重做就用 `ci build <tag> --force`。
- **失败**：构建失败后隔 `RETRY_WAIT`（默认 10 分钟）自动重试，同一提交最多 `RETRY_MAX`（默认 3）次，因为网络类的偶发失败（下载 Go 模块等）无法和代码错误区分开；用完仍失败就停下，等新提交，或手动 `ci build main --force`。发布失败（网络、token 过期）按同样的间隔重试。失败的构建带着 `build.log` 留在 `out/`，轮询日志里会打印日志末尾 15 行，GitHub 上该提交旁边显示红叉（需要 Commit statuses 权限）。
- **中断**：服务被停止或服务器重启时正在进行的构建标记为 `interrupted`，下次轮询自动重做。
- **并发**：同一时间只有一个构建（`state/lock`）。手动运行遇到定时器正在构建时会直接告知，不排队。
- **资源**：构建以 `Nice=10` 运行，不挤占服务器上的其他任务。
- 构建前后都不改 `out/` 里别的文件夹；`work/` 里的临时检出和 `TMPDIR` 在结束时一并删除。

## 安全

- 服务器会构建、并**执行**（`go vet`、`go test` 会运行测试代码）推送到 `main` 或 `v*` tag 的任何内容。谁能推送到仓库，谁就能在这台服务器上以 `tony` 的身份运行代码；所以请保护好 GitHub 账号（开启两步验证），不要把别人的写权限加进来。
- token 只授予这一个仓库；存放在权限 600 的 `etc/github-token`，调用 GitHub 时通过管道交给 curl，不出现在进程列表、命令行或日志里。怀疑泄露时在 GitHub 上撤销，再 `ci set-token` 换新的。
- 安装包未签名，也不含 `reference/` 机台数据；服务器不运行 `-selftest`（需要 Windows 桌面）。

## 卸载

```bash
ssh AI 'systemctl --user disable --now afvi-ci.timer; rm ~/.config/systemd/user/afvi-ci.{service,timer}; systemctl --user daemon-reload'
ssh AI 'rm -rf ~/ci/afvi-teach'      # 连同全部构建结果；只想省空间就删 cache/ 和 src.git/
```
