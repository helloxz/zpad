# Vim 模式

状态：**已实现，待签收**  
范围：编辑器级、全局开关、核心档（不含宏、多寄存器、marks、`.` 重复、visual block）

## 目标

给 ZPad 加一个可选的 Vim 模式，让习惯 Vim / Zed Vim 的用户可以用模态编辑写笔记和代码，同时不影响默认的非模态编辑体验。

- 设置 → 编辑器 →「Vim 模式（实验）」开关，**默认关闭**；
- 开关是全局的，开启后所有标签的编辑器都进入 Vim 模式；
- 每个标签保留自己的模式、光标和可视选区，分屏两边互不干扰；
- 不进入 WebView，不修改 MyGo，全部由应用层实现。

## 交互

### 模式

- Normal：按键是命令，光标为方块；
- Insert：和普通编辑器一致，中文输入法正常组合；
- Visual / Visual-line：按字符 / 按行选择；
- Normal / Visual 下关闭系统输入法，避免拼音字母被当成中文输入；Insert 下恢复输入法；
- 状态栏在路径前显示 `NORMAL` / `INSERT` / `VISUAL` / `V-LINE`；有待执行的 count 或 operator 时光标改为下划线。

### 支持的键

Normal / Visual 共用：

- 移动：`h j k l`、`w b e`、`W B E`、`0 ^ $`、`gg G`、`{ }`、方向键、Home / End、PageUp / PageDown、Enter；
- 数字前缀：`3j`、`2dd`、`3x` 等；
- 搜索：`/`（命令行输入，按 Enter 执行）、`n`、`N`；
- 命令：`:`（`w`、`q`、`wq`、`x`、`:行号`）。

Normal：

- 进入插入：`i a I A o O`；
- 删除 / 修改：`x X D C s S`、`J`、`~`；
- 复制粘贴：`yy`、`Y`、`p`、`P`；
- 撤销 / 重做：`u`、`Ctrl+r`；
- 操作符：`d c y` 加移动或文本对象，例如 `dw`、`d$`、`dd`、`dj`、`dG`、`dgg`、`cw`、`ciw`、`di"`、`da(`；
- 文本对象：`iw aw iW aW`、`i" a"`、`i' a'`、`` i` a` ``、`i( a(`、`i[ a[`、`i{ a{`、`i< a<`、`ip ap`；
- 缩进：`>>`、`<<`；
- 括号匹配：`%`。

Visual：

- `v` / `V` 切换字符 / 行选择，`o` 交换选区端点；
- `d` / `x` 删除，`y` 复制，`c` 修改，`p` 替换；
- `>` / `<` 缩进；
- `:` / `/` 退出可视并打开命令行。

### 明确不做

- `.` 重复、宏、marks、命名寄存器、visual block；
- `f F t T ; ,`、`r`、`*` / `#`、`gv`、`gi` 等进阶移动；
- `:s`、`:set`、`:e`、`:q!`；
- 侧栏、文件树、标签栏、分屏分隔线的全局 hjkl；
- 预览模式接管按键（Markdown 预览仍用 `Esc` 返回源码）。

## 设计

### 输入钩子

`editor.State.OnInput` 在编辑器自己的按键处理之前收到每个输入事件，返回 `true` 即吞掉。Vim 模式只在这个钩子上工作：

- 可打印字符走 `InputText`，作为命令解析；
- 方向键、Home / End、PageUp / PageDown、Enter、Backspace、Delete、Tab、`Esc` 走 `InputKeyDown`；
- `Ctrl+[` 映射为 `Esc`，`Ctrl+r` 映射重做；
- Normal / Visual 下 Ctrl / Alt 组合默认吞掉，避免触发编辑器的非模态命令；应用菜单快捷键（macOS 的 Command，Windows / Linux 的 Ctrl+S / Ctrl+F 等）放行；
- Insert 下除 `Esc` 外的所有键放行给编辑器，输入法保持开启。

`Esc` 在编辑器获得焦点时由 `App.View` 的窗口级 Esc 处理入口接管：先交给 Vim 退出 Insert / Visual 或取消 pending，再考虑 Markdown 预览；设置、搜索、重命名、新建、关于和命令行浮层仍然优先。

### 状态

- `vim.Session` 每个标签一个，保存在 `Tab.vim`；关闭 Vim 模式时置空；
- 模式、count、pending operator、可视选区、匿名寄存器都在 Session 里；
- 不写会话，不写草稿，重启后从 Normal 开始；
- 打开 Markdown 预览时重置为 Normal；返回源码后重新安装。

### 编辑

- 所有改动通过 `Buffer.Edit` 一次写入，配合 `Buffer.Transact` / 编辑器自身 undo；
- 不复制整份文档：移动和文本对象只读 `Buffer.Snapshot()`，不调用 `Buffer.String()`；
- 匿名寄存器同时写入系统剪贴板，`p` 在没有寄存器时读剪贴板；
- 只读标签允许移动、复制和搜索，编辑命令直接跳过。

### 搜索与命令

- `/` 在窗口底部打开命令行，Enter 后调用 `editor.State.Find`，再用 `editor.Cancel` 关闭内建查找栏；
- 当前匹配范围记录在 Session 里，`n` / `N` 把范围交给 finder，避免重复命中同一个结果；
- `:` 支持 `w`、`q`、`wq`、`x`、`:行号`；`wq` 走现有 `saveTab` 回调，未保存提示和草稿清理保持不变。

## 实现记录

- 新增 `internal/vim`：模式、count、motion、operator、text object、可视选区、寄存器、搜索范围；
- `internal/app/vim.go`：`OnInput` 接线、`vimHost`、模式切换、状态栏 badge、底部命令行；
- `Settings.VimMode`（`vimMode`，默认 false）与设置 → 编辑器开关；
- `Tab.vim` 每标签状态；分屏、会话、草稿、保存路径不变；
- locales 新增 `vimMode`、`vimModeDescription`、`vimNormal`、`vimInsert`、`vimVisual`、`vimVisualLine`、`vimCommand`；
- 测试：`internal/vim` 的模式 / 移动 / operator / 文本对象 / 可视 / 粘贴 / 缩进 / 只读；`internal/app` 的默认关闭、NORMAL 安装、插入与 Esc、可视删除、`/` 与 `n`、`:行号`、关闭状态放行文本、开关切换、只读、输入法随模式切换。

## 验收清单

- [ ] 默认关闭；设置 → 编辑器打开「Vim 模式（实验）」后当前标签进入 NORMAL，关闭后恢复普通编辑；
- [ ] `hjkl`、`w/b/e`、`0/^/$`、`gg/G`、方向键移动正确，行尾不会越过最后一个字符；
- [ ] `i/a/I/A/o/O` 进入插入，中文输入法正常；`Esc` 回到 NORMAL，光标停在上一个字符；
- [ ] `dd/yy/p`、`dw/cw`、`diw/ci"` 行为正确；
- [ ] `v/V` 选择并 `d/y/c`，`>`/`<` 缩进；
- [ ] `u` 与 `Ctrl+r` 正常，单条命令一次撤销；
- [ ] `/` 搜索、`n`/`N`、`:w/:q/:wq/:行号` 可用；
- [ ] 只读标签不修改；Markdown 预览不接管按键，`Esc` 仍返回源码；
- [ ] 设置、搜索、重命名、新建、关于弹窗打开时 Vim 不抢按键；
- [ ] `go test ./...`、`go vet ./...`、`go build ./...` 通过。
