---
title: ZPad
hero:
  title: "轻量、原生、纯粹的文本编辑器"
  lead: "左右分屏、多标签、Markdown 预览。界面全部由 Go 绘制，没有 WebView。"
  note: "macOS、Windows 与 Linux"
  primary: "下载"
  secondary: "GitHub"
  shot: hero
  shot_hover: hero-dark
  alt: "ZPad 主界面：左侧 Markdown 清单，右侧 Go 代码，两栏分屏"
features_title: "只做编辑器该做的事"
features_lead: "没有插件、没有云同步、没有 Git、没有 AI、没有 WebView。"
features:
  - icon: native
    title: "轻量低占用"
    text: "原生绘制，启动快，常驻内存远低于 Electron 编辑器。"
  - icon: encoding
    title: "多格式与高亮"
    text: "txt、md、json、go、html、css 双击即开，按扩展名高亮。"
  - icon: split
    title: "左右分屏"
    text: "两栏各开多标签，同一文件共享一份缓冲区。"
  - icon: markdown
    title: "Markdown 预览"
    text: "原生渲染标题、列表、代码块和本地图片。"
  - icon: sun
    title: "20 套内置主题"
    text: "One Dark、Dracula、Nord、Tokyo Night 等，也能跟随系统，窗口和编辑器一起变。"
  - icon: grid
    title: "开箱即用"
    text: "没有配置向导和插件市场，文件拖进来就能写。"
features_more: "还有编码与行尾（UTF-8 / UTF-16 / GB18030）、查找替换、会话恢复。"
compare_title: "同类编辑器对比"
compare_label: "对比项"
compare_columns: ["ZPad", "Sublime Text", "Notepad++"]
compare_rows:
  - label: "支持平台"
    values:
      - "macOS、Windows、Linux"
      - "macOS、Windows、Linux"
      - "仅 Windows"
  - label: "价格与许可"
    values:
      - "免费 · AGPL-3.0 开源"
      - "$99 买断 · 含 3 年更新 · 可无限期试用"
      - "免费 · GPL-3.0 开源"
  - label: "界面语言"
    values:
      - "中文 / English（内置）"
      - "需安装社区汉化插件（官方仅英文）"
      - "内置多语言（含简体中文）"
  - label: "编码与行尾"
    values:
      - "UTF-8·BOM·UTF-16·GB18030，自动检测，保存保留原行尾"
      - "UTF-8·UTF-16·GB18030 等，需手动选择"
      - "UTF-8·UTF-16·本地代码页，自动检测"
  - label: "分屏与标签"
    values:
      - "左右双栏 × 每栏多标签，同一文件共享缓冲"
      - "2–4 栏 × 多标签"
      - "双视图 × 多标签"
  - label: "Markdown 预览"
    values:
      - "内置"
      - "需安装插件"
      - "需安装插件"
  - label: "插件生态"
    values:
      - "无（有意取舍）"
      - "Package Control"
      - "Plugin Admin"
  - label: "在线更新"
    values:
      - "有（需装在可写目录）"
      - "有"
      - "有（WinGUp，安装时可不装）"
compare_note: "对比基于各软件在 2026 年 11 月的稳定版本，随着时间推移，功能可能发生变化，对比结果仅供参考！"
gallery_title: "截图"
gallery:
  - name: split
    alt: "亮色主题下的左右分屏"
    caption: "左右分屏：同一份文件在两栏各自滚动。"
  - name: split-dark
    alt: "暗色主题下的左右分屏"
    caption: "暗色主题，整个窗口一起变。"
  - name: settings
    alt: "设置面板，主题下拉已展开"
    caption: "设置：20 套主题、字体、字号、语言，改完即生效。"
  - name: settings-dark
    alt: "暗色主题下的设置面板，主题下拉已展开"
    caption: "暗色下的设置面板，主题在下拉里直接换。"
  - name: preview
    alt: "GitHub Light 主题下的 Markdown 预览"
    caption: "Markdown 原生预览：标题、任务列表、代码块和表格。"
  - name: search-dark
    alt: "One Dark 主题下的跨文件搜索"
    caption: "跨文件搜索：在已打开的标签里查找，结果按文件分组。"
faq_title: "常见问题"
faq:
  - q: "ZPad 收费吗？"
    a: "免费。以 AGPL-3.0 开源，源码和每个版本都在 GitHub 上。"
  - q: "会上传我的文件吗？"
    a: "不会。应用不联网也能正常使用，唯一的网络请求是向 GitHub 检查更新。"
  - q: "有统计或追踪吗？"
    a: "应用内没有。代码完全开源，不含任何追踪和匿名数据上报；官网只做基础的访问统计（页面浏览量），用来了解下载页的使用情况，不收集个人数据。详见[隐私政策](/privacy/)。"
  - q: "为什么 macOS 或 Windows 会拦截？"
    a: "安装包没有 Apple / Microsoft 的开发者签名，系统会按默认策略先拦一下：macOS 右键打开，Windows 选择「仍要运行」。介意的话可以自己核对源码，或者从源码构建。"
  - q: "自动更新安全吗？"
    a: "更新包用 Ed25519 签名校验，签名不符不会安装。应用需要装在可写目录才能自更新。"
  - q: "支持哪些系统？"
    a: "macOS 12 及以上、Windows 10 / 11（64 位）、Linux（x86_64，deb / rpm / tar.gz）。"
  - q: "为什么不支持插件、Git、AI？"
    a: "定位是一个纯粹的文本编辑器：启动够快、行为可预期。需要这些能力时，用更重的 IDE 更合适。"
changelog_title: "最近更新"
changelog_lead: "每次发布都会写在 GitHub Releases 里。"
changelog_link: "全部版本"
---
