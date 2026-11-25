# AGENTS.md

ZPad 是一个基于 MyGo 原生 UI 的轻量级文本 / 代码编辑器：单窗口、左右双栏、每栏多标签，全部界面由 Go 绘制，不启动 WebView。

## 背景

- 仓库 `helloxz/zpad`，分支 `main`，Module `github.com/helloxz/zpad`
- 应用名 ZPad，Bundle ID `com.xiaoz.zpad`，许可证 AGPL-3.0
- macOS / Windows 已发布，Linux 提供 deb / rpm / tar.gz
- 版本只写在 `mygo.json`；官网 <https://zpad.xphub.dev>

## 技术栈

- Go 1.27.4（`/usr/local/go`）；MyGo v0.10.0（`github.com/egoist/mygo`，用 `go tool mygo` 固定版本）
- UI：MyGo `package ui` 原生绘制；编辑器：`mygo/ui/editor`
- 编码：`golang.org/x/text`；字体：内置 JetBrains Mono（OFL）
- 国际化：`locales/en.json` + `locales/zh-Hans.json` 生成 `locales/catalog_gen.go`

常用命令：

```sh
go tool mygo dev          # 开发版，改 .go 自动重启
go test ./... && go vet ./... && go build ./...
go tool mygo generate     # 改了 locales/*.json 后生成 catalog_gen.go
go tool mygo build        # 打包
```

## 约束

- 不引入 WebView。
- `editor.Buffer` 只接受 UTF-8，编码转换只走 `internal/encoding`。
- UI / App 状态只在主线程改；文件 IO 与原生对话框在 goroutine，完成后 `win.Update` 回主线程。
- 同一文件跨栏共享 `*Document`，每个标签各自持有 `editor.State`。
- 保存用原子写，保留原文件权限、编码、BOM、行尾。
- 超过 20 MB 或 50 万行的文件只读打开。
- 用户可见文案必须进两个 catalog 并重新 generate，禁止硬编码。
- 主题颜色只走 `internal/theme`，不在 view 里写死色值。
- 版本只从 `mygo.json` 读，代码里不再写第二份；发布产物沿用 `mygo build` 默认命名。
- 本文件只写背景、约束、技术栈、目录结构；功能行为、实现记录、进度、测试清单写 `docs/` 或 `CHANGELOG.md`。

## 目录结构

```
main.go             入口：窗口、设置、主题源、打开文件、退出
internal/app/       UI 与业务：窗口、分栏、标签、编辑、预览、搜索、草稿、片段、文本对比
internal/<domain>/  各领域包：encoding、files、settings、theme、session、draft、vim、snippets、texttools、caseconv、atomicfile，纯逻辑不含 UI
locales/            中英文文案与生成的 catalog
fonts/              内置字体与许可
docs/               需求、里程碑、实现记录与签收
website/            Hugo 官网
```

## 开发流程

- 写 MyGo 代码前先加载 `mygo` skill（`.opencode/skills/mygo/SKILL.md`）。
- 先读 `docs/README.md` 与对应文档，实现后补「实现记录」。
- UI 变更优先用 `ui.Tester`；`CHANGELOG.md` 只写简短的用户可见变化。
