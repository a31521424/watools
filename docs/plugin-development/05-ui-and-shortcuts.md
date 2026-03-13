# 插件开发 05: UI 与快捷键

## iframe UI 设计限制

插件运行在隔离 iframe 环境中。

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

## 官方风格参考

核心要求:

- 使用扁平化 iframe 内布局,优先 `topbar + workspace + status/footer`
- 外层不要再做“大卡片套小卡片”的窗口感容器
- 页面背景优先浅色中性底,工作区用白色或接近白色的面
- 控件边界依赖 `1px` 细线和轻微聚焦态,不要依赖厚阴影和大圆角
- 按钮优先分为 `primary`、`ghost`、默认边框按钮三层级
- 正文字体优先 `"IBM Plex Sans", "SF Pro Text", "PingFang SC", "Segoe UI", sans-serif`
- 代码/输入字体优先 `"JetBrains Mono", "SFMono-Regular", "SF Mono", "Consolas", monospace`
- 状态提示尽量短,放在底部状态栏或 toast,不要堆大段说明文案
- 浮层/侧栏只在功能必须时出现,默认优先 inline 或贴边抽屉
- 同类插件尽量共享以下 token 命名: `--bg`、`--surface`、`--surface-soft`、`--line`、`--text`、`--muted`、`--accent`

推荐基线 token:

```css
:root {
    --bg: #f3f6fa;
    --surface: #ffffff;
    --surface-soft: #f8fbff;
    --line: rgba(28, 47, 67, 0.12);
    --text: #162433;
    --muted: #627487;
    --accent: #1968ab;
    --accent-soft: rgba(25, 104, 171, 0.12);
}
```

避免:

- 在插件内部再做“窗口中窗口”的大圆角壳
- 大面积渐变、拟物阴影、厚描边
- 为了说明功能而长期占据主要阅读区域的大段文案
- 同一产品里每个官方插件都使用完全不同的字体和控件语言

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
