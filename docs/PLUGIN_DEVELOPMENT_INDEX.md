# WaTools 插件开发索引

这是插件开发的**入口文件**。默认不要继续完整阅读所有模块。

## LLM 必读

先读这一页,再按任务打开最少量的模块。

- 目标产物必须是 `.wt` 包内的可运行文件,不是源码工程
- 新 UI 插件统一使用 `window.pluginContext` 和 `watools:context-ready`
- `seed` 查询参数和 `window.inputValue` 只是兼容层
- 插件运行在 iframe 内,UI 设计和快捷键必须遵守宿主限制
- 不要使用 `alert` / `confirm` / `prompt`
- 快捷键必须同时支持 `Ctrl` 和 `Meta`

## 最小阅读路径

按任务选择:

- 想快速生成一个新插件:
  [`01-llm-must-read.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/01-llm-must-read.md)
  [`02-templates-and-packaging.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/02-templates-and-packaging.md)
- 需要处理输入注入、命令面板带入、剪贴板图片:
  [`03-runtime-and-context.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/03-runtime-and-context.md)
- 需要调用宿主能力、存储、HTTP、剪贴板:
  [`04-api-and-browser.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/04-api-and-browser.md)
- 需要做复杂 UI、弹层、快捷键:
  [`05-ui-and-shortcuts.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/05-ui-and-shortcuts.md)
- 需要完整类型、最终交付检查:
  [`06-reference-and-checklist.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/06-reference-and-checklist.md)

## 模块目录

- [`01-llm-must-read.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/01-llm-must-read.md)
  信任模型、快速指南、关键限制、快速检查表
- [`02-templates-and-packaging.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/02-templates-and-packaging.md)
  简单模式、构建模式、配置、常见打包错误
- [`03-runtime-and-context.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/03-runtime-and-context.md)
  PluginContext、UI 注入协议、运行时上下文
- [`04-api-and-browser.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/04-api-and-browser.md)
  `window.runtime`、`window.watools`、API 包装、浏览器 API 限制
- [`05-ui-and-shortcuts.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/05-ui-and-shortcuts.md)
  iframe UI 设计、弹层、响应式、跨平台快捷键
- [`06-reference-and-checklist.md`](/Users/banbxio/Desktop/watools/DOCS/plugin-development/06-reference-and-checklist.md)
  API 附录、完整类型、最终检查清单

## 兼容性提示

- 插件开发文档的唯一入口现在是本文件
- 如果发现旧引用仍然指向 `PLUGIN_DEVELOPMENT.md`,应改为指向本索引
