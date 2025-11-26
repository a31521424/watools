import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {Route} from "wouter";
import {WaPlugin} from "@/components/watools/wa-plugin";
import {PluginManagement} from "@/components/watools/plugin-management";
import {useEffect} from "react";
import {WaApi} from "@/api/api";

const Watools = () => {
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })
    useEffect(() => {
        // @ts-ignore
        window.watools = WaApi
        return () => {
            // @ts-ignore
            delete window.watools
        }
    }, []);


    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide border-0">
        <Route path='/'>
            <WaCommand/>
        </Route>
        <Route path='/plugin'>
            <WaPlugin/>
        </Route>
        <Route path='/plugin-management'>
            <PluginManagement/>
        </Route>
    </div>
}


export default Watools