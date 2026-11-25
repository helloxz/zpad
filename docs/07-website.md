# 官网（zpad.xphub.dev）

ZPad 的产品官网：首屏、功能、截图、常见问题、下载。源码在 `website/`，用 Hugo 生成静态文件，部署在 Cloudflare Pages。

- 线上地址：<https://zpad.xphub.dev>
- 默认语言中文（根路径），英文在 `/en/`
- 下载按钮读发布数据，不需要改代码
- 没有 npm、没有前端框架、没有构建服务端

## 范围

首页六段：首屏（截图 + 下载 / GitHub 按钮）、六个功能点、与 Notepad++ / Sublime Text 的对比表、六张截图（两列，点击放大）、常见问题、最近更新。导航是 功能 / 对比 / 下载 / 更新 / 反馈 / GitHub（窄屏只留 下载 / 更新 / 反馈 / GitHub），反馈直接进 GitHub 新建 issue；反馈与 GitHub 两个外链带 `rel="nofollow noopener"`，不外链权重。下载页按平台列安装包，macOS / Windows / Linux 各两个按钮。页脚只有两行内容：左侧 `© <年份> ZPad. All rights reserved.`（年份取构建时间），右侧隐私政策链接；不再放许可、源码、构建来源。另有 `/privacy/`（英文 `/en/privacy/`）隐私政策，只在页脚右侧链接。不做的：博客、文档站、评论区、账号体系。

`docs/` 下的开发文档不对外，官网只链到 GitHub。

## 技术选型

| 项目 | 选择 | 原因 |
| --- | --- | --- |
| 生成器 | Hugo **0.170.1** | 单二进制、无 npm，CF Pages 有官方预设；多语言内建 |
| 样式 | 手写 CSS（`assets/css/site.css`） | 页面不多，用不上 UI 框架；色板直接沿用 `internal/theme` |
| 字体 | 系统字体 | 首屏不下载字体文件 |
| 图标 | 内联 SVG（`layouts/partials/icon.html`）+ 应用图标 PNG | 不引外部图标库；favicon 和页头 logo 用应用自己的 icon |
| 主题 | 亮 / 暗，跟随系统，可手动切换 | 和 App 一致；首屏固定 GitHub Light，悬停一层切 One Dark |
| 截图 | `go test` 里的 headless 渲染 | 不需要屏幕录制权限，中英各一套，可复现 |

配色取自 `internal/theme`：亮色 `#4F46E5` 主色、`#F7F8FA` 表面色；暗色 `#0F1115` 背景、`#7C8CFF` 主色。

## 目录

```
website/
├── hugo.toml                  # 站点、语言、输出设置
├── content/
│   ├── _index.md|.en.md       # 首页文案：功能点、截图说明、FAQ
│   ├── download.md|.en.md     # 下载页文案与安装说明
│   └── privacy.md|.en.md      # 隐私政策，front matter 带 updated
├── i18n/{zh,en}.toml          # 模板里复用的 UI 字符串
├── layouts/
│   ├── baseof.html            # 含截图灯箱 <dialog>
│   ├── index.html             # 首页
│   ├── index.redirects        # 由版本数据生成 _redirects
│   ├── _default/download.html # 下载页（layout: download）
│   ├── _default/single.html   # 普通正文页（隐私政策）
│   ├── robots.txt
│   └── partials/{head,header,footer,icon,shot,release,release-assets}.html
├── assets/
│   ├── css/ js/
│   ├── icon.png               # resources/icon.png 裁掉留白的副本
│   └── screenshots/*.png      # 截图原图，Hugo 构建时转 WebP
└── data/release.json          # 最新 release 快照：构建失败时回退，也供本地预览
```

## 多语言

- `defaultContentLanguage = "zh"`、`defaultContentLanguageInSubdir = false`：中文在 `/`，英文在 `/en/`
- 页面翻译按文件名：`content/_index.md`（中）+ `content/_index.en.md`（英）
- 模板里的固定文案走 `i18n`，页面文案走各自语言 front matter
- `head.html` 输出 `hreflang`（含 `x-default` 指中文），语言切换链接指向当前页面的对应译文
- 截图按语言分两套：中文 `split.png`，英文 `split-en.png`，由 front matter 里的名字决定；首屏的 `hero` 与悬停用的 `hero-dark` 同理

## 图标与 favicon

`resources/icon.png` 是应用图标（1024×1024，圆角外透明）。构建期用 Hugo 的图片处理生成 32 / 192 / 180（apple-touch-icon）三个 PNG，页头 logo 用同一张图的 28px 版本，所以官网和 App 的图标永远一致。

`website/assets/icon.png` 是从 `resources/icon.png` 裁掉约 10% 透明留白后的副本（824×824）。**换应用图标时要同步这个文件**，否则官网还是旧图标：

```sh
sips -c 824 824 resources/icon.png --out website/assets/icon.png
```

## 截图

截图不是手截的，由 `internal/app/site_shots_test.go` 在内存里渲染，因此不需要屏幕录制权限，也不会混进真实路径：

```sh
ZPAD_SHOTS=/tmp/shots go test ./internal/app -run TestSiteScreenshots
cp /tmp/shots/*.png website/assets/screenshots/     # 先按需缩小
```

渲染出 8 个场景 × 中英 = 16 张，1600×900 左右：

| 场景 | 内容 | 主题 |
| --- | --- | --- |
| `split` | 左右分屏：左边 Markdown 源码，右边 Go 代码 | ZPad 亮色 / 暗色 |
| `settings` | 设置面板，主题下拉展开 | ZPad 亮色 / 暗色 |
| `preview` | Markdown 原生预览，单栏 | GitHub Light |
| `search` | 跨文件搜索，5 个标签全开 | One Dark |
| `hero` | 与 `split` 同一个画面，只给首屏用 | GitHub Light / One Dark |

截图区用 `split`、`settings`、`preview`、`search` 共 6 张：前两组亮暗各一张，后两张各一张（预览亮、搜索暗），中英各一套。首屏不用 `split`，而是 `hero` 的 GitHub Light 版本，鼠标停在图上时由 CSS 淡入上面的 One Dark 版本（`hero-hover` / `hero-dark`，`@media (hover: hover)` 内，触屏不会卡在暗色）；首屏不再跟随站点亮暗切换。

场景由测试里的 demo 工作区（`zpad-demo`，5 个文件）拼出来，改文案或改功能后重跑一次即可。原图按 `sips -Z 1600` 缩到 1600 宽再放进 `assets/`，Hugo 构建时转成 WebP（`shot.html`）。

截图区固定展示 6 张（两列），点击用原生 `<dialog>` 放大：控制条在图片下方居中（上一张 / 张数 / 下一张 / 关闭），支持 Esc、点击遮罩、左右方向键，脚本不跑时链接直接打开原图。首屏大图不参与灯箱，点它不会弹出。

README 的截图直接引用 `website/assets/screenshots/` 下的 `split`、`split-dark`、`settings`、`settings-dark`、`preview`、`search-dark`，重截后自动跟着更新。

## 下载数据

版本号、下载地址、文件大小和首页「最近更新」都不再手写，也没有 CI 提交：

- `layouts/partials/release.html` 在**生产构建**时向 `https://api.github.com/repos/helloxz/zpad/releases/latest` 取最新 release，`transform.Unmarshal` 后直接给模板使用；`partialCached` 保证一次构建只请求一次。
- 缓存键带 `now.Unix`，所以 CF 重建不会从 Hugo 的远程缓存拿到上一版；每次构建都是当下最新。
- `hugo server` 是 development 环境，不请求 GitHub，直接读 `website/data/release.json` 快照；`hugo --minify` 是 production，会真实请求。
- 请求失败（断网、限流、GitHub 不可达）输出 `WARN latest release: ...` 并回退到同一份快照，构建继续。只有快照也没有数据时，胶囊、最近更新和下载按钮才不渲染，导航「更新」也会隐藏。

发布后到 CF Pages → Deployments → **Retry deployment**（或 Create deployment，选 `main`）。构建时取最新 release，官网跟着更新。这一步是手动的；忘了官网就停在上一版。仓库里不会出现 bot commit。

`website/data/release.json` 是快照，模板只用到 GitHub API 的 `tag_name`、`published_at`、`html_url`、`body` 和 `assets[].{name,size,browser_download_url}`；快照可以保留整份 API 响应，也可以只留这些字段。平时不用手改，本地要预览最新版本时刷新一次：

```sh
curl -fsSL -H 'Accept: application/vnd.github+json' \
  https://api.github.com/repos/helloxz/zpad/releases/latest \
  > website/data/release.json
```

下载地址和大小从 `assets` 里按文件名分类（`release-assets.html`），所以 `.delta`、`update-*.json`、`install.*` 不会出现在下载页。下载页只给 deb / rpm 两个按钮（各平台都两个，排版对齐）；tar.gz 不进按钮，一条 `/download/linux-tar` 短链和安装说明里的 `install.sh` 一行命令都靠它。

固定短链由 `layouts/index.redirects` 在构建时写进 `public/_redirects`（`hugo.toml` 里自定义的无后缀 `text/redirects` 输出格式），不再放 `static/`：

```
/download/mac-arm64    → 最新 Apple Silicon dmg
/download/mac-intel    → 最新 Intel dmg
/download/windows      → 最新安装包
/download/windows-portable
/download/linux        → 最新 deb
/download/linux-rpm    → 最新 rpm
/download/linux-tar    → 最新 Linux tar.gz
```

GitHub API 匿名限额是每 IP 每小时 60 次，每次构建只请求一次；CF 的构建 IP 可能共享，真被限流时可以在 CF Pages 的环境变量里放一个只读 `GITHUB_TOKEN`，`release.html` 会自动加上 `Authorization`（`hugo.toml` 的 `security.funcs.getenv` 已放行）。发布成 draft 或 pre-release 时 `/releases/latest` 不会返回它，构建会继续用上一版，所以仍要发布为 latest 并等产物传完。

## 部署（Cloudflare Pages）

1. Workers & Pages → Create → Pages → Connect to Git → 选 `helloxz/zpad`
2. 构建配置：

   | 字段 | 值 |
   | --- | --- |
   | Production branch | `main` |
   | Framework preset | Hugo |
   | Root directory | `website` |
   | Build command | `hugo --minify` |
   | Build output directory | `public` |

3. 环境变量 `HUGO_VERSION=0.170.1`。CF 的默认 Hugo 版本会漂，锁住才能和本地一致。
4. Custom domains → 填 `zpad.xphub.dev`，DNS 加一条 CNAME 指向 `<project>.pages.dev`。
5. 部署后确认 `/download/mac-arm64` 会 302 到当前版本的 dmg。
6. 每次发布后在 Deployments → Retry deployment 触发构建；构建时取最新 release，仓库里不会产生提交。

`website/public` 不进 Git，CF 自己构建。

## 访问统计

官网用第三方访问统计脚本（`tj.rss.ink`）。脚本标签写在 `website/hugo.toml`：

```toml
[params]
  analyticsScript = '<script defer src="https://tj.rss.ink/custom.js" data-website-id="<id>"></script>'
```

`head.html` 在 `analyticsScript` 非空时把它原样输出到 `<head>` 末尾，留空则页面没有任何统计脚本。FAQ 里「有统计或追踪吗」的回答要与此保持一致：应用零统计，官网只有页面浏览量的基础访问统计，不收集个人数据。

## 隐私政策

`/privacy/` 与 `/en/privacy/` 由 `content/privacy.md|.en.md` + `layouts/_default/single.html` 生成，标题是「隐私政策 / Privacy Policy」（写「协议」会像要用户签署，而这页只是告知）。正文十段：本政策说明什么、应用、官网访问日志、访问统计、本地存储、第三方链接、数据共享与安全、未成年人、你的权利与联系方式、政策更新。统计段写明字段、无 cookie、不跨站追踪、保留口径（仅在汇总分析所需期间保留）；只要 `analyticsScript` 的行为或字段有变，这条页面和首页 FAQ 一起改，三处说法必须一致。页面不放弹窗，入口只在页脚右侧。

## 本地开发

```sh
brew install hugo                  # 0.170.1
cd website && hugo server          # http://localhost:1313/，development，用 data/release.json 快照
hugo --minify                      # production，会请求 GitHub 取最新 release，产出 website/public
```

`hugo server` 不请求 GitHub，避免边改边触发 API 限额；要看最新版本先执行上面的 `curl` 刷新快照。只在 `website/` 下改，`go test ./...` 不会碰它；`release_test.go` 会检查快照结构。

## 验收清单

- [ ] 首页首屏显示版本号、截图、下载 / GitHub 按钮，logo 是应用图标
- [ ] 语言切换在首页和下载页都能回到对应译文
- [ ] 亮 / 暗切换后页面配色跟着换，刷新后保持；首屏仍是 GitHub Light，悬停淡入 One Dark
- [ ] 截图区六张、两列排布，点击后放大，按钮在图片下方居中，Esc / 遮罩 / 方向键可用
- [ ] 常见问题折叠展开正常，内容与隐私表述一致
- [ ] 对比表中英两种语言各 8 行，ZPad 列带底色高亮；手机（390px）下每行改成一张卡片，页面不横向滚动
- [ ] 手机（390px）下导航、卡片、按钮、截图不溢出，截图单列
- [ ] 导航的「反馈」打开 GitHub 新建 issue 页面
- [ ] 下载页每个平台两个按钮（Linux 为 deb / rpm）指向当前版本，安装说明含 sudo 与 `install.sh` 命令
- [ ] `/download/mac-arm64` 302 到当前版本 dmg，`/download/linux` 302 到当前版本 deb
- [ ] 构建时取到最新 release：首屏胶囊、下载页版本 / 地址 / 大小、首页最近更新都来自它
- [ ] GitHub API 不可用时构建回退到 `data/release.json` 快照并有 warning，站点仍能部署
- [ ] 发布后在 CF Pages 点 Retry deployment，官网随构建更新；`main` 上没有 bot commit
- [ ] 页脚左侧版权（年份随构建更新）、右侧隐私政策链接，390px 下左右对齐且不换行
- [ ] 隐私政策中英文各一页，亮暗两种主题下正文与页脚都正常
- [ ] 无 JS 时页面仍可读、按钮仍可用（首屏固定 GitHub Light，悬停只靠 CSS；最近更新由构建生成；截图链接直接开原图）

## 实现记录

- 2026-11-23：Hugo 站点落地。首页、下载页、双语、亮暗主题、发布数据自动化、CF Pages 接入说明。
- 2026-11-23：首屏与截图区改用 headless 渲染的新截图（中英各一套）；favicon 与页头 logo 换成 `resources/icon.png`；新增常见问题；下载页改名「安装说明」并补 `xattr` 命令；截图支持点击放大。
- 2026-11-23：标题定为「轻量、原生、纯粹的文本编辑器」；撤掉 Markdown 预览截图，图库缩为四张两列（预览呈现方式待调整）；灯箱控制条移到图片下方居中并加张数；导航加「反馈」进 GitHub issues；`word-break: keep-all` 避免手机端把「纯粹」拆行。
- 2026-11-23：首页新增「对比」段（ZPad / Sublime Text / Notepad++，8 行），行数据写在首页 front matter 的 `compare_rows`，`index.html` 只负责渲染，样式在 `site.css` 的 `.compare*`；窄屏（≤720px）thead 隐藏、每人一列改成同一张卡片里按 `data-col` 标注；导航加「对比」。ZPad 列那层 `--accent-soft` 底色上普通 `--border` 分隔线几乎看不见，所以该列的分隔线走新增的 `--accent-line`（亮暗各一档）。脚注不写具体版本号：`TestWebsiteHasNoHardcodedVersion` 会拦住 `x.y.z` 文案，版本用「2026 年 11 月的稳定版本」表述。
- 2026-11-23：下载页补上 Linux：生成脚本认领 `zpad_<版本>_amd64.deb`、`zpad-<版本>-1.x86_64.rpm`、`zpad-<版本>-linux-amd64.tar.gz`，新增 `/download/linux`、`/download/linux-rpm`、`/download/linux-tar` 短链并重新生成现有版本的数据；Linux 卡片给 deb（主按钮）与 rpm 两个按钮，与另外两个平台对齐，tar.gz 只在安装说明里通过 `install.sh` 一行命令出现。说明里写清 deb / rpm 由包管理器更新、只有装进 `~/.local` 的实例能自更新；首页首屏与常见问题同步去掉「即将上线」。
- 2026-11-24：0.5.0 界面截图更新。截图场景扩到 4 组：分屏与设置保持 ZPad 亮 / 暗色，设置展开主题下拉；新增 Markdown 预览（GitHub Light）与跨文件搜索（One Dark），图库从 4 张变 6 张。功能卡「亮 / 暗主题」改为「20 套内置主题」。首屏换成 `hero`（GitHub Light），鼠标悬停淡入 One Dark 版本，不再跟站点主题切换；`shot.html` 与 `site.js` 里原来的首屏亮暗切换逻辑一并删掉。
- 2026-11-24：版本数据改为构建时从 GitHub API 取。新增 `partials/release.html`（生产构建取 `releases/latest`，失败回退 `data/release.json` 快照，缓存键带构建时间）和 `partials/release-assets.html`（按文件名分类并格式化大小）；首页、下载页、JSON-LD 都用它。`_redirects` 改由 `layouts/index.redirects` 生成，删除 `static/_redirects`。删除 `.github/workflows/website.yml`、`release.yml` 的 `site` 作业和 `website/scripts/release/main.go`：仓库不再有 bot commit；发布后手动在 CF Pages 点 Retry deployment。`hugo.toml` 放行只读 `GITHUB_TOKEN`（可选）。
- 2026-11-25：官网统计换成 `tj.rss.ink` 的访问统计脚本，配置在 `hugo.toml` 的 `analyticsScript`；移除 Cloudflare Web Analytics 的 `cfAnalyticsToken` 与 beacon，FAQ 和本节文案不再提 Cloudflare。
- 2026-11-25：新增 `/privacy/` 与 `/en/privacy/` 页面（`content/privacy.md|.en.md` + `_default/single.html` + `.legal` 样式），页脚加隐私入口，首页 FAQ 链接过去；不加同意弹窗，因为统计脚本不写 cookie。
- 2026-11-25：页面定名「隐私政策 / Privacy Policy」，正文扩到十段（应用、访问日志、统计字段、本地存储、第三方、共享与安全、未成年人、权利与联系、更新）；页脚改成左版权 / 右隐私政策，移除许可、源码、构建来源三项；对比表「Markdown 预览」的 ZPad 一列从「内置，源码 + 预览可分屏」收成「内置」，中英同步。
- 2026-11-25：隐私政策去掉标题下那句 lead（页面直接从标题进日期）；顶部「反馈」与「GitHub」两个外链加 `nofollow`。
- 说明：`analyticsScript` 留空即不加载任何统计脚本。
