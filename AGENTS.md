# AGENTS.md

ZPad 是一个基于 **MyGo 原生 UI** 的轻量级文本 / 代码编辑器。本文件给后续接手本项目的 agent 提供核心上下文，详细需求与里程碑见 `docs/`。

## 项目概况

- 仓库：`https://github.com/helloxz/zpad.git`，分支 `main`
- Module：`github.com/helloxz/zpad`
- 应用名：**ZPad**
- Bundle ID：**`com.xiaoz.zpad`**
- 版本：`0.5.0`
- 许可证：**AGPL-3.0**（对外分发前确认源码义务）
- 平台：**macOS 与 Windows 已发布；Linux 自 0.4.1 起提供打包**（deb / rpm / tar.gz + `install.sh`）
- 当前状态：**0.4.1 已发布**（macOS / Windows / Linux），官网 <https://zpad.xphub.dev> 已上线

## 技术栈

- Go **1.27.4**，官网 tarball 安装，`/usr/local/go`，`GOROOT=/usr/local/go`
- MyGo **v0.10.0**：`github.com/egoist/mygo`
- UI：`package ui` 原生绘制，**不使用 WebView**
- 编辑器：`github.com/egoist/mygo/ui/editor`
- 编码：`golang.org/x/text`（纯 Go，无 cgo）
- 字体：内置 JetBrains Mono（OFL），`ui.RegisterFont` + `ui.FontFamilies`
- 国际化：MyGo catalogs，`locales/en.json` + `locales/zh-Hans.json`，生成 `locales/catalog_gen.go`
- CLI：项目内固定为 tool，使用 `go tool mygo`

常用命令：

```sh
go tool mygo dev     # 开发版，改 .go 自动重启
go test ./...        # 单元测试 + ui.Tester 视图测试
go vet ./...
go tool mygo generate  # 改了 locales/*.json 后重新生成 catalog_gen.go
go tool mygo build   # 打包 macOS .app / .dmg
cd website && hugo server   # 官网，http://localhost:1313/
ZPAD_SHOTS=/tmp/shots go test ./internal/app -run TestSiteScreenshots   # 重截官网截图
```

产物沿用 `mygo build` 的默认命名：`build/darwin-arm64/ZPad.app`、`zpad-0.5.0-arm64.dmg`、`zpad-0.5.0-darwin-arm64.tar.gz`、`update-darwin-arm64.json`；Windows 为 `zpad-setup-0.5.0-amd64.exe`、`zpad-0.5.0-windows-amd64.tar.gz`、`update-windows-amd64.json`，CI 额外打一个 `zpad-0.5.0-windows-amd64-portable.zip`（内容平铺，解压即得 `ZPad.exe`）；Linux 为 `zpad_0.5.0_amd64.deb`、`zpad-0.5.0-1.x86_64.rpm`、`zpad-0.5.0-linux-amd64.tar.gz`、`update-linux-amd64.json`、`install.sh`（`linux.formats` 里没有 `pkgbuild`，所以不生成 PKGBUILD）。不要重命名这些文件（见下）。

## 产品需求（一期）

- 单窗口；侧栏双 Tab（打开的文件 / 工作区，默认关闭，打开文件夹时展开并切到工作区）；内容区左右双栏；底部状态栏。
- 每栏多标签；同一文件跨栏共享 `Buffer`，光标 / 滚动独立。
- 编辑 HTML、CSS、JS、JSON、Markdown、纯文本。
- Markdown 原生预览：标签内切换源码 / 预览（`⌘⇧V`），状态栏语言旁、标签右键菜单与编辑器正文右键菜单都可进入，`Esc`（焦点在窗口任意位置）或预览标签上的眼睛返回源码；新建的未命名文件在状态栏选择 Markdown 后同样可以预览；分屏可源码 + 预览共享同一 `Buffer`。
- 状态栏路径旁可调用系统文件管理器定位当前激活标签的文件；未命名文件不显示该按钮。
- 查找替换（内建 FindBar）、大小写转换（大写 / 小写 / 首字母大写 / 反转，多选区）。
- 跨文件搜索：`⌘⇧F` 在居中浮层中搜索两个分栏里所有已打开标签的文本（含未保存内容与未命名文件）。输入时不自动搜索，按 `↵` 或点击右侧搜索按钮才搜索；结果按文件分组，`↑`/`↓` 选择，`↵` 打开选中项。关闭面板保留关键词和结果，输入框末尾的 X 清空后才丢弃；`Esc` 只关闭面板。只搜内存中的标签，不读未打开的磁盘文件。
- 代码片段：工具栏片段按钮（`⌥⌘S`）打开居中面板，一套视图做完所有事——左栏是分组（首行「全部」）与各自命中数，右栏是选中分组或全部的片段，`↑`/`↓` 选择、`↵` 或点击插入到当前激活标签的光标处（有选区替换，多光标各插一份，一步撤销，多行跟随行缩进）。面板里直接新建 / 编辑 / 删除片段；分组的新建 / 重命名 / 删除在左栏表头（「全部」时后两项置灰）；新建时若有选中的文本，正文与名称直接取它。整库最多 100 条、单条正文 8 KB、名称最多 40 个字符（控制字符剥掉、空白折叠），存于用户数据目录的 `snippets.json`，没有分组时自动建 `default`；插入按原样写入（末尾换行由片段自己带，行尾插入时不额外留空行）；只读、Markdown 预览与 Vim 模式下不插入。
- 编码：UTF-8、UTF-8 BOM、UTF-16 LE/BE、GB18030；自动检测 +「重新以编码打开 / 以编码保存」。
- 换行符：LF / CRLF / CR，保存保留原行尾。
- 空白字符：设置 → 编辑器可选「显示空白字符」为仅选区（默认）/ 关闭 / 全部 / 仅行尾；空格显示为圆点、Tab 显示为箭头，颜色随主题，全局生效。
- Vim 模式（默认关闭）：设置 → 编辑器开启后所有标签进入模态编辑，Normal / Insert / Visual / Visual-line，支持常用移动、数量前缀、`d/c/y` 操作符、文本对象、`u` / `Ctrl+r`、`/` 搜索与最小 `:` 命令；中英文输入法随模式切换，只读标签不修改；不含宏、marks、多寄存器、`.` 重复与 visual block。
- 主题：ZPad 亮色 / ZPad 暗色 / 跟随系统保持不变，另有 18 套命名主题（One Dark、Dracula、Nord、Tokyo Night、Catppuccin Mocha、Monokai、Gruvbox Dark、Ink Night、GitHub Light、One Light、Solarized Light、Catppuccin Latte、Gruvbox Light、Rosé Pine Dawn、Ayu Light、Everforest Light、Celadon、Ink Wash），设置外观里用下拉选择，名称后标注浅色 / 深色；中英双语（M3）；内置 JetBrains Mono + 系统等宽字体（M3）。
- 手动保存 + 未保存标记；关闭标签提示；关闭窗口 / 退出汇总提示「保存全部 / 不保存 / 取消」。
- 保留未保存内容（默认开启）：设置 → 工作区开启后，窗口关闭 / 退出不再提示保存，未保存的标签（含未命名文件）写到用户数据目录的草稿区，下次启动恢复并仍带未保存标记；断电、崩溃、强杀同样可恢复，最多丢约 5 秒输入。关闭单个标签仍提示「保存 / 不保存 / 取消」，选不保存即删草稿。
- 会话恢复：工作区、分屏、标签、活动标签、滚动、侧栏；窗口位置由 MyGo `StateKey: "main"` 管理。
- 侧栏「打开的文件」汇总两栏文档、去重，行内只显示文件名，悬停时在行右侧浮出完整路径来区分同名文件；「工作区」保留文件树。两个 tab 共用的 ⋯ 菜单只在工作区 tab 显示「刷新 / 显示隐藏文件」，「打开的文件」tab 只剩「关闭侧栏」。
- 文件树单击打开文件、单击展开/收起文件夹；标签可拖到另一栏，也可在栏内拖拽排序（拖动时标签栏显示插入位置），点击分屏按钮按内容区 50/50 拆分。
- 侧栏工作区里的文件可拖到任意一栏打开；关闭某栏最后一个标签、或拖走最后一个标签时自动收起分屏。
- 文件树右键菜单：在副窗口打开、重命名、删除；重命名同步打开标签的路径，删除前拦截未保存修改。
- 标签右键菜单：关闭当前 / 左侧 / 右侧、移动到主 / 副窗口、复制到主 / 副窗口；已保存文件另有「在文件管理器中显示」，HTML 另有「在浏览器中打开」，Markdown 另有「预览 Markdown / 返回源码」；右键不切换活动标签，未分屏且只有一个标签时禁止移动到副窗口；批量关闭未保存文档只弹一次汇总提示。
- 「打开的文件」列表对已保存文件提供同样的文件操作；弹窗菜单在窗口内用主题绘制，顶部菜单栏保持原生。
- 文件关联、拖拽打开；文件关联在 M4 完成。
- 自动更新：应用从 GitHub Releases（`helloxz/zpad`）检查新版本，原生更新窗口显示发布说明并安装；更新包用 Ed25519 签名校验；菜单「检查更新…」可手动触发。开发版不能更新。

非目标：LSP、调试器、Git 集成、插件系统、云同步、HTML 内嵌预览、移动端。HTML 文件可从状态栏或标签右键菜单（或 `⌘⇧B`）在系统浏览器中打开；Markdown 原生预览已实现，工作区文件搜索与预览滚动同步放二期。

## 代码结构

```
main.go                    入口：设置、主题源、窗口、OnOpenFile、OnBeforeQuit
internal/app/
  app.go                   状态、pane/tab、分屏、主题、窗口标题
  document.go              文档加载 / 保存、编码、二进制与大小拦截
  preview.go               Markdown 预览、链接、预览选项与大小上限
  preview_image.go         预览图片：本地文件、data: 与远程下载、解码、缓存
  search.go                跨文件搜索：扫描规则、结果模型、后台扫描与跳转
  search_ui.go             跨文件搜索的居中浮层与结果列表
  actions.go               打开 / 保存 / 关闭标签 / 编码 / 换行 / 大小写
  close.go                 关闭 / 退出协调、脏文档、保存全部
  session.go               会话保存与恢复
  drafts.go                未保存内容的草稿：写入节奏、恢复、生命周期
  fileops.go               文件树的新建 / 重命名 / 删除 / 在副窗口打开
  fileops_ui.go            新建与重命名对话框
  view.go                  工具栏、侧栏、双栏、标签、状态栏、查找栏样式
  about_ui.go              关于对话框（版本、许可证、GitHub 仓库）
  settings_ui.go           设置面板、字体列表、语言切换
  menu.go                  应用菜单
  case.go                  多选区大小写转换
  selections.go            多选区替换：偏移重算与选区恢复
  tools.go                 文本工具：Base64 / MD5 / JSON 与「工具」菜单
  snippets.go              代码片段：状态、行模型、插入与增删改
  snippets_ui.go           代码片段：插入面板、管理视图与编辑弹窗
fonts.go                   嵌入并注册 JetBrains Mono
fonts/                     JetBrains Mono TTF + OFL
locales/                   中英文 catalogs + 生成的 catalog_gen.go
internal/encoding/         编码检测 / 解码 / 编码 / 往返校验
internal/files/            工作区目录树
internal/caseconv/         大小写转换
internal/settings/         设置模型、JSON 持久化
internal/texttools/        Base64、MD5、JSON 格式化与压缩
internal/session/          会话模型、原子读写
internal/theme/            主题注册表：默认亮 / 暗 + 命名主题（UI 与编辑器 token）
internal/atomicfile/       原子写文件
internal/draft/            未保存内容的草稿存储：索引、写入、删除、清空
internal/vim/              Vim 模式：模式、motion、operator、文本对象
internal/snippets/         代码片段：模型、校验、原子读写
docs/                      需求、里程碑、实现记录与签收
website/                   官网（Hugo）：首页、下载页、双语、发布数据
```

关键实现约定：

- `editor.Buffer` **只接受 UTF-8**；所有文件编码转换必须走 `internal/encoding`。
- `Document` 不保存文件原始字节：打开期间内存里只有 rope 一份文本（约 1.1 倍文件大小）；外部修改检测比 `Size` + `ModTime`，保存从 `Buffer` 重新编码，别为了「保险」再存一份原文。
- 诊断日志写到用户数据目录的 `zpad.log`（1 MB 轮转，`ZPAD_LOG=off` 关闭，`ZPAD_LOG=<路径>` 换位置）。MyGo 的渲染器日志（Windows 上是否回退 WARP）只有这里能看到；慢帧统计 `app.NoteFrame` 每帧零分配，正常运行时保持安静。
- 超过 50 万行的文件不给 `Buffer` 设语言（高亮会拖慢滚动），`Document.Language` 只用于状态栏；改语言一律走 `Document.setLanguage`，不要直接调 `Buffer.SetLanguage`。
- 同一文件在左右栏时复用同一个 `*Document`；每个标签各自持有 `editor.State`。
- 所有 UI / App 状态修改在主线程；文件读写与原生对话框在 goroutine，完成后用 `win.Update` 回到主线程。
- 脏文档按 `*Document` 去重；关闭协调器用 `closePromptOpen` / `allowClose` / `allowQuit` 防止重复弹窗。
- 保存使用原子写；保留原文件权限、编码、BOM 和行尾。
- 会话只保存有路径的文件；未命名文件不持久化。
- 保留未保存内容（`KeepUnsaved`，默认开）用 `<用户数据目录>/drafts/`：`index.json` 加每个文档一个 UTF-8 文件（真实换行符），目录 0700、文件 0600。`session.json` 只存草稿 id 与未命名编号，正文不进会话。写盘必须走 `Buffer.Snapshot().WriteTo` + `atomicfile.WriteTo` 流式落盘，不要 `Buffer.String()` 或把全文塞进 JSON（会多一份拷贝、顶高堆水位）。节奏：停止输入 1 秒后写，连续输入最多 5 秒一次，每秒扫描一次（`mygo.RunOnMain`，不重绘）。草稿只在设置开启时写；关闭开关即清空 `drafts/`，退出回到原来的保存提示。
- 草稿生命周期收口：`saveTab` / `saveDocsForClose` / `saveDirtySync` 写盘成功、文档变干净、关闭最后一个标签、删除文件时 `forgetDraft`；重命名走 `draftMoved`（只改索引路径）；启动恢复不删草稿，等重新写出或保存 / 丢弃后再删。每个文档有一个 epoch，保存 / 丢弃 / 重命名时递增，写盘 goroutine 在存储锁内检查，过期写入直接跳过，防止「保存后旧副本又被写回来」。
- 启动恢复先按会话标签的草稿 id、再按路径匹配；没有对应标签的草稿追加到左栏。草稿文本与磁盘相同（崩溃前已保存）就丢弃；不同则覆盖为草稿并沿用草稿里的 size/modtime，让现有外部修改检查在聚焦时提示；文件已被删除时直接从草稿建文档，保存走「重新创建」提示。未命名编号接着最大编号继续。
- 单实例用 `mygo.App.RequestSingleInstanceLock()`（`main` 开头）+ `mygo.App.OnSecondInstance(a.HandleSecondInstance)`：第二进程立即退出，第一个实例前置窗口后打开传来的文件 / 文件夹。参数由应用自己解析（MyGo 只转发 `fileAssociations` 里登记的扩展名），`openPaths` 用 `beginLoading` / `endLoading` 的「正在读取」集合保证同一路径只读一次。
- 版本只在 `mygo.json`：`main` 里 `app.SetVersion(mygo.App.Version())` 注入，状态栏与关于对话框读 `app.Version`；不要在 Go 代码里再写一份版本号（`go test` / `go run` 下显示 `dev`）。
- 自动更新用 MyGo 内置链路：`mygo.json` 的 `updates` 提供 Ed25519 公钥与 GitHub 仓库；`main` 调用 `mygo.Use(native.New(updater.Options{Icon: appIconPNG}))`，不要换成 WebView 版插件。菜单项用 `locales.CheckForUpdates()` + `updater.CheckForUpdates()`，不要用 `updater.MenuItem()`：后者在未注册插件的测试里会 panic，文案也不走本项目 catalogs。
- 发布产物一律沿用 `mygo build` 的默认命名，不重命名：`zpad-<版本>-<arch>.dmg`、`zpad-setup-<版本>-<arch>.exe`、`zpad-<版本>-<target>.tar.gz`、`.delta`、`update-<target>.json`。CI 只额外产生 `zpad-<版本>-windows-amd64-portable.zip`，内容平铺在根目录（解压即得 `ZPad.exe`，不套文件夹），CI 会校验 `ZPad.exe` 在归档根目录。发布时用 `mygo build -upload`，它会先传安装包与更新包、最后传 manifest，release 必须发布为 latest（`releases/latest/download/` 才能读到）。
- 更新私钥只在 `~/Library/Application Support/mygo/update-keys/mygo-update.key` 和 GitHub secret `MYGO_UPDATER_PRIVATE_KEY`；不要提交、不要打印。私钥丢失后，已安装的用户无法再收到更新。发布构建缺私钥会跳过签名，只适合本地打包。
- 强制外观用 `mygo.Theme.SetSource`，UI 主题用 `c.SetTheme` / `ui.Themed`。
- 失败提示分两类：用户动作的一次性结果（打开 / 重新打开 / 对话框失败）用 `SetErrorToast(title, detail)`，走 toast 队列，6 秒自动消失、悬停暂停、Esc 关闭；`SetError` 只用于文档还开着、需要用户处理的保存 / 关闭失败（状态栏红字，下一次成功打开时清零）。不要把一次性结果写进 `errMsg`，否则它会一直留在状态栏。
- 弹窗菜单统一 `c.SetMenuPresentation(ui.PopoverMenus)` 在窗口内绘制；顶部系统菜单栏保持原生。侧栏树菜单用 `PopoverMenu` + `sidebarMenuOptions`，避免继承文件树的选中样式。
- 菜单项手型光标在 `PopoverMenuOptions` / `PopoverMenuButtonOptions` 的 `Item` 中设置；MyGo 内部的编辑器右键与输入框编辑菜单无法从应用层设置，仍是箭头。
- 内存：`main` 开头 `debug.SetGCPercent(50)`，把 Go 堆高水位从 24–28 MB 压到 14 MB；About 图标按需解码。不要再加 `FreeOSMemory` 定时器或 `SetMemoryLimit`：实测前者在 macOS 上只多释放约 3 MB 且 Activity Monitor 不变，后者在 20 MB 大文件下可能触发连续 GC。新增内存相关改动时同时量一遍 Activity Monitor 的物理占用。
- 文件树行同时承担拖拽与单击打开；整行（缩进、箭头、文件名、左右空白）都由 `treeRow` 自己绘制，`ui.OutlineState.Prefix` 置空以去掉 Outline 自带的缩进和箭头，因此行内任意位置都能命中同一个 Row。点击处理用 `Clicked()` 并手动 `selectTreePath`：整行是应用自己的元素，Outline 的行收不到点击，选中必须自己同步。
- 标签栏本身是拖拽的放置目标：`ScrollHorizontal` 的 `ui.DragOver` / `ui.Drop` 接住 `paneDrag`，用 `area.PointerPosition()`（拖拽期间照样更新）加各标签上一帧的 `Bounds()` 算出插入索引与竖线位置，再交给 `moveTabAt`。落在同一栏即重排（活动标签跟着原标签走，不跟着指针换），落在另一栏的标签栏按位置插入，落在编辑区仍追加到末尾。分屏分隔线有一圈抓取区域盖在两侧栏上，落在抓取区里的拖拽没有目标，属既有行为。
- macOS 的工具栏就是标题栏：窗口用 `TitleBarHidden`，`toolbar()` 从 `Context.TitleBar()` 取红绿灯占位，左右用 `10+Left` / `10+Right` 留白，空白处 `DragWindow`；全屏时 `TitleBar` 为零。改工具栏高度（45）或字号时同步调 `main.go` 的 `TrafficLightPosition`，否则红绿灯会歪、或占位超出工具栏。
- Windows 的菜单条兼作标题栏：`customMenuBar` 在 `Attach` 里按 `runtime.GOOS` 打开（不要在 `New` 里设，否则 Windows 上的视图测试都会多一行），`MenuBarHeight` 与 `main.go` 的 `TitleBarHeight` 共用；菜单条从 `Context.TitleBar()` 取右侧系统按钮占位，点击用 `mygo.NewMenu(顶层菜单.Submenu).PopupAt(a.win, x, MenuBarHeight)`，在 goroutine 里调用让 `onMain` 把菜单放到帧之间；每次打开都用 `a.menuItems()` 重建菜单项（勾选状态取当前值），不要把已安装的应用菜单 item 再挂进另一个 `Menu`。
- 会话保存标签顺序，标签重排后必须 `saveSession()`，否则重启按旧顺序恢复。
- 官网是独立静态站：源码在 `website/`，Hugo 生成，CF Pages 构建（`root=website`、`output=public`、`HUGO_VERSION=0.170.1`）。版本数据在构建时由 `layouts/partials/release.html` 从 GitHub 最新 release 取（`resources.GetRemote`；失败回退 `website/data/release.json` 快照并 warning），下载页、首屏胶囊、首页「最近更新」和 `_redirects` 都由它生成；发布后在 CF Pages 点 Retry deployment 触发构建，仓库里不再有 bot commit。官网截图由 `internal/app/site_shots_test.go` 的 headless 渲染生成（`ZPAD_SHOTS` 打开），原图在 `website/assets/screenshots`；换应用图标时同时更新 `website/assets/icon.png`（`sips -c 824 824 resources/icon.png`）。
- 不引入 WebView。

## 当前进度

- **M0 项目骨架**：已签收。
- **M1 单栏编辑闭环**：已签收。打开文件 / 文件夹、侧栏、标签、编辑器、保存、编码、换行、大小写、状态栏、菜单、拖拽。
- **M2 左右双栏与会话恢复**：已签收。分屏、跨栏共享 Buffer、会话恢复、关闭 / 退出汇总提示。
- **M3 视觉、设置、字体与中英双语**：已签收。完整主题、设置面板、JetBrains Mono、中英切换、查找栏 / 跳行栏样式。
- **M4 加固、文件关联与 macOS 发布**：0.1.0 已发布。大文件只读、编码边界、外部修改检测、文件关联、正式图标、`.app` / `.dmg`。
- **Markdown 原生预览（后续增补）**：已实现。标签内切换、状态栏 / 预览标签眼睛 / 快捷键 / 右键菜单、分屏源码 + 预览、本地图片、链接、代码高亮；上限约 1 MB / 1 万行；未命名文件在状态栏选择 Markdown 后也可预览。
- **Markdown 远程图片（后续增补）**：已实现。`preview_image.go` 接管 `markdown.Options.LoadImage`：本地路径仍走 `markdown.FileImages`，`data:`（仅 base64）与 `//host/…`、`http`、`https` 由应用自己加载。远程图片先查包级 LRU（32 MB，按解码字节，先进先出提升），再在后台下载：15 秒超时、10 MB 上限（`Content-Length` 加 `io.LimitReader` 双保险）、24 MP 像素预检（挡掉全尺寸解码的瞬时峰值）、并发 4；SVG 走 `ui.ParseSVG`，其它用 `ui.DecodeBitmapScaled` 缩到长边 1600 px（820 DIP 预览宽在 2× 屏刚好 1:1）。`User-Agent` 为 `ZPad/<版本>`。离开预览（`dropPreviewOptions`，两栏都不再显示该文档时）丢掉 `doc.md`，图片随之释放；再次进入预览重建选项，顺带重试失败的图。不改 `Options.Block`，所以预览里的选区与「复制为 Markdown」不受影响。未发布。
- **编辑器右键菜单预览（后续增补）**：已实现。`configureEditor` 把当前标签交给 `editorContextMenu`，在应用项最前面加「预览 Markdown」（超过 1 MB / 1 万行时显示「预览过大」并禁用），只在 `.md` 出现，右侧显示 `⌘⇧V`；标签右键菜单与状态栏按钮不变。未发布。
- **预览 Esc 返回源码（后续增补）**：已实现。`App.View` 注册窗口级 `Esc`：活动标签在预览时退回源码，焦点落在工具栏 / 侧栏 / 标签栏也生效。设置 / 关于 / 重命名 / 新建与跨文件搜索都是 `ui.DialogBase`，弹窗菜单与 toast 用 `OverlayShortcut`，按键路由浮层在窗口之前，先吃掉 `Esc`；源码编辑器自己的 `Esc`（取消选择 / 多光标）在焦点链里，也优先。预览态「返回源码」的 tooltip 显示 `Esc / ⌘⇧V`。未发布。
- **自动更新（0.3.0 起）**：已实现。GitHub Releases + Ed25519 签名 + 原生更新窗口 + 增量包；菜单「检查更新…」；发布流程见 `docs/06-auto-update.md`。0.3.1 是第一个可自更新的版本，产物改用 `mygo` 默认命名。
- **0.3.2**：首页空状态的快捷键提示按平台显示（macOS 为 `⌘`，Windows / Linux 为 `Ctrl+`），Markdown 预览与浏览器按钮的 tooltip 共用 `shortcutText`。已发布。
- **跨文件搜索（后续增补）**：已实现。`⌘⇧F` 居中浮层搜索所有已打开标签，按文件分组，支持区分大小写、全词、正则。输入和切换选项后不自动搜索，按 `↵` 或点击右侧搜索按钮才扫描；结果是最新时 `↵` 打开选中项，`↑`/`↓` 选择。关闭面板保留关键词和结果，便于点开另一个结果；输入框末尾的 X 才清空；`Esc` 只关闭。关闭标签时同步剔除该文档的结果，避免结果引用住它的 `Buffer`。后台读 `Buffer` 快照扫描，结果上限每文件 200、全局 2000；预览标签跳转前切回源码；只覆盖打开标签，不读工作区磁盘文件。未发布。
- **标签拖拽排序（后续增补）**：已实现。栏内拖动改变顺序，标签栏显示插入竖线；拖到另一栏的标签栏按落点位置插入，拖到编辑区仍追加到末尾；活动标签不因拖拽切换；文件拖到标签栏沿用追加并提示。未发布。
- **文件树排序（后续增补）**：已实现。文件名按自然顺序比较：连续数字按数值大小（`3月` 在 `10月` 前、`file2` 在 `file10` 前），字母忽略大小写、仅大小写不同的名称固定小写在前；目录仍排在文件前。比较器在 `internal/files/sort.go`，不提供配置。
- **工作区新建（后续增补）**：已实现。工作区目录树的空白处右键「新建文件 / 新建文件夹」建在工作区根目录；文件夹行的右键菜单顶部也有同样两项，建在该文件夹里。命名沿用重命名弹窗，空名、含 `/` 或 `\`、同名文件/文件夹、或同名文档已打开都会在弹窗内拦截；写盘在 goroutine，成功后刷新目录树并展开父目录，新建的文件在活动栏打开，新建的文件夹只展开不建标签。未发布。
- **Linux 打包（后续增补）**：已接入 `mygo build -platform linux/amd64`，产出 `zpad_<版本>_amd64.deb`、`zpad-<版本>-1.x86_64.rpm`、`zpad-<版本>-linux-amd64.tar.gz`、`update-linux-amd64.json` 与 `install.sh`。`mygo.json` 的 `linux` 段：`webview: false`（不声明 WebKitGTK，只依赖 GTK3）、`formats` 只留 `deb` 与 `rpm`、命令名 `zpad`、维护者与分类。CI 新增 `Linux` 作业，`site` 作业改为等三个平台都上传完再补官网数据。0.4.1 起发布；官网下载页给 deb（主按钮）与 rpm 两个按钮，tar.gz 只在安装说明里通过 `install.sh` 一行命令出现。
- **macOS 窗口最小化（后续增补）**：已实现。应用自带菜单栏原先没有 Window 菜单，`⌘M` 按下去没有反应；现在 macOS 的菜单栏多出「窗口 → 最小化」（`⌘M`），`⌘W` 仍只关闭标签，关窗行为不变。未发布。
- **工具栏入口（后续增补）**：已实现。左侧工具栏末尾新增信息按钮，打开关于对话框；右侧工具栏行首新增检查更新按钮，与菜单里的「检查更新…」共用 `App.checkUpdates`。未发布。
- **命名主题（后续增补）**：已实现。ZPad 亮色 / ZPad 暗色的色值与 token 保持不变，新增 One Dark、Dracula、Nord、Tokyo Night、Catppuccin Mocha、Monokai、Gruvbox Dark、Ink Night、GitHub Light、One Light、Solarized Light、Catppuccin Latte、Gruvbox Light、Rosé Pine Dawn、Ayu Light、Everforest Light、Celadon、Ink Wash。设置 → 外观用下拉选择，按「默认主题 → 深色主题 → 浅色主题」分组（默认组含跟随系统与 ZPad 亮 / 暗色），命名主题的浅色 / 深色标注在行右侧对齐，下拉超过约 320px 后内部滚动并立即生效；设置面板为 700×640 的卡片分组，所有设置下拉选项悬停显示小手；主题同时覆盖侧栏、标签栏、状态栏、弹窗与编辑器 token。命名主题固定明暗，原生标题栏 / 菜单 / 对话框跟随其基准；「跟随系统」只在 ZPad 亮色 / ZPad 暗色之间切换。原生菜单保留三档 +「更多主题…」，工具栏不设主题按钮。未发布。
- **macOS 标题栏合并（后续增补）**：已实现。窗口在 macOS 上用 `TitleBarHidden` 隐藏原生标题栏、保留红绿灯，并用 `TrafficLightPosition{20,14}` 把它们放进 45 DIP 工具栏左侧；`toolbar` 按 `Context.TitleBar()` 的 `Left` / `Right` 留白，空白处 `DragWindow` 拖动窗口、双击缩放。全屏时 `TitleBar` 归零、留白回到 10，不设拖动。Windows / Linux 分支不变，仍是原生标题栏。未发布。
- **Windows 自定义顶部菜单（后续增补）**：已实现。Windows 窗口隐藏原生标题栏，用 32 DIP 菜单条兼作标题栏：左边「文件 / 编辑 / 显示 / 编码 / 帮助」，点击用 `mygo.Menu.PopupAt` 弹应用菜单的顶层子菜单，右上角让出 138 DIP 给系统按钮；工具栏回到第二行、保留自己的 10 DIP 左右内边距。菜单条空白处拖动窗口、双击最大化，按钮 `.KeepFocus()` 保持编辑器焦点（Edit 的 Undo / Cut / Copy / Paste 才作用在文本上）。每次打开重建菜单项，勾选状态取当前设置；原生应用菜单仍保留，隐藏标题栏下 `Ctrl+…` 快捷键照常工作，Alt/F10 仍能弹出原生菜单。macOS 仍是工具栏即标题栏，Linux 不变。未发布。
- **保留未保存内容（后续增补）**：已实现。设置 → 工作区新增开关，0.6.0 起默认开启（此前的设置文件保留用户存下的值）；开启后窗口关闭 / 退出不弹保存提示，未保存的标签（含未命名文件）写到 `<用户数据目录>/drafts/`（`index.json` + 每文档一个 UTF-8 文件，目录 0700），停止输入 1 秒后写、连续输入最多 5 秒一次；启动时按草稿 id / 路径并入会话，没有对应标签的草稿追加到左栏，恢复后仍带未保存标记；文件已保存则丢弃旧草稿，崩溃期间文件被改则保留草稿里的 stamp 交给外部修改检查；保存、变干净、关闭最后一个标签、删除文件即删草稿，重命名只改草稿路径；关闭开关清空草稿。断电 / 强杀最多丢约 5 秒输入；不恢复光标、选区和撤销栈。未发布。
- **新建文件的语言选择（后续增补）**：已实现。未命名文档的状态栏语言标签变成下拉（纯文本 + 35 种语言，常用的一批排在前面、其余按名称排序，两组之间一条分隔线，弹层最高 320 滚动），选择后立即按该语言高亮，保存对话框默认名与「名字不写后缀」都补上语言的第一个扩展名（JSON → `.json`）。文件一旦有路径，语言只由文件名决定，标签变回不可点的文字（tooltip 提示改名后自动生效），改后缀（侧栏重命名 / 另存为）仍走既有 `setLanguage` 重新高亮；保存时存成别的扩展名就以扩展名为准。语言列表只收有可保存扩展名的语言，Makefile / Dockerfile 除外。未命名文档的选择随会话 `TabState.Language` 与草稿 `Entry.Language` 跨重启保留；选语言本身不标记文档为已修改。未发布。
- **文本工具（后续增补）**：已实现。右键一级「工具」子菜单与工具栏扳手按钮共用一套菜单：Base64 编码 / 解码、MD5 四种十六进制、JSON 格式化 / 压缩；大小写转换不重复放进「工具」，右键用 MyGo 自带的「转换」，应用菜单「编辑 → 大小写转换」保留（含反转大小写）；系统菜单栏保持原样。工具只处理选中文本，单次选区上限 512 KB；多选区先全部转换再一次性写入，一步撤销。Base64 解码忽略空白、兼容无填充与 URL-safe，解码结果必须是 UTF-8；JSON 保持成员顺序、按选区所在行对齐缩进，深度上限 128、输出上限 8 MB（`jsontext` 流式写入限长 writer）。右键作用于被右键的标签，不依赖活动栏。未发布。
- **标签中键关闭（后续增补）**：已实现。在标签上按下鼠标中键立即关闭该标签，不激活它、不移动焦点。MyGo 的 `Clicked` 只认主键，中键既不产生点击也没有 `MiddleClicked`，所以 `tabItem` 用 `HandleInput` 自己接 `InputPointerDown` 的 `Button == 2` 并吞掉这次按压，其它键与左右键原样放行（左键点击 / 拖拽排序 / 右键菜单不变）。关闭仍走 `CloseTab`：未保存照常提示，关掉某栏最后一个标签照常收起分屏，草稿与搜索结果的清理不变。按在标签栏空白处、`+` 号或标签列表按钮上不触发；侧栏「打开的文件」不支持中键；不加额外 tooltip。macOS 触控板默认没有中键，需三键鼠标。未发布。
- **显示空白字符（后续增补）**：已实现。设置 → 编辑器新增「显示空白字符」下拉，可选仅选区（默认）/ 关闭 / 全部 / 仅行尾，存储在 `Settings.Whitespace`（`selection | none | all | trailing`）。渲染直接用 MyGo 的 `editor.State.Whitespace`：空格画 2px 圆点、Tab 画箭头，颜色走主题的 `editor.Theme.Whitespace`（默认亮暗用 `TextMuted`，命名主题用 muted 45%），不新增主题 token，也不改 `configureEditor` 之外的渲染路径。`configureEditor` 每帧按设置赋值，切换立即生效，无需重开标签；默认「仅选区」与 VS Code / Zed / Sublime 一致，升级后观感不变。未发布。
- **Vim 模式（后续增补）**：已实现。`Settings.VimMode`（`vimMode`，默认 false）与设置 → 编辑器开关；每个标签一个 `internal/vim.Session`（`Tab.vim`），关闭开关时置空。用 `editor.State.OnInput` 在编辑器前接键：Normal / Visual 关输入法、光标方块（pending 下划线），Insert 恢复输入法、光标竖线；状态栏显示 `NORMAL` / `INSERT` / `VISUAL` / `V-LINE`。支持 `h j k l w b e W B E 0 ^ $ gg G { }`、数量前缀、`i a I A o O`、`x X D C s S J ~`、`d c y` 加移动与文本对象（`iw/aw/i"/a"/i(/a(…`、`ip/ap`）、`yy/Y/p/P`、`u`/`Ctrl+r`、`>>`/`<<`、`%`、`v`/`V` 可视模式；`/` 走内建 FindBar 后关闭，`n`/`N` 用 Session 记录的当前匹配范围避免重复命中；`:w`、`:q`、`:wq`、`:x`、`:行号`。Esc 在编辑器聚焦时由窗口级入口先交给 Vim，命令行为窗口底部浮层；Markdown 预览、设置、搜索、重命名、新建、关于弹窗仍优先。只读标签不改动，编辑全部走 `Buffer.Edit`，不复制整份文本。不含宏、marks、多寄存器、`.` 重复、visual block、`f/t`、`:s` / `:set`，侧栏 / 文件树 / 标签不做全局 hjkl。未发布。
- **代码片段（后续增补）**：已实现。`internal/snippets` 存 `<用户数据目录>/snippets.json`（`version` + 分组与片段，稳定 `id`，CRLF 归一为 LF，损坏落 `.bak`，原子写），整库上限 100 条、单条正文 8 KB，超限在弹窗里被拦下。工具栏扳手右边的片段按钮与 `⌥⌘S`（Windows / Linux `Ctrl+Alt+S`）打开同一面板，编辑菜单「编辑 → 代码片段」另有插入片段 / 用当前选区新建。面板与跨文件搜索同一套 `DialogBase` 卡片，但只有一套视图：左栏 160 DIP 的分组列（「全部」是 `snippetsGroup` 的 -1，命中数来自 `snippetCounts`，0 命中置灰；表头是「✎ 🗑 ＋」三个 22px 图标，前两个在「全部」视图下置灰），右栏是 `ui.List` 虚拟化的片段列表（`↑`/`↓` 选择、`↵` 或点击插入，行尾 ✎ 编辑 / 🗑 删除，在「全部」时每行右侧标出来源分组）。两栏表头共用 `snippetHeaderHeight` 常量，内容框顶边始终对齐；行内没有额外控件，分组行与「全部」行等宽。搜索同时匹配分组名、片段名与正文（`foldContains` 逐字节 `EqualFold`，不复制正文）；某分组里 0 命中但别处有时，空状态给「其它分组还有 N 个」+「查看全部」。新建片段写进当前分组，在「全部」时写进选中那条片段的组，一条都没有时建 `default`（不再拿当前文件的语言起名）；若有选中文本，正文与名称直接取它。插入不走 `State.Insert`（粘贴路径可能把文本当整行粘贴放到上一行），而是每个选区从后往前 `Buffer.Edit`、`Transact` 合成一步撤销、光标落在插入文本之后，多行正文复用 `alignToLine` 跟随行缩进；只读、Markdown 预览、Vim 模式下拒绝并给 toast，面板不关。分组与片段的删除都用原生确认框。名称上限 40 字按 rune 计（`MaxNameRunes`），加载走 `normalizeName` 只折叠空白、剥控制字符，**不因超长丢弃**（`cleanName` 只在新增 / 改名时校验），手写文件里的超长名字留到编辑时再改。插入不再裁掉正文末尾的换行：`alignSnippet` 在正文以换行结尾时去掉末尾缩进，光标落在行尾时连该行的换行一起替换，所以块状片段不会多出空行。编辑弹窗的名称只有 placeholder（分组弹窗是「请输入分组名称」）、没有可见标题，正文有「正文」标题，底部是 `名称 n / 40 · 正文 n B / 8.0 KB` 计数，标题下写「所属分组：X」。选择由应用层持有：`refreshSnippetRows` 结尾用 `ListState.Choose` 把选择交给列表，否则 `ui.List` 会按上一帧记住的片段 key 在新行里找不到就写回 -1（切分组、删片段时表现为「什么都没选中」）；点分组 / 删片段 / 保存后把键盘焦点还给搜索框（`snippetsFocus`）。每次打开面板重新读文件，手改的片段能被看到。未发布。
- **官网**：`website/` 的 Hugo 站点（首页、下载页、更新日志），中英双语、亮暗主题；构建时从 GitHub 最新 release 取版本数据，发布后手动重建 CF；线上 <https://zpad.xphub.dev>。
- **官网截图（0.5.0）**：已更新。图库 6 张：分屏与设置（含主题下拉）各亮暗一张，Markdown 预览用 GitHub Light、跨文件搜索用 One Dark；首屏 `hero` 默认 GitHub Light，鼠标悬停由 CSS 淡入 One Dark，不再跟随站点亮暗切换。

测试覆盖：编码、行尾、大小写、跨文件搜索的匹配规则与跳转、文件树、文档读写、分屏合并、跨栏共享、脏文档去重、会话往返、缺失文件跳过、滚动恢复、无修改退出保存会话、设置面板渲染、设置下拉光标、字体过滤、中英文案、编辑器 token 颜色、主题注册表与对比度、主题切换持久化、大文件只读、只读目录保存失败、外部重载、文件关联配置、更新公钥与 CHANGELOG 版本小节、更新菜单项、图标尺寸、标签拖拽重排与跨栏按位置插入、文件树自然排序、工作区空白处与文件夹行的新建菜单、侧栏菜单按 tab 裁剪、草稿写入 / 恢复 / 生命周期（保存、变干净、关闭最后一个标签、重命名、切换开关）、未命名文件跨重启恢复、只读跳过、草稿写入节奏、标题栏占位与工具栏留白、Windows 菜单条布局 / 焦点 / 分屏高度 / 顶层菜单过滤、未命名文档的语言选择与高亮、语言列表只含可保存的扩展名、已保存文件的语言只读、保存名与补后缀、会话 / 草稿里的语言往返、未命名 Markdown 的预览与语言切换收口、文本工具的 Base64 / MD5 / JSON、选区上限、只读跳过与菜单置灰、标签中键关闭（含另一栏）与左键仍能激活、空白字符档位映射与设置下拉、Vim 模式默认关闭 / 模式安装 / 插入与 Esc / 可视删除 / `/` 与 `n` / `:行号` / 开关切换 / 只读 / 输入法切换、`internal/vim` 的移动、数量、操作符、文本对象、粘贴、缩进与行尾、代码片段的存储往返与修复 / 空名与重名 / 100 条与 8 KB 上限 / 光标与多光标插入 / 缩进对齐 / 一步撤销 / 只读与预览与 Vim 拒绝 / 过滤 / 选择回绕 / 弹窗增删改 / 从选区新建（含 default 分组与预填）/ 面板内新建分组 / 「全部」与按分组过滤 / 分组命中数与「其它分组」提示 / 表头 ✎ 打开重命名且「全部」时置灰 / 删除选中分组回落「全部」/ 名称 40 字上限与控制字符剥离 / 加载保留超长名字 / 正文末尾换行不产生空行且不留行尾空白 / 行中插入把后文推到下一行 / 切分组与删片段后仍有选中项 / 点分组后焦点回到搜索框，以及 `internal/snippets` 的原子读写。

## 已知限制 / 后续

- 超过设置阈值（默认 20 MB）或 50 万行的文件以只读方式打开。
- 文本工具按编辑器内部的 LF 计算多行文本的 MD5 / Base64，与磁盘上 CRLF 文件的字节不同；JSON 只接受严格 JSON（注释、尾逗号会报错），不支持排序 / 去重 / 自定义缩进；系统菜单栏没有「工具」入口。
- 代码片段上限固定（100 条 / 单条正文 8 KB / 名称 40 字），没有设置项，也不做导入导出与 VS Code `.code-snippets` 兼容；占位符（`$1` / `${1:name}`）按字面插入；不自动补换行（要换行就在正文末尾自己按回车）；分组与片段只能追加到末尾，没有拖动排序；每次打开面板都重读文件，但在 ZPad 里改了未保存的 `snippets.json` 时，面板里的下一次写入会覆盖手改内容。
- 空载物理占用约 60–75 MB：其中约 28 MB 是 MyGo 的一个全尺寸 Metal drawable；聚焦编辑时 MyGo 保留 3 个（1920×995 屏幕约 85 MB）并伴随数十 MB Metal 缓存，应用层无法控制，需上游支持。
- 文件树删除为永久删除，未接入废纸篓；重命名使用模态框，未做 Finder 式行内编辑。
- 设置修改立即写盘，未做防抖。
- 命名主题固定明暗，不支持导入自定义主题；「跟随系统」只在 ZPad 亮色 / ZPad 暗色之间切换，不做按主题的明暗配对。主题名保持英文。
- 工作区（磁盘）全局搜索、完整文件监听在二期；已实现的跨文件搜索只覆盖当前打开的标签。
- 保留未保存内容（默认开启）不恢复光标、选区和撤销栈；空白未命名标签不写草稿，只靠会话，崩溃时可能少一个空标签；断电瞬间可能丢最后一次副本（最多约 5 秒输入），上一份仍完整；开启后退出不再询问保存，关闭开关会清空草稿。
- Vim 模式默认关闭，只覆盖核心档：没有宏、marks、命名寄存器、`.` 重复、visual block、`f/t`、`*`/`#`、`:s` / `:set`；侧栏、文件树、标签栏和分屏分隔线不做全局 hjkl；`/` 用内建 FindBar 的普通查找，不是 Vim 正则；`n`/`N` 依赖最近一次 `/` 记录的范围；Windows / Linux 上与 `Ctrl+S/O/W/F…` 等应用快捷键冲突时应用优先；`NoInputMethod` 的运行时切换需在 macOS / Windows 中文输入法上真机确认。
- Markdown 预览上限约 1 MB / 1 万行；原始 HTML 块按源码显示，YAML frontmatter 按普通 Markdown 显示，`.mdx` 不启用预览；远程图片按 1600 px 长边 / 10 MB / 2400 万像素 / 15 秒超时加载（并发 4），失败、超限或网站禁止热链时显示 alt 占位块，重新进出预览才会重试，`data:` 只支持 base64；预览期间图片内存随当前文档的图片张数线性增长（1600 px 一张约 7–9 MB），关掉预览或标签后回落到 32 MB 缓存；数学公式显示 TeX 源码、Mermaid 等按代码显示；预览内查找、滚动同步、目录 / TOC 未做；预览模式不写入会话，重启后按源码恢复；宽表格在窄栏内横向滚动（单列上限约 600px）。
- 关闭 / 退出对话框仍需人工点选验证。
- 分隔线宽度在切换分屏 / 关闭时保存，不在拖拽每一帧写盘。
- 编码底层的错误详情可能仍为英文技术信息；主提示已本地化。
- macOS 构建未签名、未公证；正式分发需 Apple Developer 账号。自更新不依赖 Apple 签名，但 ad-hoc 每次构建指纹不同，更新后 TCC 权限可能要重授。
- 自动更新：0.3.0 是首个带更新器的版本，自身收不到更新，0.3.1 起可自更新；标准用户需装在 `~/Applications`，只读 DMG 或包管理器安装的实例 `Updater.Enabled()` 为 false；Windows 装在 `Program Files` 的机器级安装不能自更新；更新窗口不参与主窗口会话。
- Windows 安装包只把 ZPad 登记到「打开方式」，默认应用需用户手动选择；Windows 打包尚未验证。Linux 的 deb / rpm 只在 macOS 上交叉构建过，还没在真实发行版上安装验证；用 `install.sh` 装到 `~/.local` 才能自更新，deb / rpm 装的由包管理器更新。
- UTF-16 无 BOM 检测失败时通过状态栏手动切换编码，没有独立预览选择框。

## 工作流程

- 写 MyGo 相关代码前先加载项目 skill `mygo`（`.opencode/skills/mygo/SKILL.md`），按其中的顺序查 API，不要凭记忆猜。
- 先读 `docs/README.md` 与对应里程碑文档；实现完成后在文档中补「实现记录」，状态改为「待签收」。
- 未获得用户明确签收前，不自行把里程碑标记为「已签收」。
- 修改后至少运行 `go test ./...`、`go vet ./...`、`go build ./...`；涉及打包时运行 `go tool mygo build`。
- UI 变更优先用 `ui.Tester` 做视图测试，必要时生成截图人工检查。
- 用户可见文案必须加到 `locales/en.json` 和 `locales/zh-Hans.json`，然后运行 `go tool mygo generate`，通过 `locales.X()` 使用；不要硬编码。
- `CHANGELOG.md` 只写简短的发布说明：一个版本一节，每条一句话说清用户可感知的变化，不写实现细节、测试过程或逐项罗列；不按「新增 / 修复 / 优化」分子标题，直接一个列表；细节留在 `docs/`，更新窗口直接展示这节，越短越好读。
- 新增字体必须确认 OFL / 商业许可，并把许可文件放进 `fonts/`；许可文本不随应用打包（`resources/` 只放构建期素材，用 `resourcesExclude` 排除，或直接不放进去）。
- 主题颜色优先走 `internal/theme`，不要在 view 里散落硬编码色值。
- 新增命名主题只在 `internal/theme/builtin.go` 加一组种子色，由派生规则补齐悬停、选区、滚动条与 token；不要在 view 里按亮 / 暗写分支。默认亮 / 暗的色值与 token 保持 0.4.1 原样，改动前先看 `theme_test.go` 的固定值。
- 主题 id 由 `theme.Normalize` / `theme.Resolve` 校验：`Settings.Theme` 是唯一真源，`App.Dark` 只表示当前基准；命名主题固定明暗，「跟随系统」只在 ZPad 亮色 / ZPad 暗色之间切换。
- 文件关联写在 `mygo.json` 的 `fileAssociations`；构建后检查 Info.plist 的 `CFBundleDocumentTypes`。`internal/app/textExtensions`（打开对话框筛选器）必须与关联列表一致，`release_test.go` 会校验。
- 发布构建用 `go tool mygo build`，产物在 `build/darwin-arm64/`；当前未签名，正式分发另行处理公证。
- 发布顺序：打 tag 推送后等 CI 把 draft 的 macOS / Windows / Linux 产物传完再 Publish。提前发布会让 `releases/latest/download/update-<target>.json` 短暂 404（自动更新报错）。发布后到 CF Pages → Deployments → Retry deployment 触发一次构建，官网构建时取最新 release；没有 website bot commit，不会和本地冲突。
