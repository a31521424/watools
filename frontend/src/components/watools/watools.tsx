import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {Route, Switch} from "wouter";
import {useEffect} from "react";
import {usePluginActions} from "@/store/pluginStore";
import {GetPluginExecEntryApi, GetPluginsApi, HideAppApi} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {PluginPackage} from "@/schemas/plugin";
import {WaPluginRender} from "@/components/watools/wa-plugin-render";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {isDevMode} from "@/lib/env";

const Watools = () => {
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })
    const {setPlugins} = usePluginActions()

    useEffect(() => {
        (async () => {
            const allPlugins = await GetPluginsApi()
            const loadedPlugins = await Promise.all(
                allPlugins.map(async plugin => {
                    let execEntry = await GetPluginExecEntryApi(plugin.id)
                    execEntry = `/api/plugin-entry?path=${encodeURIComponent(execEntry)}&timestamp=${Date.now()}`
                    const {default: pluginPackage}: {
                        default: PluginPackage
                    } = await import(/* @vite-ignore */ execEntry)
                    pluginPackage.metadata = plugin
                    return pluginPackage
                })
            )
            setPlugins(loadedPlugins)
        })()

    }, [])

    useWindowFocus((focused) => {
        if (!focused) {
            if (isDevMode()) {
                return
            }
            HideAppApi()
        }
    })

    // TODO: define page paths enum
    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide">
        <Switch>
            <Route path="/"> <WaCommand/> </Route>
            <Route path="/plugins/:entryID"> <WaPluginRender/> </Route>
        </Switch>
    </div>
}


export default Watools