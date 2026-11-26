# ZPad

ZPad 是一款轻量、原生、纯粹的文本编辑器，基于 [MyGo](https://mygo.run/) 框架进行开发，没有插件、没有云同步、没有 Git、没有 AI、没有 WebView。

## 功能特点

- 支持打开整个项目，文件树随点随开
- 支持常见文本格式：txt、md、css、js、json、yaml、yml、html、conf、toml、go、py、php 等
- 原生渲染，不使用 WebView，启动快、内存占用更低
- 跨平台：支持 macOS、Windows 和 Linux
- 支持左右、上下分屏查看与编辑
- 支持一键在线更新
- 支持 Markdown 原生预览
- 支持代码片段：按分组管理，点一下插入到光标处
- 内置 20 套主题：ZPad 亮 / 暗色、One Dark、Dracula、Nord 等，可跟随系统
- 支持中文、英文双语切换
- 内置基础文本工具：查找、替换、大小写转换等

## 快捷键

macOS 使用 `⌘`（Command），Windows / Linux 使用 `Ctrl`；`⌃` 表示 macOS 的 Control 键。

### 文件

| 动作 | macOS | Windows / Linux |
| --- | --- | --- |
| 新建文件 | `⌘N` | `Ctrl+N` |
| 打开文件 | `⌘O` | `Ctrl+O` |
| 打开文件夹 | `⌘⇧O` | `Ctrl+Shift+O` |
| 保存 | `⌘S` | `Ctrl+S` |
| 另存为 | `⌘⇧S` | `Ctrl+Shift+S` |
| 关闭当前标签 | `⌘W` | `Ctrl+W` |

### 编辑

| 动作 | macOS | Windows / Linux |
| --- | --- | --- |
| 撤销 / 重做 | `⌘Z` / `⌘⇧Z` | `Ctrl+Z` / `Ctrl+Shift+Z` |
| 查找 | `⌘F` | `Ctrl+F` |
| 替换 | `⌘⌥F` | `Ctrl+H` |
| 下一个 / 上一个匹配 | `⌘G` / `⌘⇧G` | `F3` / `Shift+F3` |
| 跳转到行 | `⌃G` | `Ctrl+G` |
| 切换注释 | `⌘/` | `Ctrl+/` |
| 选择下一个相同文本 | `⌘D` | `Ctrl+D` |
| 选择所有相同文本 | `⌘⇧L` | `Ctrl+Shift+L` |
| 删除当前行 | `⌘⇧K` | `Ctrl+Shift+K` |

大小写转换没有默认快捷键，可在工具栏的 `Aa` 菜单或「编辑 → 大小写转换」中使用。

### 视图

| 动作 | macOS | Windows / Linux |
| --- | --- | --- |
| 切换侧栏 | `⌘B` | `Ctrl+B` |
| 切换分屏 | `⌘\` | `Ctrl+\` |
| 在所有已打开文件中搜索 | `⌘⇧F` | `Ctrl+Shift+F` |
| 插入代码片段 | `⌘⇧J` | `Ctrl+Shift+J` |
| 切换 Markdown 预览 | `⌘⇧V` | `Ctrl+Shift+V` |
| 返回源码（Markdown 预览中） | `Esc` | `Esc` |
| 在浏览器中打开 HTML | `⌘⇧B` | `Ctrl+Shift+B` |
| 打开设置 | `⌘,` | `Ctrl+,` |
| 最小化窗口 | `⌘M` | 使用标题栏按钮 |

跨文件搜索面板中，输入后按 `↵` 搜索（结果是最新时直接打开选中项），`↑` / `↓` 选择结果，`Esc` 关闭面板。

代码片段面板中，`↑` / `↓` 选择、`↵` 插入、`Esc` 关闭；左栏是分组（首行「全部」），右栏是片段，表头的 `＋` 新建分组或片段，`✎` / `🗑` 改名或删除选中的分组。

编辑器保留了 MyGo 内置的完整键位（多光标、行移动、折行、代码折叠等），以上只列出 ZPad 自己定义和常用的部分。

## 截图

| 亮色 | 暗色 |
| --- | --- |
| ![亮色](./website/assets/screenshots/split.png) | ![暗色](./website/assets/screenshots/split-dark.png) |

| 设置（亮色） | 设置（暗色） |
| --- | --- |
| ![设置亮色](./website/assets/screenshots/settings.png) | ![设置暗色](./website/assets/screenshots/settings-dark.png) |

| Markdown 预览（GitHub Light） | 跨文件搜索（One Dark） |
| --- | --- |
| ![Markdown 预览](./website/assets/screenshots/preview.png) | ![跨文件搜索](./website/assets/screenshots/search-dark.png) |

## 联系

- 博客：<https://blog.xiaoz.org/>
- X：<https://x.com/xiaozblog>
