import {useEffect, useRef, useState} from 'react'
import {FolderCode, Layers, RefreshCcw, Settings2, Terminal} from 'lucide-react'
import {WaWorkspaceHeader} from './wa-workspace-header'
import {host} from '@/api/host'
import type {LocalInstallationConfiguration, LocalInstallationRequest, LocalInstallationStatus} from '@/api/host/contracts'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {WaPluginManagement} from './wa-plugin-management'
import {usePluginStore} from '@/stores/pluginStore'

export function WaManagement({initialTab = 'application'}: {initialTab?: 'application' | 'plugins'}) {
    const [tab, setTab] = useState(initialTab)
    const [status, setStatus] = useState<LocalInstallationStatus>()
    const [sourceRoot, setSourceRoot] = useState('')
    const [goExecutable, setGoExecutable] = useState('')
    const [toolPath, setToolPath] = useState('')
    const [error, setError] = useState('')
    const [refreshError, setRefreshError] = useState('')
    const [pending, setPending] = useState(false)
    const mounted = useRef(false)
    const initialized = useRef(false)
    const refreshedJob = useRef(0)
    const refreshPlugins = usePluginStore(state => state.refreshPlugins)
    const busy = pending || status?.state === 'running'

    useEffect(() => {
        mounted.current = true
        let timer: ReturnType<typeof setTimeout>
        let stopped = false
        const poll = async () => {
            try {
                const next = await host.GetLocalInstallationApi()
                if (stopped) return
                setStatus(next)
                if (!initialized.current) {
                    setSourceRoot(next.configuration.sourceRoot)
                    setGoExecutable(next.configuration.goExecutable)
                    setToolPath(next.configuration.toolPath)
                    initialized.current = true
                }
                if (next.state === 'succeeded' && next.jobId !== refreshedJob.current) {
                    try {
                        await refreshPlugins()
                        refreshedJob.current = next.jobId
                        if (!stopped) setRefreshError('')
                    } catch (failure) {
                        if (!stopped) setRefreshError(failure instanceof Error ? failure.message : String(failure))
                    }
                }
            } catch (failure) {
                if (!stopped) setError(failure instanceof Error ? failure.message : String(failure))
            } finally {
                if (!stopped) timer = setTimeout(poll, 1000)
            }
        }
        void poll()
        return () => {stopped = true; mounted.current = false; clearTimeout(timer)}
    }, [refreshPlugins])

    const perform = async (action: () => Promise<void>) => {
        if (pending) return
        setPending(true); setError('')
        try {
            await action()
            const next = await host.GetLocalInstallationApi()
            if (mounted.current) setStatus(next)
        } catch (failure) {
            if (mounted.current) setError(failure instanceof Error ? failure.message : String(failure))
        } finally {if (mounted.current) setPending(false)}
    }
    const start = (scope: LocalInstallationRequest['scope'], packageIds: string[] = []) => {
        if (busy) return
        void perform(() => host.StartLocalInstallationApi({scope, packageIds}))
    }
    const save = () => {
        const configuration: LocalInstallationConfiguration = {...(status?.configuration ?? {applicationPath:'',installedAt:'',version:''}), sourceRoot:sourceRoot.trim(),goExecutable:goExecutable.trim(),toolPath:toolPath.trim()}
        void perform(() => host.ConfigureLocalInstallationApi(configuration))
    }

    return <div className="wa-page flex h-full min-h-0 flex-col">
        <WaWorkspaceHeader title="管理中心" subtitle="应用更新、官方工具与插件管理" icon={<Settings2/>}/>
        <div className="shrink-0 px-6 py-4"><nav className="wa-tabs" aria-label="管理分类">
            <button aria-pressed={tab==='application'} onClick={()=>setTab('application')}>应用与更新</button>
            <button aria-pressed={tab==='plugins'} onClick={()=>setTab('plugins')}>插件</button>
        </nav></div>
        {(error || refreshError || status?.message || status?.restartRequired) && <div className="wa-notice shrink-0" aria-live="polite">
            {error && <p role="alert" className="text-destructive">{error}</p>}
            {refreshError && <p role="alert" className="text-destructive">{refreshError}</p>}
            {status?.message && <p className={status.state==='failed' ? 'text-destructive' : ''}>{status.message}</p>}
            {status?.restartRequired && <p className="mt-1 text-amber-800">更新已完成。保存小窗内容后，退出并重新打开 WaTools。</p>}
            {status?.canCancel && <Button className="mt-2" size="sm" variant="outline" disabled={pending} onClick={()=>void perform(()=>host.CancelLocalInstallationApi())}>取消构建</Button>}
            {!!status?.output.length && <details className="mt-2"><summary className="cursor-pointer text-muted-foreground">查看任务输出</summary><pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap break-all rounded-lg border bg-card p-3 text-xs">{status.output.join('\n')}</pre></details>}
        </div>}
        {tab==='application' ? <main className="min-h-0 flex-1 overflow-y-auto px-6 pb-6">
            <div className="grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)]">
                <section className="wa-card space-y-5">
                    <div className="flex items-start gap-3"><FolderCode className="mt-0.5 size-5 text-primary"/><div><h2 className="wa-section-heading">本地源码</h2><p className="wa-section-caption">连接你的 WaTools 源码，用同一套流程完成构建与更新。</p></div></div>
                    <label className="block text-xs font-medium">源码目录<Input className="mt-2" value={sourceRoot} disabled={busy} onChange={event=>setSourceRoot(event.target.value)} placeholder="/Users/…/watools"/></label>
                    <details className="rounded-lg border bg-background p-3"><summary className="cursor-pointer text-xs text-muted-foreground">编译环境</summary>
                        <label className="mt-4 block text-xs">Go 可执行文件<Input className="mt-2" value={goExecutable} disabled={busy} onChange={event=>setGoExecutable(event.target.value)} placeholder="留空时自动查找"/></label>
                        <label className="mt-4 block text-xs">工具搜索路径 PATH<Input className="mt-2" value={toolPath} disabled={busy} onChange={event=>setToolPath(event.target.value)} placeholder="安装命令会自动记录终端路径"/></label>
                    </details>
                    <div className="flex items-center justify-between gap-3"><p className="text-[11px] text-muted-foreground">只使用你信任的源码目录。</p><Button variant="outline" size="sm" disabled={busy || !sourceRoot.trim()} onClick={save}>保存设置</Button></div>
                </section>
                <div className="space-y-5">
                    <section className="wa-card space-y-4">
                        <div className="flex items-start gap-3"><RefreshCcw className="mt-0.5 size-5 text-primary"/><div><h2 className="wa-section-heading">安装与更新</h2><p className="wa-section-caption">更新应用和工具，保留已有配置与插件数据。</p></div></div>
                        <div className="rounded-lg bg-background px-3 py-3 text-xs"><p className="text-muted-foreground">应用位置</p><p className="mt-1.5 break-all">{status?.configuration.applicationPath || '/Applications/watools.app'}</p></div>
                        {status?.configuration.installedAt && <p className="text-[11px] text-muted-foreground">上次安装 {new Date(status.configuration.installedAt).toLocaleString()}{status.configuration.version && ` · v${status.configuration.version}`}</p>}
                        <div className="flex flex-wrap gap-2"><Button disabled={busy || !status?.configuration.sourceRoot} onClick={()=>start('all')}>更新应用和官方插件</Button><Button variant="outline" disabled={busy || !status?.configuration.sourceRoot} onClick={()=>start('application')}>只更新应用</Button></div>
                    </section>
                    <section className="wa-card"><div className="flex items-start gap-3"><Terminal className="mt-0.5 size-5 text-muted-foreground"/><div><h2 className="wa-section-heading">先获取最新源码</h2><p className="wa-section-caption">在源码目录执行 <code className="rounded bg-secondary px-1.5 py-0.5 text-[11px]">git pull --ff-only</code>，再开始更新。</p></div></div></section>
                </div>
            </div>
        </main> : <div className="flex min-h-0 flex-1 flex-col">
            <section className="shrink-0 px-6 pb-4">
                <details className="rounded-xl border bg-card p-4"><summary className="cursor-pointer text-xs font-medium"><span className="inline-flex items-center gap-2"><Layers size={14} className="text-primary"/>官方工具<span className="text-muted-foreground">{status?.officialPlugins.length || 0} 个</span></span></summary>
                    <div className="mt-3 flex justify-end"><Button size="sm" variant="outline" disabled={busy || !status?.officialPlugins.length} onClick={()=>start('plugins')}>安装／更新全部</Button></div>
                    <div className="mt-3 grid max-h-36 gap-2 overflow-y-auto sm:grid-cols-2 lg:grid-cols-3">{status?.officialPlugins.map(plugin=><div key={plugin.packageId} className="flex items-center justify-between gap-2 rounded-lg bg-background p-2.5 text-xs"><span>{plugin.name}<span className="ml-2 text-muted-foreground">v{plugin.version}</span></span><Button size="sm" variant="ghost" disabled={busy} onClick={()=>start('plugins',[plugin.packageId])}>更新</Button></div>)}</div>
                    {!status?.officialPlugins.length && <p className="mt-3 text-xs text-muted-foreground">设置源码目录后，即可安装官方工具。</p>}
                </details>
            </section>
            <WaPluginManagement embedded busy={busy}/>
        </div>}
    </div>
}
