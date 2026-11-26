# 文本对比

状态：**待签收**（未发布）

## 背景

ZPad 的标签里经常同时开着同一份文件的两版、一个文件和它的备份，或者一个已保存文件和一个还没保存的草稿。把两段文字并排看一眼，比来回切标签快得多。这个功能只做一件事：选两个已经打开的标签，看清它们哪里不同。

它不追求完整的 diff 工具：不做磁盘版本、不做未打开的文件、不做补丁导出，也不做编辑或逐块采纳。打开、选择、看结果，关掉。

## 需求

- 入口：工具栏扳手（工具）下拉的第一项「文本对比」。编辑器右键的「工具」只保留 Base64 / MD5 / JSON / 行号，不显示文本对比；标签右键、快捷键与「显示」菜单这次都不加。
- 居中弹窗，尺寸随窗口伸缩（最大约 1100×780），Esc 或点遮罩关闭。
- 顶部两个选择器 A / B，各选一个**已打开**的标签；选择器只显示文件名，完整路径放在悬停提示里，重名文件仍能区分。中间 `⇄` 交换；右侧「对比 / 重新对比」。
- 打开时预填：A = 当前标签，B = 另一栏的活动标签，没有分屏就取下一个打开的标签。常见情况下打开后直接点对比。
- 内容来源是标签的 `editor.Buffer`：未保存的修改、未命名文件都在里面。**不读 `drafts/`**：草稿是崩溃恢复副本，最多落后约 5 秒，拿它对比会出现假差异；标签关闭后也不进入选择器。
- 以下情况在弹窗内提示并禁用「对比」，不弹 toast：
  - 少于两个标签，或某侧没选；
  - A 和 B 是同一个标签；
  - 某侧 `Buffer.Len() == 0`（空白标签或空文件）；
  - 某侧超过 **1 MB 或 2 万行**（每侧独立计算）。
- 两侧文本完全相同：不创建 diff，直接显示「两侧内容完全一致（共 N 行）」。
- 对比结果用 MyGo 自带的 `editor.DiffView`（`DiffState`）：
  - 统一视图（`DiffUnified`）：删除行在新增行上方，行号、`+` / `−` 与底色；
  - 行内词级高亮（`DiffInlineWordAlt`，默认）；
  - 未变行按 3 行上下文折叠，分隔条可展开；
  - 两侧同步滚动、行内查找、`Cmd+F8` / `Shift+Cmd+F8` 跳变更；
  - 大文本（两侧合计超过 1 万行或 1 MB）由 MyGo 在后台比较，期间显示「正在对比…」，完成后自动刷新。
- 底部状态条：`+N −M`、`K 处差异`、上一处 / 下一处、`正在对比…`，右侧是 Esc / 仅比较已打开的标签 / 上限说明。
- 两侧各复制一份快照 buffer 交给 diff，文档自身的 Buffer 不会被修改，也不与 diff 视图共享。弹窗打开期间的编辑不会被对比跟随；「重新对比」会重新取快照。
- 字体、行高、Tab 宽度、自动折行跟随设置；颜色走编辑器主题的 `Added` / `Deleted` / `Modified`（`internal/theme` 已经用 Success / Danger / Warning 设置），20 套主题都不需要单独加 token。
- 关闭弹窗即释放 diff 与两份快照；关闭 A / B 所在标签时清空对应侧和结果，弹窗保持打开，让用户重新选择。
- 弹窗打开期间，窗口级快捷键（`⌘⇧F` / `⌘⇧V` / `⌘⇧B` / `⌘⇧J` 等）暂停；`⌘F` 交给 diff 自己的查找；搜索面板与代码片段面板不会叠在它上面。

## 交互

```
工具栏：… 🔧 [<>] (i) …          ← 扳手下拉第一项
        └─ 文本对比

┌ 文本对比 ─────────────────────────────────────────────┐
│ A [ file-1.txt                   ▾ ] ⇄               │
│ B [ file-2.txt                   ▾ ]   [ 对比 ]      │
│ ┌───────────────────────────────────────────────────┐ │
│ │ 1   1   package main                              │ │
│ │ 2   2                                             │ │
│ │ 4      −   println("hello")                       │ │
│ │     4   +   println("hi")                         │ │
│ │     5   +   println("there")                      │ │
│ └───────────────────────────────────────────────────┘ │
│ +2 −1 · 1 处差异  ↑ ↓        Esc 关闭 · 上限 1 MB…   │
└───────────────────────────────────────────────────────┘
```

## 实现

- `internal/app/diff.go`：面板状态、A/B 预填与交换、空 / 同标签 / 超限判断、快照与 `DiffState` 构造、标签关闭清理、`findActive`。
- `internal/app/diff_ui.go`：弹窗、两个选择器、交换 / 对比按钮、diff 主体、底部状态条。
- `internal/app/app.go`：`diffOpen`、`diffA` / `diffB`、`diff *editor.DiffState`、`diffIdentical`、`diffFocus`；`removeTab` 调用 `diffDocumentClosed`。
- `internal/app/tools.go`：`toolMenu` 只保留文本工具；工具栏通过 `compareToolMenu` 在顶部插入「文本对比」和一条分隔线。编辑器右键继续只调 `toolMenu`，因此「工具」里不显示文本对比。
- `internal/app/view.go`：绘制弹窗；顶层快捷键门禁加 `!a.diffOpen`；工具栏查找改用 `findActive`。
- `internal/app/snippets.go` 的 `panelOpen` 与 `internal/app/search.go` 的搜索入口加 `diffOpen`，避免面板叠加。
- `internal/app/menu.go` 的查找菜单项改用 `findActive`。
- 不新增 `internal/diff`，不自己写行渲染、hunk 或 diff 颜色。

`DiffState` 的关键项：

```go
&editor.DiffState{
    Old: oldBuf, New: newBuf,        // 两侧的 Buffer.String() 快照
    Editable: false,                 // 只读
    Layout: editor.DiffUnified,
    Indicators: editor.DiffClassic,  // + / −
    ContextLines: 3,
    KeepScrollbars: true,
    Wrap: a.Settings.SoftWrap,
    Font: ui.Font{Family: a.Settings.FontFamily, Size: a.Settings.FontSize},
    LineHeight: a.Settings.LineHeight,
    TabSize: a.Settings.TabSize,
    Theme: a.themeDefinition().Code,
}
```

构建顺序：先构建 `editor.DiffView`，再读 `Stats()` / `Hunks()` / `Comparing()`。`DiffView` 会给新 buffer 注册窗口监听，大文本才会走后台比较；反过来先读统计会让它在主线程同步算完。

单侧上限：1 MB（`1 << 20` 字节）或 2 万行，`locales.CompareLimit` 的文案与 `diffMaxBytes` / `diffMaxLines` 必须同步。行数按 diff / git 的口径算：以换行结尾的文本不再多算一个空行。

## 测试

- 预填 A/B（当前标签 + 下一标签）。
- 未保存修改从 Buffer 读取，文档内容不被修改。
- 相同文本短路为「完全一致」，不建 diff。
- 空侧、超限侧、同一标签的拦截与提示。
- 关闭 A/B 标签后清空对应侧与结果，弹窗保持打开。
- 工具菜单第一项打开面板。
- 底部统计跟随 diff（`+1 −1`、`1 处差异`）。
- Esc 关闭并释放 diff。
- 现有菜单、搜索、片段、面板互斥测试不回归。

## 非目标 / 已知限制

- 只比较两个**已打开**的标签；不比较磁盘版本、未打开的文件、草稿，也不做「当前标签 vs 已保存版本」。
- 快照式：弹窗打开后文档在别处被改（菜单撤销、外部重载）不会自动跟随；点「重新对比」。
- 固定统一视图，不提供并排切换；不提供行内编辑、Revert / Keep、批注、行选择、补丁导出。
- 不持久化：关闭弹窗不保留选择、滚动和结果；会话恢复不包含它。
- 每侧 1 MB / 2 万行；更大的文件只提示，不比较。MyGo 自身可以后台比较更大的文本，以后要放宽只改两个常量与一条文案。
- 没有应用级快捷键、标签右键入口、显示菜单入口；入口只有扳手下拉；编辑器右键「工具」不显示文本对比。
- 「N 行未修改」、两侧无障碍名称、未换行标记等 diff 内部文案来自 MyGo 自己的 catalogs，随窗口语言切换。
