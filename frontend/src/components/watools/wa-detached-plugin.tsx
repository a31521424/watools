import {useEffect, useRef, useState} from 'react';
import {host} from '@/api/host';
import type {PluginWindowSession} from '@/api/host/contracts';
import {attachPluginSession} from '@/lib/plugin-ui-session';
import {restorePluginWindowState} from '@/lib/plugin-window-state';

export default function DetachedPlugin({id}: {id: string}) {
    const [session,setSession]=useState<PluginWindowSession | null>(null);
    const [error,setError]=useState('');
    const frame=useRef<HTMLIFrameElement>(null);
    const documentReady=useRef<Document | null>(null);
    const disposeSession=useRef<(() => void) | null>(null);
    const alive=useRef(true);
    useEffect(()=>{
        alive.current=true;
        let cancelled=false;
        host.GetPluginWindowApi(id).then(value=>{if (!cancelled) setSession(value)}).catch(()=>{if (!cancelled) setError('插件窗口会话不可用')});
        return ()=>{cancelled=true; alive.current=false; disposeSession.current?.()};
    },[id]);
    const load=async()=>{
        const target=frame.current?.contentWindow;
        if (!target || !session || documentReady.current===target.document) return;
        documentReady.current=target.document;
        disposeSession.current?.();
        disposeSession.current=attachPluginSession(target,session.packageId,session.windowName,id,session.context);
        const key=(event:KeyboardEvent)=>{
            if (event.isComposing || event.keyCode===229) return;
            if (event.key==='Escape') {event.preventDefault(); event.stopImmediatePropagation(); void host.ClosePluginWindowApi(id)}
        };
        target.addEventListener('keydown',key,true);
        const dispose=disposeSession.current;
        disposeSession.current=()=>{target.removeEventListener('keydown',key,true); dispose()};
        try {
            await restorePluginWindowState(target,session.state);
            if (alive.current) await host.PluginWindowReadyApi(id);
        } catch {
            setError('插件窗口状态恢复失败，请返回主窗口重试');
            await host.ClosePluginWindowApi(id).catch(()=>{});
        }
    };
    return <div className="flex h-screen min-h-0 flex-col bg-white">
        {error && <div role="alert" className="p-4 text-red-600">{error}</div>}
        {session && <iframe ref={frame} title={session.title} src={`${session.homeUrl}/${session.file}?instance=${id}`} onLoad={()=>void load()} className="block min-h-0 w-full flex-1 border-0"/>}
    </div>;
}
