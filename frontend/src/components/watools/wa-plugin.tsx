import {useCallback, useEffect, useRef, useState} from "react";
import {useLocation, useSearchParams} from "wouter";
import {useAppStore, usePluginStore} from "@/stores";
import {createWaToolsApi} from "@/api/api";
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

    const handleHotkey = useCallback((e: KeyboardEvent) => {
        if (e.isComposing || e.keyCode === 229) return;
        if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            navigate('/');
        }
    }, [navigate]);

    useEffect(() => {
        window.addEventListener('keydown', handleHotkey);
        return () => {
            window.removeEventListener('keydown', handleHotkey);
            listenerWindow.current?.removeEventListener('keydown', handleHotkey);
            releasePluginLaunchContext(launchId);
        };
    }, [launchId, handleHotkey]);

    const handleIframeLoad = () => {
        const frame = iframeRef.current?.contentWindow;
        if (!frame || !launch.context || deliveredDocument.current === frame.document) return;
        listenerWindow.current?.removeEventListener('keydown', handleHotkey);
        frame.addEventListener('keydown', handleHotkey);
        listenerWindow.current = frame;
        deliveredDocument.current = frame.document;
        type Runtime = typeof import('@/lib/legacy-runtime');
        const target = frame as Window & {runtime: Runtime; watools: ReturnType<typeof createWaToolsApi>; inputValue: string; pluginContext: PluginContext};
        target.runtime = (window as unknown as {runtime: Runtime}).runtime;
        target.watools = createWaToolsApi(packageId);
        // Give every iframe document its own mutable wrappers; strings remain
        // shared, and a plugin cannot mutate the session used on a later reload.
        target.pluginContext = snapshotPluginContext(launch.context);
        target.inputValue = target.pluginContext.input.value;
        frame.dispatchEvent(new CustomEvent('watools:context-ready', {detail: target.pluginContext}));
        releasePluginLaunchContext(launchId);
        useAppStore.getState().clearValue(launch.revision);
    };

    return <div className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
        {pluginUrl && <iframe ref={iframeRef} className="block h-full min-h-0 w-full flex-1 border-0" src={pluginUrl} onLoad={handleIframeLoad}/>}
        {!pluginUrl && <div className="p-6">{launch.error || plugin?.loadError || '插件不可用，请检查启用状态或重新安装。'}<button className="ml-4 underline" onClick={() => navigate('/')}>返回启动器</button></div>}
    </div>;
};
