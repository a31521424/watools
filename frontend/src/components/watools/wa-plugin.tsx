import {useCallback, useEffect, useRef, useState} from "react";
import {useLocation, useSearchParams} from "wouter";
import {useAppStore, usePluginStore} from "@/stores";
import {host} from '@/api/host';
import {attachPluginSession} from '@/lib/plugin-ui-session';
import {capturePluginWindowState} from '@/lib/plugin-window-state';
import {normalizePluginAssetPath} from "@/lib/plugin";
import {resolvePluginLaunchContext, releasePluginLaunchContext, snapshotPluginContext} from "@/lib/plugin-context";
import type {PluginContext} from '@/schemas/plugin';

type LaunchParams = {packageId: string; file: string | null; launchId: string | null; seed: string};

export const WaPlugin = () => {
    const [params] = useSearchParams();
    const launch: LaunchParams = {
        packageId: params.get('packageId') || '', file: params.get('file'),
        launchId: params.get('launchId'), seed: params.get('seed') || '',
    };
    return <PluginSession key={JSON.stringify(launch)} {...launch}/>;
};

// A route is a single launch session. Input clearing, usage updates and iframe
// reloads cannot resolve its payload again from mutable launcher state.
const PluginSession = ({packageId, file, launchId, seed}: LaunchParams) => {
    const iframeRef = useRef<HTMLIFrameElement | null>(null);
    const listenerWindow = useRef<Window | null>(null);
    const deliveredDocument = useRef<Document | null>(null);
    const disposeSession = useRef<(() => void) | null>(null);
    const alive = useRef(true);
    const pendingDetach = useRef<string | null>(null);
    const detachBusy = useRef(false);
    const [detaching, setDetaching] = useState(false);
    const [frameReady, setFrameReady] = useState(false);
    const [detachError, setDetachError] = useState('');
    const plugin = usePluginStore(state => state.getPluginById(packageId));
    const [, navigate] = useLocation();
    const [nonce] = useState(() => Date.now().toString());
    const [launch] = useState(() => {
        const state = useAppStore.getState();
        try {
            const liveContext = state.input.kind === 'inline' ? state.input.context : null;
            return {context: snapshotPluginContext(resolvePluginLaunchContext({launchId, seed, liveContext})), revision: state.inputRevision, error: ''};
        } catch (error) {
            return {context: null, revision: state.inputRevision, error: String(error)};
        }
    });
    const safeFile = normalizePluginAssetPath(file);
    const available = plugin?.enabled && safeFile && launch.context && plugin.entry.some(entry => entry.type === 'ui' && entry.file === safeFile);
    const pluginUrl = available ? plugin.homeUrl + '/' + safeFile + '?t=' + nonce : null;
    useEffect(() => {
        if (!available) releasePluginLaunchContext(launchId);
    }, [available, launchId]);

    const detach = useCallback(async () => {
        const frame = iframeRef.current?.contentWindow;
        if (!frame || deliveredDocument.current!==frame.document || !launch.context || !plugin || !safeFile || detachBusy.current) return;
        detachBusy.current = true;
        setDetaching(true); setDetachError('');
        frame.document.body.inert = true;
        try {
            const state = await capturePluginWindowState(frame);
            if (!alive.current) return;
            const id = await host.DetachPluginApi({packageId, file:safeFile, homeUrl:plugin.homeUrl, context:launch.context, state});
            if (!alive.current) {await host.ClosePluginWindowApi(id); return}
            pendingDetach.current = id;
            const deadline = Date.now()+30_000;
            while (alive.current && Date.now()<deadline) {
                if ((await host.GetPluginWindowStatusApi(id)).ready) {
                    if (!alive.current) return;
                    pendingDetach.current = null;
                    await host.HideAppApi();
                    if (alive.current) navigate('/');
                    return;
                }
                await new Promise(resolve => setTimeout(resolve,100));
            }
            throw new Error('插件窗口未完成加载，请重试');
        } catch (error) {
            if (alive.current) setDetachError(`分离失败：${error instanceof Error ? error.message : String(error)}`);
        } finally {
            if (pendingDetach.current) {void host.ClosePluginWindowApi(pendingDetach.current).catch(()=>{}); pendingDetach.current=null}
            frame.document.body.inert = false;
            detachBusy.current=false;
            if (alive.current) setDetaching(false);
        }
    }, [launch.context,plugin,safeFile,packageId,navigate]);

    const handleHotkey = useCallback((e: KeyboardEvent) => {
        if (e.isComposing || e.keyCode === 229) return;
        if (e.metaKey && !e.shiftKey && !e.altKey && e.key.toLowerCase()==='d') {
            e.preventDefault(); e.stopImmediatePropagation();
            if (!e.repeat) void detach();
            return;
        }
        if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            navigate('/');
        }
    }, [navigate,detach]);

    useEffect(() => {
        window.addEventListener('keydown', handleHotkey,true);
        return () => {
            window.removeEventListener('keydown', handleHotkey,true);
            listenerWindow.current?.removeEventListener('keydown', handleHotkey,true);
        };
    }, [handleHotkey]);
    useEffect(() => {
        alive.current=true;
        return () => {
            alive.current=false;
            disposeSession.current?.();
            if (pendingDetach.current) void host.ClosePluginWindowApi(pendingDetach.current).catch(()=>{});
            releasePluginLaunchContext(launchId);
        };
    }, [launchId]);
    useEffect(() => {
        const frame=listenerWindow.current;
        frame?.addEventListener('keydown',handleHotkey,true);
        return () => frame?.removeEventListener('keydown',handleHotkey,true);
    },[handleHotkey]);

    const handleIframeLoad = () => {
        const frame = iframeRef.current?.contentWindow;
        if (!frame || !launch.context || deliveredDocument.current === frame.document) return;
        listenerWindow.current?.removeEventListener('keydown', handleHotkey,true);
        frame.addEventListener('keydown', handleHotkey,true);
        listenerWindow.current = frame;
        deliveredDocument.current = frame.document;
        disposeSession.current?.();
        disposeSession.current=attachPluginSession(frame,packageId,'launcher',launchId || nonce,launch.context);
        setFrameReady(true);
        releasePluginLaunchContext(launchId);
        useAppStore.getState().clearValue(launch.revision);
    };

    return <div className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
        <div className="flex shrink-0 items-center gap-3 border-b px-3 py-1 text-sm text-gray-600">
            <button onClick={()=>navigate('/')}>返回</button>
            <span className="flex-1 truncate">{plugin?.name}</span>
            <button disabled={detaching || !frameReady || !pluginUrl} onClick={()=>void detach()}>{detaching ? '正在分离…' : '分离 ⌘D'}</button>
        </div>
        {detachError && <div role="alert" className="px-3 py-1 text-sm text-red-600">{detachError}</div>}
        {pluginUrl && <iframe ref={iframeRef} className="block h-full min-h-0 w-full flex-1 border-0" src={pluginUrl} onLoad={handleIframeLoad}/>}
        {!pluginUrl && <div className="p-6">{launch.error || plugin?.loadError || '插件不可用，请检查启用状态或重新安装。'}<button className="ml-4 underline" onClick={() => navigate('/')}>返回启动器</button></div>}
    </div>;
};
