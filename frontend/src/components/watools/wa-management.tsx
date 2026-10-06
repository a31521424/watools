import {useEffect, useRef, useState} from 'react'
import {useLocation} from 'wouter'
import {ArrowLeft, RefreshCcw} from 'lucide-react'
import {host} from '@/api/host'
import type {LocalInstallationConfiguration, LocalInstallationRequest, LocalInstallationStatus} from '@/api/host/contracts'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {WaPluginManagement} from './wa-plugin-management'
import {usePluginStore} from '@/stores/pluginStore'

export function WaManagement({initialTab = 'application'}: {initialTab?: 'application' | 'plugins'}) {
    const [, navigate] = useLocation()
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

    return <div className="flex h-full min-h-0 flex-col bg-slate-50 text-slate-900">
        <header className="shrink-0 border-b bg-white px-6 py-4">
            <Button variant="ghost" size="sm" onClick={() => navigate('/')}><ArrowLeft className="mr-2 size-4"/>返回命令面板</Button>
            <h1 className="mt-2 text-xl font-semibold">管理中心</h1>
            <p className="mt-1 text-sm text-slate-500">从本地源码安装和更新 WaTools，管理插件。</p>
            <div className="mt-3 flex gap-2">
                <Button variant={tab === 'application' ? 'default' : 'outline'} onClick={() => setTab('application')}>应用与更新</Button>
                <Button variant={tab === 'plugins' ? 'default' : 'outline'} onClick={() => setTab('plugins')}>插件</Button>
            </div>
        </header>
        {(error || refreshError || status?.message || status?.restartRequired) && <div className="shrink-0 border-b px-6 py-3 text-sm" aria-live="polite">
            {error && <p role="alert" className="text-red-700">{error}</p>}
            {refreshError && <p role="alert" className="text-red-700">{refreshError}</p>}
            {status?.message && <p className={status.state === 'failed' ? 'text-red-700' : ''}>{status.message}</p>}
            {status?.restartRequired && <p className="mt-1 text-amber-800">应用已更新。请保存小窗内容后退出并重新打开 WaTools。</p>}
            {status?.canCancel && <Button className="mt-2" size="sm" variant="outline" disabled={pending} onClick={() => void perform(() => host.CancelLocalInstallationApi())}>取消构建</Button>}
            {!!status?.output.length && <details className="mt-2"><summary className="cursor-pointer text-slate-500">任务输出</summary><pre className="mt-2 max-h-36 overflow-auto whitespace-pre-wrap break-all rounded border bg-white p-2 text-xs">{status.output.join('\n')}</pre></details>}
        </div>}
        {tab === 'application' ? <main className="min-h-0 flex-1 space-y-5 overflow-y-auto p-6">
            <section className="space-y-3 rounded-lg border bg-white p-4">
                <h2 className="font-medium">本地源码</h2>
                <label className="block text-sm">WaTools 源码目录<Input className="mt-1" value={sourceRoot} disabled={busy} onChange={event => setSourceRoot(event.target.value)} placeholder="/Users/…/watools"/></label>
                <details><summary className="cursor-pointer text-sm text-slate-500">编译环境</summary>
                    <label className="mt-3 block text-sm">Go 可执行文件<Input className="mt-1" value={goExecutable} disabled={busy} onChange={event => setGoExecutable(event.target.value)} placeholder="留空时自动查找"/></label>
                    <label className="mt-3 block text-sm">工具搜索路径 PATH<Input className="mt-1" value={toolPath} disabled={busy} onChange={event => setToolPath(event.target.value)} placeholder="安装命令会自动记录终端的工具路径"/></label>
                </details>
                <Button variant="outline" disabled={busy || !sourceRoot.trim()} onClick={save}>保存源码设置</Button>
                <p className="text-xs text-slate-500">只使用你信任的源码目录。安装命令会自动记录此设置。</p>
            </section>
            <section className="space-y-3 rounded-lg border bg-white p-4">
                <h2 className="font-medium">安装与覆盖更新</h2>
                <p className="break-all text-sm text-slate-500">应用位置：{status?.configuration.applicationPath || '/Applications/watools.app'}</p>
                {status?.configuration.installedAt && <p className="text-sm text-slate-500">上次安装：{new Date(status.configuration.installedAt).toLocaleString()} · {status.configuration.version}</p>}
                <div className="flex flex-wrap gap-2">
                    <Button disabled={busy || !status?.configuration.sourceRoot} onClick={() => start('all')}><RefreshCcw className="mr-2 size-4"/>更新应用和官方插件</Button>
                    <Button variant="outline" disabled={busy || !status?.configuration.sourceRoot} onClick={() => start('application')}>只更新应用</Button>
                </div>
                <p className="text-xs text-slate-500">先在源码目录运行 git pull --ff-only 获取新代码。更新保留配置、插件数据与第三方插件。</p>
            </section>
        </main> : <div className="flex min-h-0 flex-1 flex-col">
            <section className="shrink-0 border-b bg-white px-6 py-3">
                <div className="flex items-center justify-between gap-2"><h2 className="text-sm font-medium">源码中的官方插件</h2><Button size="sm" variant="outline" disabled={busy || !status?.officialPlugins.length} onClick={() => start('plugins')}>安装／更新全部</Button></div>
                <div className="mt-2 max-h-32 overflow-y-auto">
                    {status?.officialPlugins.map(plugin => <div key={plugin.packageId} className="flex items-center justify-between gap-2 py-1 text-sm"><span>{plugin.name} <span className="text-xs text-slate-500">{plugin.version}</span></span><Button size="sm" variant="ghost" disabled={busy} onClick={() => start('plugins', [plugin.packageId])}>安装／更新</Button></div>)}
                    {!status?.officialPlugins.length && <p className="text-sm text-slate-500">在“应用与更新”设置源码目录后，可安装官方插件。</p>}
                </div>
            </section>
            <WaPluginManagement embedded busy={busy}/>
        </div>}
    </div>
}
