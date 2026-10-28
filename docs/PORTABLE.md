# 两种发行形态：安装版与绿色版

同一套程序有两种发行方式，**功能完全一样**，只有"怎么装、更新怎么落地"不同。

| | 安装版（默认） | 绿色版（免安装） |
| --- | --- | --- |
| 文件 | `AFVI-Teach-Setup-<版本>.exe` | `AFVI-Teach-Portable-<版本>.zip` |
| 使用 | 运行安装向导（或 `-silent`） | 解压到任意可写目录，运行 `teach-imgui.exe` |
| 写入系统 | 开始菜单 / 桌面快捷方式（可选）、PATH（可选）、卸载项、安装日志 `uninstall.json` | **什么都不写**：配置、输出、更新全在自己文件夹里 |
| 卸载 | `uninstall.exe`（默认保留 `afvi.toml` 与 `captures`） | 直接删掉文件夹 |
| 更新平台 | `win-x64` | `win-x64-portable` |
| 更新方式 | 下载安装包 → `setup.exe -silent -dir <本程序目录> -launch` | 下载 zip → 关闭后由脚本替换文件 → 自动重启 |

两者都**支持自动更新推送**：启动 20 秒后检查，每 6 小时一次，发现新版本自动下载、验签 + 校验 sha256，用户点「安装并重启」才动手。区别只在最后一步怎么落地，见 [UPDATE.md](UPDATE.md)。

## 绿色版的判定

绿色版压缩包里有一个 `portable.txt` 标记文件（也是给用户看的说明）。程序启动时看自己所在目录里有没有它：

- **有** → 绿色模式：向更新服务要 `win-x64-portable` 平台的**压缩包**，点「安装并重启」后：
  1. 把下载到的 zip 解到 `updates\staged\`
  2. 在 `updates\` 里写一个 `apply-update.cmd`
  3. 本程序退出，`apply-update.cmd` 用 `robocopy` 把文件覆盖到程序目录（**只覆盖包里的文件，不删除** `afvi.toml`、`captures` 等用户数据；exe 还被占用时最多重试 60 秒）
  4. 启动新的 `teach-imgui.exe`，删除脚本自己
- **没有** → 安装版：要 `win-x64` 平台的**安装包**，交给安装器静默升级（它会把正在运行的 exe 改名成 `.old`）。

两个平台在更新服务里是分开的，所以**不会互相串味**：安装版不会被一个裸文件夹替换掉（那会丢快捷方式与卸载项），绿色版也不会莫名其妙变成已安装。

删掉 `portable.txt` 就等于告诉程序"我现在是安装版"——不要这么做，除非你确实把整个文件夹装成了安装版。

## 打包

```bash
./scripts/package-portable.sh                 # 先构建三个 exe，再打 zip
./scripts/package-portable.sh --no-build      # 只打包 bin/ 里现成的程序
./scripts/build-wsl.sh                        # 一键构建：安装包 + 绿色 zip（--no-portable 跳过 zip）
./scripts/package-portable.sh --reference reference   # 连同机台数据打包（默认不含）
```

产物：`dist/AFVI-Teach-Portable-<版本>.zip`，内容与安装包的载荷一致：

```
teach-imgui.exe  camera-gui.exe  camera.exe
afvi.example.toml  portable.txt
docs/{README,MANUAL,USAGE,ROLES,GOLDEN,LINK,INSPECT_RULES,SCENARIO,REFERENCE_SETUP,UPDATE,PORTABLE}.md
```

版本号默认取 `git describe --tags --always --dirty`（和安装包同一个名字，便于追提交）。

## 发布

安装包与绿色包**一起**发到更新服务和 GitHub Release：

```bash
./scripts/publish-relhub.sh                                   # 安装包 -> win-x64
./scripts/publish-relhub.sh dist/AFVI-Teach-Portable-*.zip --platform win-x64-portable
./scripts/build-wsl.sh --publish-relhub                       # 上面两步都做
```

构建服务器（`ci poll`）每次构建都会做这两步，见 [BUILD.md](BUILD.md#更新服务release-publish--relhub) 与 [SERVER_CI.md](SERVER_CI.md)。

## 注意

- 绿色版要放在**可写**目录：配置 `afvi.toml`、相机输出 `captures`、下载的更新 `updates` 都写在程序旁边。放在 `Program Files` 这类地方会写不进去。
- 绿色版**没有**开始菜单项、没有 PATH 入口、不能用"应用和功能"卸载——这是设计意图。
- 两种形态共用同一份 `afvi.toml` 格式，所以把一个绿色文件夹里的 `afvi.toml` 拷到安装版目录里也能用。
- 安装包内含 `docs/` 全套文档；绿色 zip 也一样（`docs/PORTABLE.md` 与 `docs/UPDATE.md` 都在里面）。
