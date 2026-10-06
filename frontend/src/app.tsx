import './app.css'
import {lazy, Suspense} from 'react';
const Watools=lazy(()=>import('@/components/watools/watools'));
const DetachedPlugin=lazy(()=>import('@/components/watools/wa-detached-plugin'));

export default () => {
    const id=new URLSearchParams(window.location.search).get('pluginSession');
    return <Suspense fallback={<div className="p-4">正在加载…</div>}>{id ? <DetachedPlugin id={id}/> : <Watools/>}</Suspense>;
};
