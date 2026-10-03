# 2026年10月3日修复与迁移验证材料

[返回证据索引](../README.md) · [修复记录](../../remediation-2026-10-03.md)

下列旧输出记录已知问题修复时的检查，不是 Wails v3、小窗或后续 Keychain 移除的专门验证材料。边界和后续本地检查说明见修复记录；本次文档整理不重新解释旧输出。

## 材料目录

- [watools-final-go-test.txt](./watools-final-go-test.txt)：Go 测试。
- [watools-final-go-race.txt](./watools-final-go-race.txt)：Go race。
- [watools-final-go-vet.txt](./watools-final-go-vet.txt)：静态检查。
- [watools-final-vuln-macos.txt](./watools-final-vuln-macos.txt)：macOS 生产代码漏洞扫描。
- [watools-final-vuln-windows.txt](./watools-final-vuln-windows.txt)：Windows 生产代码漏洞扫描。
- [watools-final-windows-build.txt](./watools-final-windows-build.txt)：Windows 交叉编译。
- [watools-final-wails-build.txt](./watools-final-wails-build.txt)：macOS Wails 应用构建。
- [macos-production-build.txt](./macos-production-build.txt)：macOS 生产编译。
- [frontend-tests.txt](./frontend-tests.txt)：前端回归。
- [frontend-build.txt](./frontend-build.txt)：前端构建。
- [frontend-audit.json](./frontend-audit.json)：依赖扫描。
- [official-package.txt](./official-package.txt)：官方插件打包。

编译、源码扫描与函数测试不替代 GUI、真实权限、Windows 实机或更新重启验收。


## Wails v3 迁移材料

- [wails-v3-acceptance.txt](./wails-v3-acceptance.txt)：本轮迁移的脱敏检查、构建、原生 WebKit/server 实际回归与两位独立 review 摘要，包含并发剪贴板及重复启动输出。完整范围与回退边界见 [迁移记录](../../../backlog/wails-v3-migration.md)。

这份新增材料与上面的 v2 修复输出分开；物理输入法、全局快捷键、权限及真实更新重启仍未验收。
