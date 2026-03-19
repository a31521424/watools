import React, {useEffect, useMemo, useState} from 'react'
import {ArrowLeft, ExternalLink, Package2, Plus, Puzzle, RefreshCcw, Search} from 'lucide-react'
import {useLocation} from 'wouter'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle} from '@/components/ui/sheet'
import {Switch} from '@/components/ui/switch'
import {cn} from '@/lib/utils'
import {Plugin} from '@/schemas/plugin'
import {usePluginStore} from '@/stores/pluginStore'
import {InstallPluginByFileDialogApi} from '../../../wailsjs/go/coordinator/WaAppCoordinator'
import {BrowserOpenURL} from '../../../wailsjs/runtime/runtime'

const SURFACE_STYLE = {
    '--bg': '#f3f6fa',
    '--surface': '#ffffff',
    '--surface-soft': '#f8fbff',
    '--line': 'rgba(28, 47, 67, 0.12)',
    '--text': '#162433',
    '--muted': '#627487',
    '--accent': '#1968ab',
    '--accent-soft': 'rgba(25, 104, 171, 0.12)',
    fontFamily: '"IBM Plex Sans", "SF Pro Text", "PingFang SC", "Segoe UI", sans-serif',
} as React.CSSProperties

const MONO_FONT = '"JetBrains Mono", "SFMono-Regular", "SF Mono", "Consolas", monospace'

const formatLastUsedAt = (value: Date | null) => {
    if (!value || Number.isNaN(value.getTime()) || value.getTime() <= 0) {
        return '从未使用'
    }

    return new Intl.DateTimeFormat('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
    }).format(value)
}

type PluginDetailsProps = {
    plugin: Plugin
    onToggle: (plugin: Plugin) => Promise<void>
    onUninstall: (plugin: Plugin) => Promise<void>
    onOpenHomepage: (plugin: Plugin) => void
    onClose?: () => void
}

type PluginHelpSectionProps = {
    title: string
    items: string[]
    mono?: boolean
}

function PluginHelpSection({title, items, mono = false}: PluginHelpSectionProps) {
    if (items.length === 0) {
        return null
    }

    return (
        <div className="border border-[color:var(--line)] bg-[var(--surface)]">
            <div className="border-b border-[color:var(--line)] px-4 py-3">
                <h3 className="text-sm font-semibold text-[var(--text)]">{title}</h3>
            </div>
            <div className="divide-y divide-[color:var(--line)]">
                {items.map(item => (
                    <div
                        key={item}
                        className={cn(
                            "px-4 py-3 text-sm text-[var(--text)]",
                            mono && "break-all"
                        )}
                        style={mono ? {fontFamily: MONO_FONT} : undefined}
                    >
                        {item}
                    </div>
                ))}
            </div>
        </div>
    )
}

function PluginDetails({plugin, onToggle, onUninstall, onOpenHomepage, onClose}: PluginDetailsProps) {
    const hasQuickStart = Boolean(plugin.usage) || plugin.triggerKeywords.length > 0

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="border-b border-[color:var(--line)] px-5 py-5">
                <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                            <span
                                className={cn(
                                    'inline-flex h-6 items-center rounded-sm border px-2 text-[11px] font-medium',
                                    plugin.enabled
                                        ? 'border-transparent bg-[var(--accent-soft)] text-[var(--accent)]'
                                        : 'border-[color:var(--line)] bg-[var(--surface)] text-[var(--muted)]'
                                )}
                            >
                                {plugin.enabled ? '已启用' : '已停用'}
                            </span>
                            {plugin.uiEnabled && (
                                <span className="inline-flex h-6 items-center rounded-sm border border-[color:var(--line)] bg-[var(--surface-soft)] px-2 text-[11px] font-medium text-[var(--muted)]">
                                    UI 插件
                                </span>
                            )}
                            <span className="inline-flex h-6 items-center rounded-sm border border-[color:var(--line)] bg-[var(--surface)] px-2 text-[11px] font-medium text-[var(--muted)]">
                                v{plugin.version || '未标注'}
                            </span>
                        </div>
                        <div className="space-y-1">
                            <h2 className="text-xl font-semibold tracking-tight text-[var(--text)]">
                                {plugin.name || '未命名插件'}
                            </h2>
                            <p className="text-sm leading-6 text-[var(--muted)]">
                                {plugin.description || '这个插件还没有提供描述。'}
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
                <div className="grid gap-5 sm:grid-cols-2">
                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-4">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            Package ID
                        </div>
                        <div className="break-all text-sm text-[var(--text)]" style={{fontFamily: MONO_FONT}}>
                            {plugin.packageId}
                        </div>
                    </div>

                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-4">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            作者
                        </div>
                        <div className="text-sm text-[var(--text)]">{plugin.author || '未标注'}</div>
                    </div>

                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-4">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            使用次数
                        </div>
                        <div className="text-sm text-[var(--text)]">{plugin.usedCount} 次</div>
                    </div>

                    <div className="space-y-2 border border-[color:var(--line)] bg-[var(--surface-soft)] px-4 py-4">
                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                            最近使用
                        </div>
                        <div className="text-sm text-[var(--text)]">{formatLastUsedAt(plugin.lastUsedAt)}</div>
                    </div>
                </div>

                <div className="mt-5 border border-[color:var(--line)] bg-[var(--surface)]">
                    <div className="border-b border-[color:var(--line)] px-4 py-3">
                        <h3 className="text-sm font-semibold text-[var(--text)]">运行信息</h3>
                    </div>
                    <dl className="divide-y divide-[color:var(--line)]">
                        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
                            <dt className="text-[var(--muted)]">状态</dt>
                            <dd className="font-medium text-[var(--text)]">
                                {plugin.enabled ? '已加载，可参与命令匹配' : '已禁用，不参与命令匹配'}
                            </dd>
                        </div>
                        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
                            <dt className="text-[var(--muted)]">入口加载</dt>
                            <dd className="font-medium text-[var(--text)]">
                                {plugin.enabled ? `${plugin.entry.length} 个入口已加载` : '启用后加载入口'}
                            </dd>
                        </div>
                        <div className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
                            <dt className="text-[var(--muted)]">UI 能力</dt>
                            <dd className="font-medium text-[var(--text)]">
                                {plugin.uiEnabled ? '包含 UI 界面' : '仅命令式插件'}
                            </dd>
                        </div>
                    </dl>
                </div>

                <div className="mt-5 space-y-5">
                    {hasQuickStart && (
                        <div className="border border-[color:var(--line)] bg-[var(--surface)]">
                            <div className="border-b border-[color:var(--line)] px-4 py-3">
                                <h3 className="text-sm font-semibold text-[var(--text)]">快速上手</h3>
                            </div>
                            <div className="space-y-4 px-4 py-4">
                                {plugin.usage && (
                                    <p className="text-sm leading-6 text-[var(--text)]">
                                        {plugin.usage}
                                    </p>
                                )}
                                {plugin.triggerKeywords.length > 0 && (
                                    <div className="space-y-2">
                                        <div className="text-[11px] font-medium uppercase tracking-[0.16em] text-[var(--muted)]">
                                            触发词
                                        </div>
                                        <div className="flex flex-wrap gap-2">
                                            {plugin.triggerKeywords.map(keyword => (
                                                <span
                                                    key={keyword}
                                                    className="inline-flex items-center rounded-sm border border-[color:var(--line)] bg-[var(--surface-soft)] px-2 py-1 text-xs text-[var(--text)]"
                                                    style={{fontFamily: MONO_FONT}}
                                                >
                                                    {keyword}
                                                </span>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    <PluginHelpSection
                        title="输入示例"
                        items={plugin.usageExamples}
                        mono
                    />
                    <PluginHelpSection title="快捷键" items={plugin.shortcuts}/>
                    <PluginHelpSection title="备注" items={plugin.notes}/>
                </div>
            </div>

            <div className="border-t border-[color:var(--line)] px-5 py-4">
                <div className="flex flex-wrap items-center gap-2">
                    <Button variant="outline" onClick={() => void onToggle(plugin)}>
                        {plugin.enabled ? '停用插件' : '启用插件'}
                    </Button>
                    {plugin.homeUrl && (
                        <Button variant="ghost" onClick={() => onOpenHomepage(plugin)}>
                            <ExternalLink className="mr-2 h-4 w-4"/>
                            打开主页
                        </Button>
                    )}
                    <Button variant="destructive" onClick={() => void onUninstall(plugin)} className="ml-auto">
                        卸载插件
                    </Button>
                </div>
            </div>
        </div>
    )
}

export function WaPluginManagement() {
    const plugins = usePluginStore(state => state.plugins)
    const isLoading = usePluginStore(state => state.isLoading)
    const error = usePluginStore(state => state.error)
    const refreshPlugins = usePluginStore(state => state.refreshPlugins)
    const togglePlugin = usePluginStore(state => state.togglePlugin)
    const uninstallPlugin = usePluginStore(state => state.uninstallPlugin)

    const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null)
    const [isDrawerOpen, setIsDrawerOpen] = useState(false)
    const [searchValue, setSearchValue] = useState('')
    const [, navigate] = useLocation()

    useEffect(() => {
        const handleHotkey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                navigate('/')
            }
        }

        window.addEventListener('keydown', handleHotkey)
        return () => {
            window.removeEventListener('keydown', handleHotkey)
        }
    }, [navigate])

    const sortedPlugins = useMemo(() => {
        return [...plugins].sort((left, right) => {
            if (left.enabled !== right.enabled) {
                return Number(right.enabled) - Number(left.enabled)
            }

            if (left.usedCount !== right.usedCount) {
                return right.usedCount - left.usedCount
            }

            return left.name.localeCompare(right.name, 'zh-CN')
        })
    }, [plugins])

    const filteredPlugins = useMemo(() => {
        const keyword = searchValue.trim().toLowerCase()
        if (!keyword) {
            return sortedPlugins
        }

        return sortedPlugins.filter(plugin => {
            const searchableText = [
                plugin.name,
                plugin.description,
                plugin.author,
                plugin.packageId,
            ].join(' ').toLowerCase()

            return searchableText.includes(keyword)
        })
    }, [searchValue, sortedPlugins])

    useEffect(() => {
        if (filteredPlugins.length === 0) {
            setSelectedPlugin(null)
            setIsDrawerOpen(false)
            return
        }

        setSelectedPlugin(previous => {
            if (!previous) {
                return filteredPlugins[0]
            }

            const prefersVisibleSelection = searchValue.trim().length > 0
            const visibleMatch = filteredPlugins.find(plugin => plugin.packageId === previous.packageId)
            const globalMatch = sortedPlugins.find(plugin => plugin.packageId === previous.packageId)

            if (prefersVisibleSelection) {
                return visibleMatch ?? filteredPlugins[0]
            }

            return globalMatch ?? filteredPlugins[0]
        })
    }, [filteredPlugins, searchValue, sortedPlugins])

    const enabledCount = useMemo(
        () => plugins.filter(plugin => plugin.enabled).length,
        [plugins]
    )

    const uiPluginCount = useMemo(
        () => plugins.filter(plugin => plugin.uiEnabled).length,
        [plugins]
    )

    const handleTogglePlugin = async (plugin: Plugin) => {
        try {
            await togglePlugin(plugin.packageId, !plugin.enabled)
        } catch (toggleError) {
            console.error('Failed to toggle plugin:', toggleError)
        }
    }

    const handleUninstallPlugin = async (plugin: Plugin) => {
        try {
            await uninstallPlugin(plugin.packageId)
            setIsDrawerOpen(false)
        } catch (uninstallError) {
            console.error('Failed to uninstall plugin:', uninstallError)
        }
    }

    const handleInstallPlugin = async () => {
        try {
            await InstallPluginByFileDialogApi()
            await refreshPlugins()
        } catch (installError) {
            console.error('Failed to install plugin:', installError)
        }
    }

    const handleRefreshPlugins = async () => {
        try {
            await refreshPlugins()
        } catch (refreshError) {
            console.error('Failed to refresh plugins:', refreshError)
        }
    }

    const handleOpenPluginDetails = (plugin: Plugin) => {
        setSelectedPlugin(plugin)
        setIsDrawerOpen(true)
    }

    const handleOpenHomepage = (plugin: Plugin) => {
        if (!plugin.homeUrl) {
            return
        }

        BrowserOpenURL(plugin.homeUrl)
    }

    return (
        <div
            className="flex h-full min-h-0 w-full flex-col overflow-hidden bg-[var(--bg)] text-[var(--text)]"
            style={SURFACE_STYLE}
        >
            <div className="border-b border-[color:var(--line)] px-6 py-5">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div className="min-w-0 space-y-3">
                        <Button variant="ghost" size="sm" onClick={() => navigate('/')} className="h-8 px-2 text-[var(--muted)]">
                            <ArrowLeft className="mr-2 h-4 w-4"/>
                            返回命令面板
                        </Button>
                        <div className="space-y-1">
                            <h1 className="text-2xl font-semibold tracking-tight">插件管理器</h1>
                            <p className="text-sm text-[var(--muted)]">
                                管理已安装插件、启用状态与插件元信息。
                            </p>
                        </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                        <Button variant="outline" onClick={() => void handleRefreshPlugins()} disabled={isLoading}>
                            <RefreshCcw className="mr-2 h-4 w-4"/>
                            刷新列表
                        </Button>
                        <Button onClick={() => void handleInstallPlugin()} disabled={isLoading}>
                            <Plus className="mr-2 h-4 w-4"/>
                            安装插件
                        </Button>
                    </div>
                </div>

                <div className="mt-4 flex flex-wrap gap-2">
                    <div className="inline-flex items-center gap-2 border border-[color:var(--line)] bg-[var(--surface)] px-3 py-2 text-sm">
                        <Package2 className="h-4 w-4 text-[var(--accent)]"/>
                        <span>共 {plugins.length} 个插件</span>
                    </div>
                    <div className="inline-flex items-center gap-2 border border-[color:var(--line)] bg-[var(--surface)] px-3 py-2 text-sm">
                        <Puzzle className="h-4 w-4 text-[var(--accent)]"/>
                        <span>已启用 {enabledCount} 个</span>
                    </div>
                    <div className="inline-flex items-center gap-2 border border-[color:var(--line)] bg-[var(--surface)] px-3 py-2 text-sm">
                        <span className="h-2 w-2 rounded-full bg-[var(--accent)]"/>
                        <span>UI 插件 {uiPluginCount} 个</span>
                    </div>
                </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col">
                <div className="border-b border-[color:var(--line)] px-6 py-4">
                    <div className="relative">
                        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--muted)]"/>
                        <Input
                            value={searchValue}
                            onChange={event => setSearchValue(event.target.value)}
                            placeholder="按名称、作者、描述或 Package ID 搜索"
                            className="h-10 border-[color:var(--line)] bg-[var(--surface)] pl-10 shadow-none"
                        />
                    </div>
                </div>

                <div className="min-h-0 flex-1 overflow-y-auto bg-[var(--surface)]">
                    {isLoading ? (
                        <div className="flex h-full min-h-[220px] items-center justify-center px-6 text-sm text-[var(--muted)]">
                            正在加载插件列表...
                        </div>
                    ) : filteredPlugins.length === 0 ? (
                        <div className="flex h-full min-h-[220px] flex-col items-center justify-center gap-3 px-6 text-center">
                            <div className="flex h-12 w-12 items-center justify-center border border-[color:var(--line)] bg-[var(--surface-soft)]">
                                <Puzzle className="h-5 w-5 text-[var(--muted)]"/>
                            </div>
                            <div className="space-y-1">
                                <h2 className="text-base font-semibold text-[var(--text)]">
                                    {plugins.length === 0 ? '还没有已安装插件' : '没有匹配的插件'}
                                </h2>
                                <p className="text-sm text-[var(--muted)]">
                                    {plugins.length === 0
                                        ? '从顶部操作区安装 .wt 插件包后，这里会显示插件列表。'
                                        : '调整搜索关键词后再试。'}
                                </p>
                            </div>
                        </div>
                    ) : (
                        <div className="divide-y divide-[color:var(--line)]">
                            {filteredPlugins.map(plugin => {
                                const isSelected = selectedPlugin?.packageId === plugin.packageId

                                return (
                                    <div
                                        key={plugin.packageId}
                                        className={cn(
                                            'flex items-center gap-4 px-6 py-4 transition-colors',
                                            isSelected && 'bg-[var(--accent-soft)]'
                                        )}
                                    >
                                        <button
                                            type="button"
                                            className="min-w-0 flex-1 text-left"
                                            onClick={() => handleOpenPluginDetails(plugin)}
                                        >
                                            <div className="flex flex-wrap items-center gap-2">
                                                <h2 className="truncate text-sm font-semibold text-[var(--text)]">
                                                    {plugin.name || '未命名插件'}
                                                </h2>
                                                <span className="text-xs text-[var(--muted)]">v{plugin.version || '未标注'}</span>
                                                {plugin.uiEnabled && (
                                                    <span className="inline-flex h-5 items-center rounded-sm border border-[color:var(--line)] bg-[var(--surface)] px-1.5 text-[10px] font-medium text-[var(--muted)]">
                                                        UI
                                                    </span>
                                                )}
                                            </div>
                                            <p className="mt-1 line-clamp-2 text-sm text-[var(--muted)]">
                                                {plugin.description || '这个插件还没有提供描述。'}
                                            </p>
                                            <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[var(--muted)]">
                                                <span>{plugin.author || '未知作者'}</span>
                                                <span>{plugin.usedCount} 次使用</span>
                                                <span>{formatLastUsedAt(plugin.lastUsedAt)}</span>
                                            </div>
                                        </button>

                                        <div className="flex shrink-0 items-center gap-3">
                                            <span className="hidden text-xs text-[var(--muted)] sm:inline">
                                                {plugin.enabled ? '已启用' : '已停用'}
                                            </span>
                                            <Switch
                                                checked={plugin.enabled}
                                                onCheckedChange={() => void handleTogglePlugin(plugin)}
                                                aria-label={`${plugin.name} 开关`}
                                            />
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    )}
                </div>
            </div>

            <div className="border-t border-[color:var(--line)] bg-[var(--surface)] px-6 py-3 text-sm text-[var(--muted)]">
                <div className="flex flex-col gap-1 lg:flex-row lg:items-center lg:justify-between">
                    <span>
                        当前显示 {filteredPlugins.length} / {plugins.length} 个插件
                    </span>
                    <span>
                        {error ? `加载异常: ${error}` : selectedPlugin ? `已选中: ${selectedPlugin.packageId}` : '点击插件查看详情，按 Esc 返回命令面板'}
                    </span>
                </div>
            </div>

            <Sheet open={isDrawerOpen} onOpenChange={setIsDrawerOpen} className="w-full max-w-[720px] border-l border-[color:var(--line)]">
                {selectedPlugin && (
                    <>
                        <SheetHeader>
                            <SheetTitle>{selectedPlugin.name || '插件详情'}</SheetTitle>
                            <SheetDescription>
                                查看插件元信息、状态与管理操作。
                            </SheetDescription>
                        </SheetHeader>
                        <SheetContent className="p-0">
                            <PluginDetails
                                plugin={selectedPlugin}
                                onToggle={handleTogglePlugin}
                                onUninstall={handleUninstallPlugin}
                                onOpenHomepage={handleOpenHomepage}
                                onClose={() => setIsDrawerOpen(false)}
                            />
                        </SheetContent>
                        <SheetFooter className="hidden"/>
                    </>
                )}
            </Sheet>
        </div>
    )
}
