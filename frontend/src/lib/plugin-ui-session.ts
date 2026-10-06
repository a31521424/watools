import {createWaToolsApi, type WaToolsUIApi} from '@/api/api';
import {createPluginRuntime} from '@/lib/legacy-runtime';
import {snapshotPluginContext} from '@/lib/plugin-context';
import type {PluginContext} from '@/schemas/plugin';

export const attachPluginSession = (frame: Window, packageId: string, windowName: string, instanceId: string, context: PluginContext) => {
    const api = createWaToolsApi(packageId);
    const runtime = createPluginRuntime(windowName, instanceId);
    const requests = new Map<string,string>();
    let disposed = false;
    const scopedApi: WaToolsUIApi = {...api,
        session: Object.freeze({id: instanceId, windowMode: windowName === 'launcher' ? 'inline' : 'detached'}),
        HttpProxy: async (request: Parameters<typeof api.HttpProxy>[0]) => {
            if (disposed) throw new Error('插件会话已关闭');
            const key = request.requestId || crypto.randomUUID();
            if (requests.has(key)) throw new Error('请求标识已在使用');
            const id = `${instanceId}:${key}`;
            requests.set(key,id);
            try { return await api.HttpProxy({...request,requestId:id}); }
            finally { requests.delete(key); }
        },
        CancelHttpProxy: async (id: string) => {const owned = requests.get(id); if (owned) await api.CancelHttpProxy(owned)},
    };
    const target = frame as Window & {runtime: typeof runtime; watools: typeof scopedApi; pluginContext: PluginContext; inputValue: string};
    target.runtime = runtime;
    target.watools = scopedApi;
    target.pluginContext = snapshotPluginContext(context);
    target.inputValue = target.pluginContext.input.value;
    const dispose = () => {
        if (disposed) return;
        disposed = true;
        runtime.dispose();
        for (const id of requests.values()) void api.CancelHttpProxy(id).catch(() => {});
        requests.clear();
        frame.removeEventListener('pagehide',dispose);
    };
    frame.addEventListener('pagehide',dispose);
    frame.dispatchEvent(new CustomEvent('watools:context-ready',{detail:target.pluginContext}));
    return dispose;
};
