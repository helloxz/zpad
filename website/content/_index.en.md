---
title: ZPad
hero:
  title: "A lightweight, native, focused text editor"
  lead: "Split panes, multiple tabs, Markdown preview. Drawn in Go — no WebView."
  note: "macOS, Windows and Linux"
  primary: "Download"
  secondary: "GitHub"
  shot: hero-en
  shot_hover: hero-en-dark
  alt: "ZPad with a Markdown checklist on the left and Go code on the right"
features_title: "It does the editing, and only that"
features_lead: "No plugins, no cloud sync, no Git, no AI, no WebView."
features:
  - icon: native
    title: "Light on memory"
    text: "Drawn natively: it starts fast and idles far below an Electron editor."
  - icon: encoding
    title: "Many formats"
    text: "txt, md, json, go, html, css open with a double-click, highlighted by extension."
  - icon: split
    title: "Split panes"
    text: "Two panes, each with its own tabs, one shared buffer."
  - icon: markdown
    title: "Markdown preview"
    text: "Renders headings, lists, code blocks and local images natively."
  - icon: sun
    title: "Twenty built-in themes"
    text: "One Dark, Dracula, Nord, Tokyo Night and more — or follow the system, window and editor together."
  - icon: grid
    title: "Ready out of the box"
    text: "No setup wizard, no plugin market. Drop a file in and write."
  - icon: session
    title: "Nothing lost on quit"
    text: "Unsaved tabs and text are kept, and come back after a crash or restart."
  - icon: tools
    title: "Built-in text tools"
    text: "Text diff, Base64, MD5, JSON format and minify, case conversion..."
  - icon: terminal
    title: "Vim mode (off by default)"
    text: "Turn it on in settings to move, select, delete and save with Vim keys."
features_more: "Also: encodings (UTF-8 / UTF-16 / GB18030), find and replace, cross-file search, session restore and one-click updates."
compare_title: "How it compares"
compare_label: "Feature"
compare_columns: ["ZPad", "Sublime Text", "Notepad++"]
compare_rows:
  - label: "Platforms"
    values:
      - "macOS, Windows, Linux"
      - "macOS, Windows, Linux"
      - "Windows only"
  - label: "Price and license"
    values:
      - "Free · AGPL-3.0, open source"
      - "$99 one-time · 3 years of updates · unlimited trial"
      - "Free · GPL-3.0, open source"
  - label: "Interface language"
    values:
      - "Chinese / English, built in"
      - "Community translation package (English only otherwise)"
      - "Many languages built in, incl. Simplified Chinese"
  - label: "Encodings and line endings"
    values:
      - "UTF-8, BOM, UTF-16, GB18030, auto-detected; line endings kept"
      - "UTF-8, UTF-16, GB18030 and more, picked by hand"
      - "UTF-8, UTF-16, local code page, auto-detected"
  - label: "Panes and tabs"
    values:
      - "Two fixed panes, tabs in each, one shared buffer"
      - "2–4 panes, tabs in each"
      - "Two views, tabs in each"
  - label: "Markdown preview"
    values:
      - "Built in"
      - "Needs a package"
      - "Needs a plugin"
  - label: "Packages"
    values:
      - "None, on purpose"
      - "Package Control"
      - "Plugin Admin"
  - label: "Updates in the app"
    values:
      - "Built in (install folder must be writable)"
      - "Built in"
      - "Built in (WinGUp; can be left out at install)"
compare_note: "Rows reflect the stable builds available in November 2026. Features change over time, so treat this as a reference only."
gallery_title: "Screenshots"
gallery:
  - name: split-en
    alt: "Split panes in the light theme"
    caption: "Split panes: one file, scrolling on its own in each pane."
  - name: split-en-dark
    alt: "Split panes in the dark theme"
    caption: "The dark theme, the whole window at once."
  - name: settings-en
    alt: "The settings sheet with the theme drop-down open"
    caption: "Settings: twenty themes, font, size and language, applied at once."
  - name: settings-en-dark
    alt: "The settings sheet in the dark, theme drop-down open"
    caption: "The same sheet in the dark, switching themes from the drop-down."
  - name: preview-en
    alt: "Markdown preview in the GitHub Light theme"
    caption: "Native Markdown preview: headings, task lists, code blocks and tables."
  - name: search-en-dark
    alt: "Searching open tabs in the One Dark theme"
    caption: "Search across open tabs, grouped by file."
faq_title: "Questions"
faq:
  - q: "Is ZPad free?"
    a: "Yes. It is open source under AGPL-3.0; the source and every release are on GitHub."
  - q: "Does it upload my files?"
    a: "No. It works offline; the only network request is the update check against GitHub."
  - q: "Any tracking or analytics?"
    a: "Not in the app. The code is fully open source with no tracking and no anonymous usage data. The website only counts page views to see how the download page is used, with no personal data. See the [privacy policy](/en/privacy/)."
  - q: "Why does macOS or Windows warn about it?"
    a: "The installers are not signed by Apple or Microsoft, so the system blocks them by default: right-click to open on macOS, or choose More info → Run anyway on Windows. You can check the source, or build it yourself."
  - q: "Is the updater safe?"
    a: "Update packages are verified with an Ed25519 signature and a mismatch is never installed. The app must live in a writable folder to update itself."
  - q: "Which systems are supported?"
    a: "macOS 12 or later, Windows 10 / 11 (64-bit), and Linux (x86_64, deb / rpm / tar.gz)."
  - q: "Why no plugins, Git or AI?"
    a: "ZPad is a plain text editor on purpose: fast to start, predictable to use. A heavier IDE fits those jobs better."
changelog_title: "Latest release"
changelog_lead: "Every release is written up on GitHub."
changelog_link: "All releases"
---
