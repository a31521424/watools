import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {Route} from "wouter";
import {WaPlugin} from "@/components/watools/wa-plugin";
import {WaManagement} from "@/components/watools/wa-management";
import {WaLogManagement} from "@/components/watools/wa-log-management";
import {useEffect} from "react";
import {WaApi} from "@/api/api";
import {usePluginStore} from "@/stores/pluginStore";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";
import {useLocation} from "wouter";
import {cn} from "@/lib/utils";
import {useAppStore} from '@/stores/appStore';
import {watchClipboardSnapshot} from '@/api/app';
import {Logger} from '@/lib/logger';

const FIXED_PANEL_HEIGHT_CLASS = "h-[760px]"

const Watools = () => {
    useEffect(() => watchClipboardSnapshot(useAppStore, () => Logger.error('剪贴板快照维护失败')), []);
    const [location] = useLocation()
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })
    const flushPluginUsage = usePluginStore(state => state.flushBufferUpdates)
    const flushApplicationUsage = useApplicationCommandStore(state => state.flushBufferUpdates)

    useEffect(() => {
        // @ts-ignore
        window.watools = WaApi // TODO: not set plugin package id
        return () => {
            // @ts-ignore
            delete window.watools
        }
    }, []);

    useEffect(() => {
        const flushUsageBuffers = () => {
            void flushApplicationUsage()
            void flushPluginUsage()
        }

        window.addEventListener("beforeunload", flushUsageBuffers)
        window.addEventListener("pagehide", flushUsageBuffers)

        return () => {
            window.removeEventListener("beforeunload", flushUsageBuffers)
            window.removeEventListener("pagehide", flushUsageBuffers)
            flushUsageBuffers()
        }
    }, [flushApplicationUsage, flushPluginUsage]);

    const isFixedHeightRoute = location === '/plugin'
        || location === '/management'
        || location === '/plugin-management'
        || location === '/log-management'
        || location === '/update-management'

    return <div
        ref={windowRef}
        className={cn(
            "wa-shell relative flex min-h-0 w-full flex-col overflow-hidden",
            isFixedHeightRoute && FIXED_PANEL_HEIGHT_CLASS
        )}
    >
        <Route path='/'>
            <WaCommand/>
        </Route>
        <Route path='/plugin'>
            <WaPlugin/>
        </Route>
        <Route path='/management'><WaManagement/></Route>
        <Route path='/plugin-management'>
            <WaManagement initialTab="plugins"/>
        </Route>
        <Route path='/log-management'>
            <WaLogManagement/>
        </Route>
        <Route path='/update-management'>
            <WaManagement/>
        </Route>
    </div>
}


export default Watools
