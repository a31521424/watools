# 插件开发 01: LLM 必读

## 信任模型

WaTools 当前默认用户**完全了解并主动选择自己安装的插件**。

- 插件应视为用户主动信任的代码
- 当前实现不是面向“不受信任第三方插件市场”的强隔离沙箱
- 不要向用户描述为“安装任意来源插件也绝对安全”

---

## ⚡️ LLM 快速指南

### 核心原则

**最终产出 = 浏览器可直接运行的文件 + .wt 压缩包**

不是项目源代码,而是编译后的可运行文件。

### 构建流程决策树

```text
START: 用户需要插件
    ↓
复杂吗? (需要 TypeScript/React/框架?)
    ├─ 否 → 使用【简单模式】
    │        ├─ 直接编写 HTML/JS/CSS
    │        ├─ 创建 manifest.json + app.js + index.html
    │        └─ 跳转到打包
    └─ 是 → 使用【构建模式】
             ├─ src/ 放开发源码
             ├─ public/ 放 manifest.json + app.js
             ├─ vite build 输出到 dist/
             └─ 将 dist/ 根级内容打包为 .wt
```

### LLM 输出清单

**简单模式**:

- `manifest.json`
- `app.js`
- `index.html`
- 打包命令

**构建模式**:

- 完整项目目录结构
- `package.json`
- `vite.config.ts`
- `public/manifest.json`
- `public/app.js`
- `src/*`
- 构建与打包命令

### ⚠️ 关键限制

插件运行在 iframe 中,必须特别注意:

1. UI 布局限制
   - 禁止使用 `alert()` / `confirm()` / `prompt()`
   - `position: fixed` 的 Modal/Dialog 会受 iframe 边界限制
   - 不要自己做“窗口套窗口”的外层大卡片
   - 优先让输入区、结果区、列表区直接填充宿主提供的空间
   - 默认采用极简、键盘优先界面
2. 快捷键跨平台支持
   - 禁止只监听 `e.ctrlKey`
   - 必须同时监听 `e.ctrlKey || e.metaKey`

### 快速检查表

- [ ] 最终产出是 `dist/` 根内容或简单模式直接文件
- [ ] `manifest.json` 和 `app.js` 在输出根级别
- [ ] `.wt` 不包含 `src/`、`node_modules/`、`package.json`
- [ ] 解压后直接打开 `index.html` 能运行
- [ ] 不使用 `alert` / `confirm` / `prompt`
- [ ] 所有快捷键同时监听 `ctrlKey || metaKey`
- [ ] 插件页面不自绘外层边框/窗口壳层
- [ ] 插件作为宿主内嵌片段铺满可用区域
