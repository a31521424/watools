import React, {useEffect, useState} from "react"
import {ArrowLeft, Download, ExternalLink, FolderOpen, RefreshCcw, Sparkles} from "lucide-react"
import {useLocation} from "wouter"
import {Button} from "@/components/ui/button"
import {cn} from "@/lib/utils"
import {checkForUpdates, downloadUpdate, installUpdate, type InstallResult, type UpdateInfo} from "@/api/update"
import {OpenFolder} from "../../../wailsjs/go/coordinator/WaAppCoordinator"
import {BrowserOpenURL} from "../../../wailsjs/runtime/runtime"

const SURFACE_STYLE = {
    "--bg": "#f4f7fb",
    "--surface": "#ffffff",
    "--surface-soft": "#f7f9fc",
    "--line": "rgba(15, 23, 42, 0.12)",
    "--text": "#0f172a",
    "--muted": "#5b6b80",
    "--accent": "#0f766e",
    "--accent-soft": "rgba(15, 118, 110, 0.10)",
    fontFamily: "\"IBM Plex Sans\", \"SF Pro Text\", \"PingFang SC\", \"Segoe UI\", sans-serif",
} as React.CSSProperties

const MONO_FONT = "\"JetBrains Mono\", \"SFMono-Regular\", \"SF Mono\", \"Consolas\", monospace"

const formatDateTime = (value?: string) => {
    if (!value) {
        return "--"
    }

    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
        return value
    }

    return new Intl.DateTimeFormat("zh-CN", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
    }).format(date)
}

const formatBytes = (value?: number) => {
    if (!value || value <= 0) {
        return "--"
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

const statusTone = (info: UpdateInfo | null, error: string | null) => {
    if (error) {
        return "border-red-200 bg-red-50 text-red-700"
    }
    if (!info) {
        return "border-[color:var(--line)] bg-[var(--surface)] text-[var(--muted)]"
    }
    if (info.hasUpdate && info.downloadedPath) {
        return "border-emerald-200 bg-emerald-50 text-emerald-700"
    }
    if (info.hasUpdate) {
        return "border-amber-200 bg-amber-50 text-amber-700"
    }
    return "border-slate-200 bg-slate-50 text-slate-700"
}

const statusText = (info: UpdateInfo | null, error: string | null, lastActionMessage: string | null) => {
    if (error) {
        return error
    }
    if (lastActionMessage) {
        return lastActionMessage
    }
    return info?.message || "暂未检查更新"
}

export function WaUpdateManagement() {
    const [_, navigate] = useLocation()
    const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null)
    const [isChecking, setIsChecking] = useState(false)
    const [isDownloading, setIsDownloading] = useState(false)
    const [isInstalling, setIsInstalling] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [lastActionMessage, setLastActionMessage] = useState<string | null>(null)

    const refresh = async () => {
        setIsChecking(true)
        setError(null)
        try {
            const nextInfo = await checkForUpdates()
            setUpdateInfo(nextInfo)
            setLastActionMessage(nextInfo.message || null)
        } catch (nextError) {
            setError(nextError instanceof Error ? nextError.message : "检查更新失败")
        } finally {
            setIsChecking(false)
        }
    }

    useEffect(() => {
        void refresh()
    }, [])

    const handleDownload = async () => {
        setIsDownloading(true)
        setError(null)
        try {
            const nextInfo = await downloadUpdate()
            setUpdateInfo(nextInfo)
            setLastActionMessage(nextInfo.message || "更新包下载完成")
        } catch (nextError) {
            setError(nextError instanceof Error ? nextError.message : "下载更新失败")
        } finally {
            setIsDownloading(false)
        }
    }

    const handleInstall = async () => {
        if (!updateInfo?.downloadedPath) {
            setError("请先下载更新包")
            return
        }
        setIsInstalling(true)
        setError(null)
        try {
            const result: InstallResult = await installUpdate(updateInfo.downloadedPath)
            setLastActionMessage(result.message || "安装流程已启动")
            if (result.requiresManualInstall && result.openPath) {
                await OpenFolder(result.openPath)
            }
        } catch (nextError) {
            setError(nextError instanceof Error ? nextError.message : "安装更新失败")
        } finally {
            setIsInstalling(false)
        }
    }

    const hasUpdate = Boolean(updateInfo?.hasUpdate)
    const hasDownloadedAsset = Boolean(updateInfo?.downloadedPath)
    const releaseUrl = updateInfo?.releaseUrl || "https://github.com/a31521424/watools/releases"

    return (
        <div className="flex h-full min-h-0 flex-col bg-[var(--bg)] text-[var(--text)]" style={SURFACE_STYLE}>
            <div className="border-b border-[color:var(--line)] bg-[var(--surface)] px-6 py-5">
                <div className="flex flex-wrap items-center gap-3">
                    <Button variant="ghost" size="sm" onClick={() => navigate("/")}>
                        <ArrowLeft className="mr-2 h-4 w-4"/>
                        返回
                    </Button>
                    <span className="inline-flex h-8 items-center gap-2 rounded-full border border-transparent bg-[var(--accent-soft)] px-3 text-xs font-medium text-[var(--accent)]">
                        <Sparkles className="h-3.5 w-3.5"/>
                        Self Update
                    </span>
                </div>
                <div className="mt-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
                    <div className="space-y-2">
                        <h1 className="text-2xl font-semibold tracking-tight text-[var(--text)]">应用更新</h1>
                        <p className="max-w-2xl text-sm leading-6 text-[var(--muted)]">
                            这里会检查 GitHub Releases 中的最新稳定版本，并按当前平台下载对应安装包。
                        </p>
                    </div>
                    <div
                        className={cn(
                            "inline-flex min-h-10 items-center rounded-md border px-3 py-2 text-sm",
                            statusTone(updateInfo, error)
                        )}
                    >
                        {statusText(updateInfo, error, lastActionMessage)}
                    </div>
                </div>
            </div>

            <div className="flex-1 overflow-y-auto px-6 py-6">
                <div className="grid gap-4 lg:grid-cols-[1.3fr_0.9fr]">
                    <section className="space-y-4">
                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="border border-[color:var(--line)] bg-[var(--surface)] px-5 py-4">
                                <div className="text-[11px] font-medium uppercase tracking-[0.18em] text-[var(--muted)]">
                                    当前版本
                                </div>
                                <div className="mt-3 text-2xl font-semibold text-[var(--text)]" style={{fontFamily: MONO_FONT}}>
                                    {updateInfo?.currentVersion || "--"}
                                </div>
                            </div>
                            <div className="border border-[color:var(--line)] bg-[var(--surface)] px-5 py-4">
                                <div className="text-[11px] font-medium uppercase tracking-[0.18em] text-[var(--muted)]">
                                    最新版本
                                </div>
                                <div className="mt-3 text-2xl font-semibold text-[var(--text)]" style={{fontFamily: MONO_FONT}}>
                                    {updateInfo?.latestVersion || "--"}
                                </div>
                            </div>
                        </div>

                        <div className="border border-[color:var(--line)] bg-[var(--surface)]">
                            <div className="border-b border-[color:var(--line)] px-5 py-4">
                                <h2 className="text-base font-semibold text-[var(--text)]">平台安装包</h2>
                            </div>
                            <dl className="divide-y divide-[color:var(--line)]">
                                <div className="flex items-center justify-between gap-4 px-5 py-4 text-sm">
                                    <dt className="text-[var(--muted)]">平台</dt>
                                    <dd className="font-medium text-[var(--text)]">{updateInfo?.platformLabel || "--"}</dd>
                                </div>
                                <div className="flex items-center justify-between gap-4 px-5 py-4 text-sm">
                                    <dt className="text-[var(--muted)]">安装包</dt>
                                    <dd className="max-w-[60%] break-all text-right font-medium text-[var(--text)]" style={{fontFamily: MONO_FONT}}>
                                        {updateInfo?.assetName || "--"}
                                    </dd>
                                </div>
                                <div className="flex items-center justify-between gap-4 px-5 py-4 text-sm">
                                    <dt className="text-[var(--muted)]">大小</dt>
                                    <dd className="font-medium text-[var(--text)]">{formatBytes(updateInfo?.assetSize)}</dd>
                                </div>
                                <div className="flex items-center justify-between gap-4 px-5 py-4 text-sm">
                                    <dt className="text-[var(--muted)]">发布时间</dt>
                                    <dd className="font-medium text-[var(--text)]">{formatDateTime(updateInfo?.publishedAt)}</dd>
                                </div>
                                <div className="flex items-center justify-between gap-4 px-5 py-4 text-sm">
                                    <dt className="text-[var(--muted)]">下载状态</dt>
                                    <dd className="font-medium text-[var(--text)]">
                                        {hasDownloadedAsset ? "安装包已缓存" : hasUpdate ? "待下载" : "无需下载"}
                                    </dd>
                                </div>
                            </dl>
                        </div>

                        <div className="border border-[color:var(--line)] bg-[var(--surface)] px-5 py-5">
                            <div className="flex flex-wrap gap-3">
                                <Button onClick={() => void refresh()} disabled={isChecking || isDownloading || isInstalling}>
                                    <RefreshCcw className="mr-2 h-4 w-4"/>
                                    {isChecking ? "检查中..." : "检查更新"}
                                </Button>
                                <Button
                                    variant="outline"
                                    onClick={() => void handleDownload()}
                                    disabled={!hasUpdate || hasDownloadedAsset || isChecking || isDownloading || isInstalling}
                                >
                                    <Download className="mr-2 h-4 w-4"/>
                                    {isDownloading ? "下载中..." : "下载更新"}
                                </Button>
                                <Button
                                    variant="outline"
                                    onClick={() => void handleInstall()}
                                    disabled={!hasDownloadedAsset || isChecking || isDownloading || isInstalling}
                                >
                                    <Sparkles className="mr-2 h-4 w-4"/>
                                    {isInstalling ? "准备安装..." : "安装更新"}
                                </Button>
                                <Button variant="ghost" onClick={() => void BrowserOpenURL(releaseUrl)}>
                                    <ExternalLink className="mr-2 h-4 w-4"/>
                                    打开 Release
                                </Button>
                                {updateInfo?.downloadedPath && (
                                    <Button variant="ghost" onClick={() => void OpenFolder(updateInfo.downloadedPath)}>
                                        <FolderOpen className="mr-2 h-4 w-4"/>
                                        打开下载位置
                                    </Button>
                                )}
                            </div>
                            {updateInfo?.requiresManualInstall && (
                                <p className="mt-4 text-sm leading-6 text-[var(--muted)]">
                                    当前平台或安装位置需要人工接管安装。点击“安装更新”后会自动定位到下载包。
                                </p>
                            )}
                        </div>
                    </section>

                    <aside className="space-y-4">
                        <div className="border border-[color:var(--line)] bg-[var(--surface)]">
                            <div className="border-b border-[color:var(--line)] px-5 py-4">
                                <h2 className="text-base font-semibold text-[var(--text)]">更新说明</h2>
                            </div>
                            <div className="px-5 py-5">
                                <pre className="whitespace-pre-wrap break-words text-sm leading-6 text-[var(--text)]">
                                    {updateInfo?.notes?.trim() || "当前 release 还没有附带额外说明。"}
                                </pre>
                            </div>
                        </div>

                        <div className="border border-[color:var(--line)] bg-[var(--surface-soft)] px-5 py-5 text-sm leading-6 text-[var(--muted)]">
                            <p>Windows 会下载并启动 NSIS 安装器。</p>
                            <p className="mt-2">macOS 会优先尝试替换当前 `.app`；如果目录不可写，则回退为手动替换。</p>
                            <p className="mt-2">更新清单来自 release 附件 `watools_latest.json`，由 CD 工作流自动生成。</p>
                        </div>
                    </aside>
                </div>
            </div>
        </div>
    )
}
