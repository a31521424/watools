# WaTools 审查验证材料

这组材料支持 [全仓审查报告](../../review-2026-10-02.md)，基于提交 `a213873`。探针用于证明现有缺陷，部分 Go 探针以“观察到缺陷”为通过条件，不能作为修复后的回归测试直接使用。

## 材料索引

- [runtime-probes.cjs](./runtime-probes.cjs)：对实际 TypeScript 状态模块和官方插件内联函数进行隔离执行，宿主 API 与 DOM 使用模拟实现，不发出外部请求。
- [runtime-probes-output.txt](./runtime-probes-output.txt)：输入防抖、附件状态、插件启用、统计刷盘、JSON 与 query 转换、过期翻译的观察结果。
- [api-probe_test.go.txt](./api-probe_test.go.txt)：超长路径的空指针探针。
- [watcher-probe_test.go.txt](./watcher-probe_test.go.txt)：Stop 后回调向已关闭 channel 发送的探针。
- [update-probe_test.go.txt](./update-probe_test.go.txt)：assetName 路径逃逸与未经验证文件被 Install 接受的探针。后者只在测试二进制不位于 app bundle 时执行，观察到 manual 状态；没有替换应用。
- [db-probe_test.go.txt](./db-probe_test.go.txt)：数据库应用统计转换丢失字段的探针。
- [pnpm-audit-summary.json](./pnpm-audit-summary.json)：审查当日公告、依赖链与受影响版本，不包含用户数据。
- [govulncheck.txt](./govulncheck.txt)：Go 1.26.1 下标准库调用链报告，源码扫描的可达性不能替代实际漏洞利用条件判断。

Go 探针保存为 `.go.txt`，因此不会被正常 `go test ./...` 发现，不会污染产品构建。

## 已完成检查

`pnpm build`、`go test ./...`、`go vet ./...`、`go test -race ./...` 和 Windows amd64 交叉编译均通过。前端构建有 1,588 个文件，主 JS 794.29 kB，gzip 281.77 kB，dist 磁盘占用约 7 MiB。

函数级 Go overlay 探针输出摘要：

```text
OpenFolderStatError: runtime error: invalid memory address or nil pointer dereference
SendAfterStop: send on closed channel
AssetNameEscapesCache: ../probe
InstallUnverifiedFile: accepted with status=manual
UsageConversion: database UsedCount=42 -> UsedCount=0, LastUsedAt present=false
```

## 复现运行时探针

在仓库根目录、已安装 frontend 依赖的环境运行：

```sh
node docs/review-evidence/2026-10-02/runtime-probes.cjs
```

探针读取实际源码并调用 TypeScript transpileModule；模拟状态容器保留 set 与 get 的状态更新行为，不挂载 React 或运行 Wails。输出针对函数级缺陷，完整 WebView 交互需要另外验证。

## 复现 Go 探针

从仓库根目录生成临时 overlay，不修改被审查的源码文件：

```sh
python3 - <<'PY'
from pathlib import Path
import json
import tempfile

root = Path.cwd()
evidence = root / 'docs/review-evidence/2026-10-02'
target = Path(tempfile.mkdtemp(prefix='watools-review-overlay-'))
packages = {
    'api': 'internal/api',
    'watcher': 'internal/command/watcher',
    'update': 'internal/update',
    'db': 'pkg/db',
}
replace = {}
for name, package in packages.items():
    probe = target / (name + '_probe_test.go')
    probe.write_text((evidence / (name + '-probe_test.go.txt')).read_text())
    replace[str(root / package / 'review_probe_test.go')] = str(probe)
overlay = target / 'overlay.json'
overlay.write_text(json.dumps({'Replace': replace}))
print(overlay)
PY
```

将打印出的绝对路径作为 `-overlay` 值，在 macOS 使用匹配仓库的 Go 工具链运行：

```sh
go test -overlay=/打印出的路径/overlay.json -run TestReviewProbe -v ./internal/api ./internal/command/watcher ./internal/update ./pkg/db
```

update 探针只应在普通测试二进制环境运行，不应把测试二进制放到实际 app bundle 中。探针创建的业务文件位于 testing.TempDir；路径拼接探针不写入拼接后的目录。若使用受限文件系统，把 GOCACHE 指向允许写入的临时目录。

## 依赖扫描命令

```sh
pnpm --dir frontend audit --json
govulncheck -mode=source ./...
```

源码扫描以执行扫描时的 Go 工具链为准。升级工具链或依赖后应重新扫描，不能沿用本次数量判断新发布包的安全状态。
