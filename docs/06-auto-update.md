# 06 · 自动更新（0.3.0 起）

> 前置：0.3.0 已发布
> 目标版本：0.3.0 / 0.3.1 / 0.3.2

## 1. 目标

- 用户安装 ZPad 后，应用自己发现新版本、显示发布说明、下载并安装，不需要重新下载 DMG。
- 更新包必须可验证：只接受用本项目私钥签名的新版本，走 HTTPS 也不额外信任服务器。
- 不引入 WebView：更新窗口与主界面一样由 MyGo 原生 UI 绘制。
- 发布产物沿用 MyGo 默认命名，不再额外改名。

## 2. 方案

使用 MyGo v0.10.0 内置的更新链路：

| 部分 | 用途 |
| --- | --- |
| `mygo.Updater` | 检查清单、下载、校验签名、替换应用、重启 |
| `plugins/updater/native` | 原生 UI 更新窗口（发布说明、进度、跳过 / 稍后） |
| `mygo keygen` | 生成 Ed25519 密钥对 |
| `mygo build` | 用私钥签名更新包，生成增量包与每平台清单 |
| `mygo.json` 的 `updates` | 编入公钥与更新源 |

### 2.1 更新源

GitHub Releases：`updates.github = "helloxz/zpad"`，默认 `tagPrefix` 为 `v`。

- 清单地址固定为 `https://github.com/helloxz/zpad/releases/latest/download/update-<target>.json`。
- 更新包地址为 `https://github.com/helloxz/zpad/releases/download/v<版本>/<文件名>`。
- 因此发布必须标记为 **latest**，draft 阶段应用看不到。

### 2.2 签名

- 私钥 `mygo-update.key` 只存在本机与 GitHub Actions secret `MYGO_UPDATER_PRIVATE_KEY`，不进仓库。
- 公钥写在 `mygo.json` 的 `updates.publicKey`，由 `mygo build` 编进二进制。
- 私钥丢失后，已安装用户无法再收到更新（只能重新下载安装包）。`release_test.go` 会校验公钥是合法的 Ed25519 公钥。

### 2.3 应用内集成

- `main.go` 在单实例锁之后调用 `mygo.Use(native.New(updater.Options{Icon: appIconPNG}))`。
- 菜单：ZPad（macOS）或帮助（其他平台）里，About 之后是「检查更新…」，调用 `updater.CheckForUpdates()`。
  - 文案走 `locales.CheckForUpdates()`，不直接用插件的 `updater.MenuItem()`：后者的标签在未注册插件的测试里会 panic，也不跟随应用自己的语言表。
- 更新窗口自带中英文、亮暗外观，跟随 `mygo.App.SetLocale` 与主题；用户选择（自动检查、跳过版本）存在用户数据目录的 `updater.json`，与 `settings.json` 分开。

### 2.4 增量更新

`mygo build` 默认读取已发布清单，为最近 3 个版本生成 `<slug>-<from>-to-<to>-<target>.delta`。首次带更新器的版本没有历史清单，因此没有增量；之后自动生效。增量失败时更新器回退到完整包。

## 3. 文件名约定

| 文件 | 说明 |
| --- | --- |
| `zpad-0.3.2-arm64.dmg`、`zpad-0.3.2-amd64.dmg` | `mygo build` 默认命名，CI 不改名 |
| `zpad-setup-0.3.2-amd64.exe` | `mygo build` 默认命名 |
| `zpad-0.3.2-windows-amd64-portable.zip` | CI 额外打的 Windows 绿色版，唯一非 MyGo 产物；文件平铺在压缩包根目录，解压即得 `ZPad.exe` |
| `zpad-0.3.2-darwin-arm64.tar.gz` | 不可改名：manifest 里写死了 URL |
| `zpad_<版本>_amd64.deb`、`zpad-<版本>-1.x86_64.rpm` | Linux 软件包，`linux.formats` 决定写哪些 |
| `zpad-<版本>-linux-amd64.tar.gz` | Linux 通用包，`install.sh` 装的就是它，不可改名 |
| `install.sh` | Linux（以及带更新的 macOS）安装脚本，装到 `~/.local` 才能自更新 |
| `zpad-<旧版本>-to-0.3.2-<target>.delta` | 不可改名 |
| `update-<target>.json` | 不可改名：应用从 `releases/latest/download/` 固定读取 |

`zpad` 前缀来自 `mygo.json` 的 `slug`（默认应用名小写），平台目标为 `darwin-arm64`、`darwin-amd64`、`windows-amd64` 等。0.3.1 起不再把 dmg / exe 改成 `ZPad-...`，所有 MyGo 产物原样上传；macOS 也不再额外提供 `.zip`。

## 4. 发布流程

1. 修改 `mygo.json` 的 `version`，并在 `CHANGELOG.md` 写 `## <版本>` 一节（更新窗口显示这一节，缺失会导致构建失败）。
2. 提交、打 tag `v<版本>`、推送，CI 创建 draft release。
3. macOS、Windows 与 Linux 作业带 `MYGO_UPDATER_PRIVATE_KEY` 运行 `go tool mygo build -upload`：
   - macOS：生成并上传 dmg、`.tar.gz`、`.delta`、`update-darwin-*.json` 与 `install.sh`。
   - Windows：生成并上传安装包、`.tar.gz`、`.delta`、`update-windows-amd64.json` 与 `install.ps1`。
   - Linux：生成并上传 `.deb`、`.rpm`、`.tar.gz`、`.delta`、`update-linux-amd64.json` 与 `install.sh`。
   - `-upload` 自己按“安装包 / 更新包 → manifest”的顺序上传，并合并各作业的 `SHA256SUMS`。
4. Windows 作业额外打 `zpad-<版本>-windows-amd64-portable.zip` 并上传：排除 `.mygo-files`、安装包与更新文件后，把内容平铺进压缩包根目录（不套 `ZPad/`），并校验 `ZPad.exe` 位于归档根目录。
5. 在 GitHub 上检查 draft，**等 macOS、Windows 与 Linux 作业都变绿、资产齐全**再发布为 **latest**（不要勾选 pre-release）。

   发布时机不能提前：release 一发布就是 `releases/latest`，应用会立刻检查到新版本；此时更新清单还没上传，`releases/latest/download/update-<target>.json` 会返回 404，所以等文件齐全再 Publish。

6. 发布后到 CF Pages → Deployments → **Retry deployment**（或 Create deployment，选 `main`）触发一次构建。官网构建时会从 GitHub API 取最新 release，版本号、下载地址、`_redirects` 短链和首页「最近更新」一起更新；仓库不会提交任何东西。

本地构建需要签名时：

```sh
MYGO_UPDATER_PRIVATE_KEY="$(cat "$HOME/Library/Application Support/mygo/update-keys/mygo-update.key")" go tool mygo build
```

不设置私钥时构建会跳过签名，也就没有更新文件（适合本地打包验证 UI，但发布前必须补齐）。

## 5. 已知限制

- **未签名、未公证**：自更新本身可用（Ed25519 验签，下载的文件不带 quarantine），但首次从 DMG 安装仍会被 Gatekeeper 拦截，这是 0.2.0 已有的问题。将来加 Developer ID 不影响更新链路，只要 Ed25519 私钥不变。
- **ad-hoc 签名**每次构建指纹不同，macOS 可能把更新后的应用当成新的应用，TCC 权限（文件与文件夹等）可能需要重新授权；ZPad 主要使用系统打开 / 保存面板，影响较小，需要实测。
- 应用必须能写安装目录：`/Applications` 对管理员用户可写；标准用户需要装到 `~/Applications`。从只读 DMG 直接运行、或由包管理器安装时 `Updater.Enabled()` 为 false。
- Linux：`.deb` / `.rpm` 装到 `/opt/zpad` 与 `/usr/bin/zpad`，由包管理器更新，`Updater.Enabled()` 为 false；只有 `curl -fsSL https://github.com/helloxz/zpad/releases/latest/download/install.sh | sh` 装进 `~/.local/ZPad.app`（命令在 `~/.local/bin/zpad`）的实例能自更新。AppImage 不能自更新，MyGo 直接拒绝。deb 只声明 `libgtk-3-0`（`linux.webview: false`），不依赖 WebKitGTK。
- 开发版（`mygo dev` / `go test` / `go run`）不能更新，`Check` 返回 `mygo.ErrUpdatesDisabled`；更新窗口会说明原因。
- 第一个带更新器的版本（0.3.0）自己收不到更新；0.3.1 是第一个可自更新的版本。
- 更新时 `Relaunch` 走正常退出流程，会触发 ZPad 的未保存汇总提示；用户取消退出则更新在下次启动时生效。
- 绿色版 Windows 应用放在可写目录才能自更新；装进 `Program Files` 的机器级安装不能。
- 更新窗口是独立的原生窗口，不参与主窗口的会话保存（打开 / 位置不持久化）。

## 6. 实现记录

- `main.go`：注册原生更新插件，图标取二进制内嵌的 `resources/icon.png`。
- `internal/app/menu.go`：About 之后新增「检查更新…」；macOS 在应用菜单，其他平台在帮助菜单。
- `internal/app/view.go`：左侧工具栏末尾加信息按钮（打开关于），右侧工具栏行首加检查更新按钮；图标为 `icons.go` 的 `iconInfo` / `iconUpdate`。两者与菜单共用 `App.checkUpdates`。
- `locales/en.json`、`locales/zh-Hans.json`、`locales/catalog_gen.go`：`checkForUpdates`。
- `mygo.json`：新增 `updates`（公钥 + `helloxz/zpad`）；0.3.1 起版本号随之更新。
- `.github/workflows/release.yml`：构建带私钥并直接用 `-upload` 上传，不再重命名安装包；Windows 额外打绿色版并排除安装包与更新文件。官网不再由 release 工作流更新：构建时 Hugo 自己取最新 release，发布后到 CF Pages 手动重建。
- `release_test.go`：公钥合法性、更新源与 CHANGELOG 版本小节。
- `internal/app/menu_test.go`、`internal/app/about_ui_test.go`：菜单与工具栏的「检查更新」都可点击，并断言调用一次检查；点击通过 `App.checkUpdates` 注入的计数器，避免测试里未注册插件时 panic。工具栏的信息按钮断言打开关于对话框。
- `CHANGELOG.md`：0.3.0、0.3.1 与 0.3.2 小节。
- 文档：本文件、`docs/README.md`、`README.md`、`AGENTS.md`。

密钥（2026-11-23 生成，`mygo keygen`）：

- 私钥：`~/Library/Application Support/mygo/update-keys/mygo-update.key`（同时需要配置为仓库 secret `MYGO_UPDATER_PRIVATE_KEY`）。
- 公钥：已写入 `mygo.json` 的 `updates.publicKey`。

## 7. 验收清单

- [ ] 0.3.2 发布后，「检查更新…」在中文 / 英文下都显示正确文案。
- [ ] 工具栏右侧行首的检查更新按钮与菜单行为一致；左侧末尾的信息按钮在亮 / 暗主题下都打开关于对话框。
- [ ] 开发版点击「检查更新…」提示不能更新，并说明原因。
- [ ] 安装的 0.3.1 在启动后或手动检查时发现 0.3.2。
- [ ] 更新窗口显示 0.3.2 的 `CHANGELOG.md` 小节，可安装、稍后提醒、跳过。
- [ ] 安装后重启，版本号变为 0.3.2，首页快捷键提示按平台显示（Windows / Linux 为 `Ctrl+`），会话恢复、未保存提示行为正常。
- [ ] 0.3.2 自身检查时显示「已是最新」，不再出现 404。
- [ ] 0.3.3 发布时生成 `0.3.2-to-0.3.3` 增量包，下载失败时回退完整包。
- [ ] 已安装目录不可写时，更新窗口说明不能更新。
- [ ] 未保存文档存在时点「安装并重启」，先出现 ZPad 的保存提示；取消后更新在下次启动生效。

## 8. 签收

| 项目 | 内容 |
| --- | --- |
| 状态 | 待签收 |
| 签收人 | |
| 签收日期 | |

签收意见：
