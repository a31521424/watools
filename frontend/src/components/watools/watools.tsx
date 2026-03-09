import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {Route} from "wouter";
import {WaPlugin} from "@/components/watools/wa-plugin";
import {WaPluginManagement} from "@/components/watools/wa-plugin-management";
import {useEffect} from "react";
import {WaApi} from "@/api/api";
import {usePluginStore} from "@/stores/pluginStore";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";

const Watools = () => {
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


    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide border-0">
        <Route path='/'>
            <WaCommand/>
        </Route>
        <Route path='/plugin'>
            <WaPlugin/>
        </Route>
        <Route path='/plugin-management'>
            <WaPluginManagement/>
        </Route>
    </div>
}


export default Watools
