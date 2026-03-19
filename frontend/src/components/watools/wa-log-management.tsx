import React, {useDeferredValue, useEffect, useMemo, useState} from "react"
import {
    AlertCircle,
    ArrowLeft,
    ChevronDown,
    ChevronUp,
    ChevronRight,
    Copy,
    FileText,
    FolderOpen,
    RefreshCcw,
    Search,
} from "lucide-react"
import {useLocation} from "wouter"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Sheet, SheetContent} from "@/components/ui/sheet"
import {cn} from "@/lib/utils"
import {getLogDirectory, listLogFiles, queryLogs} from "@/api/log"
import {LogFileInfo, LogRecord} from "@/schemas/log"
import {OpenFolder} from "../../../wailsjs/go/coordinator/WaAppCoordinator"

const SURFACE_STYLE = {
    "--bg": "#eef2f6",
    "--surface": "#ffffff",
    "--surface-soft": "#f5f7fa",
    "--line": "rgba(15, 23, 42, 0.12)",
    "--text": "#0f172a",
    "--muted": "#64748b",
    "--accent": "#0f172a",
    "--accent-soft": "rgba(15, 23, 42, 0.06)",
    fontFamily: "\"IBM Plex Sans\", \"SF Pro Text\", \"PingFang SC\", \"Segoe UI\", sans-serif",
} as React.CSSProperties

const MONO_FONT = "\"JetBrains Mono\", \"SFMono-Regular\", \"SF Mono\", \"Consolas\", monospace"
const ALL_FILES_VALUE = "__all__"
const DEFAULT_PAGE_SIZE = 200
const LEVEL_OPTIONS = ["all", "trace", "debug", "info", "warn", "error", "fatal"] as const

const formatDateTime = (value?: string) => {
    if (!value) {
        return "--"
    }

    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
        return value
    }

    return new Intl.DateTimeFormat("zh-CN", {
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
    }).format(date)
}

const formatBytes = (value: number) => {
    if (!Number.isFinite(value) || value <= 0) {
        return "0 B"
    }

    const units = ["B", "KB", "MB", "GB"]
    let size = value
    let unitIndex = 0

    while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024
        unitIndex += 1
    }

    return `${size >= 10 || unitIndex === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unitIndex]}`
}

const formatSourceLabel = (value?: string) => {
    switch (value) {
        case "app-backend":
            return "主体后端"
        case "app-frontend":
            return "主体前端"
        case "plugin":
            return "插件"
        case "unknown":
        case "":
        case undefined:
            return "未知"
        default:
            return value
    }
}

const formatLevelLabel = (value?: string) => {
    if (!value) {
        return "raw"
    }
    return value.toUpperCase()
}

const levelBadgeClassName = (value?: string) => {
    switch (value) {
        case "error":
        case "fatal":
            return "border-red-200 bg-red-50 text-red-700"
        case "warn":
            return "border-amber-200 bg-amber-50 text-amber-700"
        case "debug":
        case "trace":
            return "border-slate-200 bg-slate-50 text-slate-700"
        case "info":
            return "border-blue-200 bg-blue-50 text-blue-700"
        default:
            return "border-[color:var(--line)] bg-[var(--surface-soft)] text-[var(--muted)]"
    }
}

const sourceBadgeClassName = (value?: string) => {
    switch (value) {
        case "app-backend":
            return "border-slate-200 bg-slate-50 text-slate-700"
        case "app-frontend":
            return "border-blue-200 bg-blue-50 text-blue-700"
        case "plugin":
            return "border-emerald-200 bg-emerald-50 text-emerald-700"
        default:
            return "border-[color:var(--line)] bg-[var(--surface-soft)] text-[var(--muted)]"
    }
}

const renderRawContent = (record: LogRecord | null) => {
    if (!record) {
        return ""
    }

    if (record.parseStatus === "structured") {
        try {
            return JSON.stringify(JSON.parse(record.raw), null, 2)
        } catch {
            return record.raw
        }
    }

    return record.raw
}

const getSelectedFileQuery = (selectedFile: string, files: LogFileInfo[]) => {
    if (!selectedFile || selectedFile === ALL_FILES_VALUE) {
        return files.map(file => file.name)
    }
    return [selectedFile]
}

type LogDetailsProps = {
    record: LogRecord | null
    onCopy: (record: LogRecord) => Promise<void>
    onClose?: () => void
}

function LogDetails({record, onCopy, onClose}: LogDetailsProps) {
    if (!record) {
        return (
            <div className="flex h-full min-h-[240px] flex-col items-center justify-center gap-3 px-6 text-center">
                <div className="flex h-12 w-12 items-center justify-center border border-[color:var(--line)] bg-[var(--surface-soft)]">
                    <FileText className="h-5 w-5 text-[var(--muted)]"/>
                </div>
                <div className="space-y-1">
                    <h2 className="text-base font-semibold text-[var(--text)]">选择一条日志查看详情</h2>
                    <p className="text-sm text-[var(--muted)]">右侧会展示结构化字段和原始日志内容。</p>
                </div>
            </div>
        )
    }

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="border-b border-[color:var(--line)] px-5 py-4">
                <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                            <span className={cn("inline-flex h-6 items-center rounded-sm border px-2 text-[11px] font-medium", levelBadgeClassName(record.level))}>
                                {formatLevelLabel(record.level)}
                            </span>
                            <span className="inline-flex h-6 items-center rounded-sm border border-[color:var(--line)] bg-[var(--surface-soft)] px-2 text-[11px] font-medium text-[var(--muted)]">
                                {formatSourceLabel(record.source)}
                            </span>
                            <span className="inline-flex h-6 items-center rounded-sm border border-[color:var(--line)] bg-[var(--surface)] px-2 text-[11px] font-medium text-[var(--muted)]">
                                {record.parseStatus}
                            </span>
                        </div>
                        <div className="space-y-1">
                            <h2 className="break-words text-base font-semibold text-[var(--text)]">
                                {record.message || "(empty message)"}
                            </h2>
                            <p className="text-xs leading-5 text-[var(--muted)]">
                                {record.file}:{record.line} · {formatDateTime(record.timestamp)}
                            </p>
                        </div>
                    </div>

                    {onClose && (
                        <Button variant="ghost" size="sm" onClick={onClose} className="shrink-0">
                            关闭
                        </Button>
                    )}
                </div>
            </div>

            <div className="flex-1 overflow-y-auto px-5 py-5">
                <div className="grid gap-3 sm:grid-cols-2">
                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-3">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            文件位置
                        </div>
                        <div className="break-all text-sm text-[var(--text)]" style={{fontFamily: MONO_FONT}}>
                            {record.filePath}:{record.line}
                        </div>
                    </div>

                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-3">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            插件标识
                        </div>
                        <div className="break-all text-sm text-[var(--text)]" style={{fontFamily: MONO_FONT}}>
                            {record.pluginId || "--"}
                        </div>
                    </div>

                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-3">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            时间
                        </div>
                        <div className="text-sm text-[var(--text)]">{formatDateTime(record.timestamp)}</div>
                    </div>

                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-3">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            错误字段
                        </div>
                        <div className="break-all text-sm text-[var(--text)]">{record.error || "--"}</div>
                    </div>
                </div>

                <div className="mt-4 border border-[color:var(--line)] bg-[var(--surface)]">
                    <div className="border-b border-[color:var(--line)] px-4 py-3">
                        <h3 className="text-sm font-semibold text-[var(--text)]">解析字段</h3>
                    </div>
                    <dl className="divide-y divide-[color:var(--line)]">
                        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
                            <dt className="text-[var(--muted)]">级别</dt>
                            <dd className="font-medium text-[var(--text)]">{record.level || "--"}</dd>
                        </div>
                        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
                            <dt className="text-[var(--muted)]">来源</dt>
                            <dd className="font-medium text-[var(--text)]">{formatSourceLabel(record.source)}</dd>
                        </div>
                        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
                            <dt className="text-[var(--muted)]">解析状态</dt>
                            <dd className="font-medium text-[var(--text)]">{record.parseStatus}</dd>
                        </div>
                        {record.fields && Object.entries(record.fields).map(([key, value]) => (
                            <div key={key} className="flex items-start justify-between gap-4 px-4 py-3 text-sm">
                                <dt className="text-[var(--muted)]">{key}</dt>
                                <dd className="max-w-[60%] break-all text-right font-medium text-[var(--text)]">{value}</dd>
                            </div>
                        ))}
                    </dl>
                </div>

                <div className="mt-4 border border-[color:var(--line)] bg-[var(--surface)]">
                    <div className="flex items-center justify-between border-b border-[color:var(--line)] px-4 py-3">
                        <h3 className="text-sm font-semibold text-[var(--text)]">原始日志</h3>
                        <Button variant="ghost" size="sm" onClick={() => void onCopy(record)}>
                            <Copy className="mr-2 h-4 w-4"/>
                            复制原文
                        </Button>
                    </div>
                    <pre
                        className="max-h-[420px] overflow-auto whitespace-pre-wrap break-all px-4 py-4 text-xs text-[var(--text)]"
                        style={{fontFamily: MONO_FONT}}
                    >
                        {renderRawContent(record)}
                    </pre>
                </div>
            </div>
        </div>
    )
}

export function WaLogManagement() {
    const [, navigate] = useLocation()
    const [logDirectory, setLogDirectory] = useState("")
    const [files, setFiles] = useState<LogFileInfo[]>([])
    const [selectedFile, setSelectedFile] = useState("")
    const [searchValue, setSearchValue] = useState("")
    const deferredSearchValue = useDeferredValue(searchValue.trim())
    const [levelFilter, setLevelFilter] = useState<string>("all")
    const [sourceFilter, setSourceFilter] = useState<string>("all")
    const [pluginFilter, setPluginFilter] = useState<string>("all")
    const [records, setRecords] = useState<LogRecord[]>([])
    const [selectedRecord, setSelectedRecord] = useState<LogRecord | null>(null)
    const [availableSources, setAvailableSources] = useState<string[]>([])
    const [availablePluginIds, setAvailablePluginIds] = useState<string[]>([])
    const [nextCursor, setNextCursor] = useState<string | null>(null)
    const [totalMatched, setTotalMatched] = useState(0)
    const [error, setError] = useState<string | null>(null)
    const [isLoadingFiles, setIsLoadingFiles] = useState(false)
    const [isLoadingLogs, setIsLoadingLogs] = useState(false)
    const [isLoadingMore, setIsLoadingMore] = useState(false)
    const [isDrawerOpen, setIsDrawerOpen] = useState(false)
    const [isFiltersOpen, setIsFiltersOpen] = useState(false)

    const selectedFileInfo = useMemo(() => {
        return files.find(file => file.name === selectedFile) || null
    }, [files, selectedFile])

    const fileQuery = useMemo(() => getSelectedFileQuery(selectedFile, files), [selectedFile, files])

    const loadLogFiles = async () => {
        setIsLoadingFiles(true)
        setError(null)

        try {
            const [directory, fileList] = await Promise.all([getLogDirectory(), listLogFiles()])
            setLogDirectory(directory)
            setFiles(fileList)
            setSelectedFile(current => {
                if (current && (current === ALL_FILES_VALUE || fileList.some(file => file.name === current))) {
                    return current
                }
                return fileList[0]?.name || ALL_FILES_VALUE
            })
        } catch (loadError) {
            setError(loadError instanceof Error ? loadError.message : "Failed to load log files")
            setFiles([])
            setSelectedFile(ALL_FILES_VALUE)
        } finally {
            setIsLoadingFiles(false)
        }
    }

    const loadLogs = async ({cursor, append}: { cursor?: string; append?: boolean } = {}) => {
        if (append) {
            setIsLoadingMore(true)
        } else {
            setIsLoadingLogs(true)
            setError(null)
        }

        try {
            const result = await queryLogs({
                files: fileQuery,
                keyword: deferredSearchValue,
                levels: levelFilter === "all" ? [] : [levelFilter],
                sources: sourceFilter === "all" ? [] : [sourceFilter],
                pluginIds: pluginFilter === "all" ? [] : [pluginFilter],
                cursor,
                limit: DEFAULT_PAGE_SIZE,
            })

            let nextRecords: LogRecord[] = []
            setRecords(current => {
                nextRecords = append ? [...current, ...result.records] : result.records
                return nextRecords
            })
            setAvailableSources(result.availableSources)
            setAvailablePluginIds(result.availablePluginIds)
            setNextCursor(result.nextCursor || null)
            setTotalMatched(result.totalMatched)
            setSelectedRecord(current => {
                if (!nextRecords.length) {
                    return null
                }
                if (!current) {
                    return nextRecords[0]
                }
                return nextRecords.find(record => record.id === current.id) || nextRecords[0]
            })
        } catch (loadError) {
            setError(loadError instanceof Error ? loadError.message : "Failed to query logs")
            if (!append) {
                setRecords([])
                setSelectedRecord(null)
                setNextCursor(null)
                setTotalMatched(0)
            }
        } finally {
            setIsLoadingLogs(false)
            setIsLoadingMore(false)
        }
    }

    useEffect(() => {
        void loadLogFiles()
    }, [])

    useEffect(() => {
        const handleHotkey = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                navigate("/")
            }
        }

        window.addEventListener("keydown", handleHotkey)
        return () => window.removeEventListener("keydown", handleHotkey)
    }, [navigate])

    useEffect(() => {
        if (files.length === 0) {
            setRecords([])
            setSelectedRecord(null)
            setAvailableSources([])
            setAvailablePluginIds([])
            setNextCursor(null)
            setTotalMatched(0)
            return
        }

        void loadLogs()
    }, [files, fileQuery, deferredSearchValue, levelFilter, sourceFilter, pluginFilter])

    useEffect(() => {
        if (sourceFilter !== "all" && !availableSources.includes(sourceFilter)) {
            setSourceFilter("all")
        }
    }, [availableSources, sourceFilter])

    useEffect(() => {
        if (pluginFilter !== "all" && !availablePluginIds.includes(pluginFilter)) {
            setPluginFilter("all")
        }
    }, [availablePluginIds, pluginFilter])

    const handleOpenRecord = (record: LogRecord) => {
        setSelectedRecord(record)
        setIsDrawerOpen(true)
    }

    const handleCopyRecord = async (record: LogRecord) => {
        if (!navigator.clipboard?.writeText) {
            return
        }
        await navigator.clipboard.writeText(record.raw)
    }

    const handleOpenLogDirectory = () => {
        if (!logDirectory) {
            return
        }
        void OpenFolder(logDirectory)
    }

    const handleLoadMore = async () => {
        if (!nextCursor) {
            return
        }
        await loadLogs({cursor: nextCursor, append: true})
    }

    return (
        <div
            className="flex h-full min-h-0 w-full flex-col overflow-hidden bg-[var(--bg)] text-[var(--text)]"
            style={SURFACE_STYLE}
        >
            <div className="border-b border-[color:var(--line)] bg-[var(--surface)] px-5 py-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div className="min-w-0 space-y-2">
                        <Button variant="ghost" size="sm" onClick={() => navigate("/")} className="h-8 px-2 text-[var(--muted)]">
                            <ArrowLeft className="mr-2 h-4 w-4"/>
                            返回命令面板
                        </Button>
                        <div className="space-y-0.5">
                            <h1 className="text-lg font-semibold">日志管理器</h1>
                            <p className="text-xs text-[var(--muted)]">
                                面向排障的文件日志查看器。
                            </p>
                        </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                        <Button
                            variant="outline"
                            onClick={() => setIsFiltersOpen(open => !open)}
                            className="min-w-[108px]"
                        >
                            {isFiltersOpen ? <ChevronUp className="mr-2 h-4 w-4"/> : <ChevronDown className="mr-2 h-4 w-4"/>}
                            {isFiltersOpen ? "收起过滤器" : "展开过滤器"}
                        </Button>
                        <Button variant="outline" onClick={() => void loadLogFiles()} disabled={isLoadingFiles || isLoadingLogs}>
                            <RefreshCcw className="mr-2 h-4 w-4"/>
                            刷新日志
                        </Button>
                        <Button variant="outline" onClick={handleOpenLogDirectory} disabled={!logDirectory}>
                            <FolderOpen className="mr-2 h-4 w-4"/>
                            打开日志目录
                        </Button>
                    </div>
                </div>

                <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[var(--muted)]" style={{fontFamily: MONO_FONT}}>
                    <span>files={files.length}</span>
                    <span>matched={totalMatched}</span>
                    <span>{selectedFileInfo ? `${selectedFileInfo.name} ${formatBytes(selectedFileInfo.size)}` : "all files"}</span>
                </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col">
                <div className="border-b border-[color:var(--line)] bg-[var(--surface)] px-5 py-3">
                    <div className="flex flex-col gap-3">
                        <div className="relative">
                            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--muted)]"/>
                            <Input
                                value={searchValue}
                                onChange={event => setSearchValue(event.target.value)}
                                placeholder="搜索 message、错误字段、插件 ID 或原始日志"
                                className="h-9 border-[color:var(--line)] bg-[var(--surface)] pl-10 shadow-none"
                            />
                        </div>

                        {isFiltersOpen && (
                            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                                <label className="space-y-1 text-sm text-[var(--muted)]">
                                <span>日志文件</span>
                                <select
                                    value={selectedFile}
                                    onChange={event => setSelectedFile(event.target.value)}
                                    className="h-10 w-full rounded-md border border-[color:var(--line)] bg-[var(--surface)] px-3 text-sm text-[var(--text)] outline-none"
                                >
                                    {files.length > 1 && <option value={ALL_FILES_VALUE}>全部日志文件</option>}
                                    {files.map(file => (
                                        <option key={file.name} value={file.name}>
                                            {file.name}
                                        </option>
                                    ))}
                                </select>
                                </label>

                                <label className="space-y-1 text-sm text-[var(--muted)]">
                                <span>级别</span>
                                <select
                                    value={levelFilter}
                                    onChange={event => setLevelFilter(event.target.value)}
                                    className="h-10 w-full rounded-md border border-[color:var(--line)] bg-[var(--surface)] px-3 text-sm text-[var(--text)] outline-none"
                                >
                                    {LEVEL_OPTIONS.map(option => (
                                        <option key={option} value={option}>
                                            {option === "all" ? "全部级别" : option.toUpperCase()}
                                        </option>
                                    ))}
                                </select>
                                </label>

                                <label className="space-y-1 text-sm text-[var(--muted)]">
                                <span>来源</span>
                                <select
                                    value={sourceFilter}
                                    onChange={event => setSourceFilter(event.target.value)}
                                    className="h-10 w-full rounded-md border border-[color:var(--line)] bg-[var(--surface)] px-3 text-sm text-[var(--text)] outline-none"
                                >
                                    <option value="all">全部来源</option>
                                    {availableSources.map(source => (
                                        <option key={source} value={source}>
                                            {formatSourceLabel(source)}
                                        </option>
                                    ))}
                                </select>
                                </label>

                                <label className="space-y-1 text-sm text-[var(--muted)]">
                                <span>插件</span>
                                <select
                                    value={pluginFilter}
                                    onChange={event => setPluginFilter(event.target.value)}
                                    className="h-10 w-full rounded-md border border-[color:var(--line)] bg-[var(--surface)] px-3 text-sm text-[var(--text)] outline-none"
                                >
                                    <option value="all">全部插件</option>
                                    {availablePluginIds.map(pluginId => (
                                        <option key={pluginId} value={pluginId}>
                                            {pluginId}
                                        </option>
                                    ))}
                                </select>
                                </label>
                            </div>
                        )}
                    </div>
                </div>

                <div className="min-h-0 flex-1 overflow-y-auto bg-[var(--surface)]">
                    {error ? (
                        <div className="flex h-full min-h-[220px] flex-col items-center justify-center gap-3 px-6 text-center">
                            <div className="flex h-12 w-12 items-center justify-center border border-red-200 bg-red-50">
                                <AlertCircle className="h-5 w-5 text-red-600"/>
                            </div>
                            <div className="space-y-1">
                                <h2 className="text-base font-semibold text-[var(--text)]">日志读取失败</h2>
                                <p className="text-sm text-[var(--muted)]">{error}</p>
                            </div>
                        </div>
                    ) : isLoadingFiles || isLoadingLogs ? (
                        <div className="flex h-full min-h-[220px] items-center justify-center px-6 text-sm text-[var(--muted)]">
                            正在加载日志...
                        </div>
                    ) : files.length === 0 ? (
                        <div className="flex h-full min-h-[220px] flex-col items-center justify-center gap-3 px-6 text-center">
                            <div className="flex h-12 w-12 items-center justify-center border border-[color:var(--line)] bg-[var(--surface-soft)]">
                                <FileText className="h-5 w-5 text-[var(--muted)]"/>
                            </div>
                            <div className="space-y-1">
                                <h2 className="text-base font-semibold text-[var(--text)]">还没有日志文件</h2>
                                <p className="text-sm text-[var(--muted)]">应用产生日志后，这里会自动显示可选文件。</p>
                            </div>
                        </div>
                    ) : records.length === 0 ? (
                        <div className="flex h-full min-h-[220px] flex-col items-center justify-center gap-3 px-6 text-center">
                            <div className="flex h-12 w-12 items-center justify-center border border-[color:var(--line)] bg-[var(--surface-soft)]">
                                <Search className="h-5 w-5 text-[var(--muted)]"/>
                            </div>
                            <div className="space-y-1">
                                <h2 className="text-base font-semibold text-[var(--text)]">没有匹配的日志</h2>
                                <p className="text-sm text-[var(--muted)]">调整搜索关键词或筛选条件后再试。</p>
                            </div>
                        </div>
                    ) : (
                        <div className="flex min-h-full flex-col">
                            <div
                                className="grid items-center gap-3 border-b border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-2 text-[11px] uppercase tracking-[0.12em] text-[var(--muted)]"
                                style={{gridTemplateColumns: "90px 78px 96px minmax(180px,220px) 132px minmax(0,1fr)", fontFamily: MONO_FONT}}
                            >
                                <span>Time</span>
                                <span>Level</span>
                                <span>Source</span>
                                <span>Plugin</span>
                                <span>File</span>
                                <span>Message</span>
                            </div>

                            <div className="divide-y divide-[color:var(--line)]">
                                {records.map(record => {
                                    const isSelected = selectedRecord?.id === record.id

                                    return (
                                        <button
                                            key={record.id}
                                            type="button"
                                            className={cn(
                                                "grid w-full items-center gap-3 px-4 py-2 text-left text-xs transition-colors hover:bg-[var(--accent-soft)]",
                                                isSelected && "bg-[var(--accent-soft)]"
                                            )}
                                            style={{gridTemplateColumns: "90px 78px 96px minmax(180px,220px) 132px minmax(0,1fr)", fontFamily: MONO_FONT}}
                                            onClick={() => handleOpenRecord(record)}
                                        >
                                            <span className="truncate text-[var(--muted)]">
                                                {formatDateTime(record.timestamp)}
                                            </span>
                                            <span className={cn("inline-flex h-6 w-fit items-center rounded-sm border px-2 text-[10px] font-medium", levelBadgeClassName(record.level))}>
                                                {formatLevelLabel(record.level)}
                                            </span>
                                            <span className={cn("inline-flex h-6 w-fit items-center rounded-sm border px-2 text-[10px] font-medium", sourceBadgeClassName(record.source))}>
                                                {formatSourceLabel(record.source)}
                                            </span>
                                            <span className="truncate text-[var(--muted)]">
                                                {record.pluginId || "--"}
                                            </span>
                                            <span className="truncate text-[var(--muted)]">
                                                {record.file}:{record.line}
                                            </span>
                                            <span className="flex min-w-0 items-center gap-2">
                                                <span className="truncate font-sans text-[13px] text-[var(--text)]">
                                                    {record.message || "(empty message)"}
                                                </span>
                                                <ChevronRight className="h-4 w-4 shrink-0 text-[var(--muted)]"/>
                                            </span>
                                        </button>
                                    )
                                })}

                                {nextCursor && (
                                    <div className="px-4 py-3">
                                        <Button
                                            variant="outline"
                                            className="w-full"
                                            onClick={() => void handleLoadMore()}
                                            disabled={isLoadingMore}
                                        >
                                            {isLoadingMore ? "正在加载更多..." : "加载更多"}
                                        </Button>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}
                </div>
            </div>

            <Sheet
                open={isDrawerOpen}
                onOpenChange={setIsDrawerOpen}
                className="w-full max-w-[760px] border-l border-[color:var(--line)]"
            >
                <SheetContent className="p-0">
                    <LogDetails
                        record={selectedRecord}
                        onCopy={handleCopyRecord}
                        onClose={() => setIsDrawerOpen(false)}
                    />
                </SheetContent>
            </Sheet>
        </div>
    )
}
