# 插件开发 05: UI 与快捷键

## iframe UI 设计限制

插件运行在独立 iframe 文档中；当前是可信插件模型，iframe 不构成安全沙箱。

### Modal / Dialog

问题:

- `position: fixed` 相对于 iframe viewport
- 弹层可能被 iframe 边界裁剪
- `z-index` 无法穿透主应用

推荐:

- 简单场景使用插件内 absolute overlay
- 更稳定的方案是 inline dialog
- 通知优先用 toast

### Dropdown / Popover

- 不要默认只向下展开
- 需要根据 `window.innerHeight` 和触发器位置做智能定位
- 需要限制 `maxHeight` 并允许滚动

### 全屏覆盖层

- `inset: 0` 只能覆盖 iframe 区域,不能覆盖整个 WaTools 主窗口
- 应按 iframe 内局部覆盖来设计

### 响应式

```css
#app {
    width: 100%;
    min-height: 100vh;
    padding: 16px;
    box-sizing: border-box;
}

.fixed-element {
    position: fixed;
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 32px);
}

.full-height {
    height: 100dvh;
}
```

## 官方风格与标题栏

宿主、管理页面和五个官方 UI 插件统一使用 [当前 UI 规范](../design/ui-style.md)。基础样式唯一来源为 `frontend/public/watools-ui.css`；官方打包命令自动把它写入 UI 插件的 `.wt`，旧 revision 保持自己的样式。

官方 UI HTML 引入：

```html
<link rel="stylesheet" href="watools-ui.css">
<body class="wt-plugin">
  <!-- 工具自己的 topbar、workspace、status/footer -->
</body>
```

- 默认采用系统字体、浅灰背景、白色工作面、蓝色主操作和细边框。按钮使用 `primary`、`ghost`、默认边框或 `text-button`。
- 使用共享 `--wt-*` token；兼容 `--bg`、`--surface`、`--surface-soft`、`--line`、`--text`、`--muted`、`--accent` 别名。不要复制维护第二套颜色或字体。
- 基础规则使用低优先级 `:where()`；插件只编写自身工作布局和响应式覆盖，不靠 `!important` 调整尺寸。
- 主窗口展开页的返回、图标、名称、功能简介与分离操作由宿主 `WaWorkspaceHeader` 绘制，空白区域负责窗口拖动。插件不重复添加宿主标题栏。
- 分离后的独立窗口使用原生标题栏，最小窗口为 480×320。按 iframe 实际尺寸设计，工具栏可换行，双栏工作区在 740px 以下改为上下布局，内容允许滚动。
- 状态提示短而明确，底部状态栏或 toast 即可。保留可见键盘焦点和细滚动条。
- 不使用大面积渐变、厚阴影或“窗口套窗口”的容器，不在使用流程暴露 SDK 或内部存储细节。

共享样式变更后需要重新打包并安装所有官方 UI 插件，单独构建宿主不会替换已安装插件。

## 跨平台快捷键

核心要求:

- 不要只监听 `ctrlKey`
- 必须同时支持 `Ctrl` 和 `Meta`

错误写法:

```javascript
if (e.ctrlKey && e.key === 'c') {
    copyToClipboard();
}
```

正确写法:

```javascript
if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
    e.preventDefault();
    copyToClipboard();
}
```

## 推荐快捷键工具

```javascript
const keyboard = {
    match: (e, key, modifiers = {}) => {
        if (e.key.toLowerCase() !== key.toLowerCase()) return false;

        const primaryPressed = e.ctrlKey || e.metaKey;
        if (modifiers.primary && !primaryPressed) return false;
        if (!modifiers.primary && primaryPressed) return false;
        if (modifiers.shift !== undefined && modifiers.shift !== e.shiftKey) return false;
        if (modifiers.alt !== undefined && modifiers.alt !== e.altKey) return false;

        return true;
    }
};
```

## 快捷键提示 UI

```javascript
function getShortcutHint(shortcut) {
    const isMac = /Mac|iPhone|iPod|iPad/.test(navigator.platform);
    const primary = isMac ? '⌘' : 'Ctrl';
    const modifiers = [];

    if (shortcut.primary) modifiers.push(primary);
    if (shortcut.shift) modifiers.push('Shift');
    if (shortcut.alt) modifiers.push(isMac ? '⌥' : 'Alt');

    return [...modifiers, shortcut.key.toUpperCase()].join(' + ');
}
```

## 冲突避免

避免使用:

- `Alt + Space`
- `ESC`
- `Ctrl/Cmd + Q`
- `Ctrl/Cmd + W`

谨慎使用:

- `Ctrl/Cmd + T`
- `Ctrl/Cmd + N`
- `Ctrl/Cmd + R`
