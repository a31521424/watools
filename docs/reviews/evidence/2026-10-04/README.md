# 2026年10月4日 Wails v3 原生验收

[返回证据索引](../README.md) · [验收记录](../../../backlog/native-self-use-acceptance.md)

基于 `99fb0c6` 的本轮修复，在 macOS 26.6.2 / arm64 实际执行。固定版本和最终脱敏原生结果见 [验收输出](./native-acceptance.txt)，构建与源码检查见 [检查摘要](./checks.txt)。材料仅含测试夹具和结果，不包含真实剪贴板或凭据。

## 材料

- [原生验收输出](./native-acceptance.txt)：窗口、快捷键冲突/回退/释放、三轮焦点交接、三类剪贴板、合成 composition、凭据故障、AppKit 菜单与对话框、双屏定位及框架缩放。
- [检查摘要](./checks.txt)：16 个工作流回归、12 个前端回归、Go 全包 race/vet/server、TypeScript/Vite 和生产构建。
- [自己的 WebView 快照](./launcher.png)：输入已清空；只捕获测试窗口内容，没有截取用户桌面，不需要屏幕录制授权。四角透明且内容完整，不能代表跨桌面/显示器的全部视觉场景。
- [临时原生探针](./native-probe.go.txt)：AppKit 元数据、菜单操作、NSOpenPanel、完整剪贴板格式备份/恢复和 WKWebView 快照。
- [临时 Carbon 回调探针](./hotkey-probe.go.txt)：向自己已注册的事件处理器定向发送合成 Carbon 事件，验证真实处理器 → Go listener → 呼出/隐藏，并交错后台注册与主线程清理十轮；不会投递系统全局键盘事件。
- [辅助应用](./focus-helper.m.txt)：自己的焦点参考窗口和独占快捷键占用/释放，不读取其他用户应用内容。
- [并发退出脚本](./rapid-exit.py.txt)：四轮并发配置更新与 AppKit Quit；不改写剪贴板。
- [执行脚本](./native-harness.py.txt)：固定测试路径的历史脚本，已内联当时 MCP client 与绑定 ID 读取逻辑，供复核操作和断言。不是日常开发命令。

## 复现约束

探针以 `.txt` 保存，不能被正常 Go 构建自动纳入。复现时在独立临时目录恢复它们为源文件，通过 Go `-overlay` 把 main 探针映射到虚拟 `main_acceptance_probe_darwin.go`，Carbon 探针映射到虚拟 `internal/app/hotkey_acceptance_probe_darwin.go`；使用 `production,private_mac_apis,mcp,acceptance` tags 构建专用 `.app`。原生服务只监听本机 9247/9248；默认生产应用没有这些测试服务。

历史脚本要求在仓库根目录运行，临时根为 `/private/tmp/watools-v3-native-round2`，测试 bundle 为 `.cache/v3-native-round2/watools-qa.app` 和 `focus.app`。必须准备隔离 SQLite 备份及插件 revision、`data/hotkeys`、独立 cache、synthetic `clipboard.txt`/PNG fixture；不要把脚本指向正式安装。辅助应用用 Cocoa/Carbon 编译并写入测试 `.app`，两份 bundle 都 ad-hoc 签名。修改复现路径时同步脚本、overlay 和 helper 参数。测试结束通过 finally 恢复剪贴板并只清理自己创建的进程。

物理全局快捷键、真实中文输入法组合与候选、手动拖拽/缩放、显示器热插拔、系统首次授权和选择安装/保存的完整流程仍未验证。合成事件、AppKit 操作、WebView 快照与实机人工输入分别记录；未请求 Accessibility 或 Screen Recording 权限。


两位独立 sub-agent 交叉 review 已完成。原生审查发现并关闭退出锁反转风险：所有快捷键生命周期入口先进入 AppKit 主线程，再取得锁；十轮交错清理与四轮并发更新/原生 Quit 均通过。工作流/前端审查未发现阻塞问题。独立审查不代表残留物理验收已完成。

归档复核补齐每次 Refresh/Reload 后的实际 `GetPluginsApi` 调用和六个插件断言，再次完整原生回归通过；`bindingRecovered` 有对应绑定返回证据。完整文档索引与九个材料文件一致，258 个相对链接检查通过。
