import Watools from "@/components/watools/watools";
import './app.css'
import {useEffect} from 'react';
import {useAppStore} from '@/stores/appStore';
import {watchClipboardSnapshot} from '@/api/app';
import {Logger} from '@/lib/logger';


export default () => {
    useEffect(() => watchClipboardSnapshot(useAppStore, () => Logger.error('剪贴板快照维护失败')), []);
    return <Watools/>
}
