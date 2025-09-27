import {WaCommand} from "./wa-command";
import {resizeWindowHeight, useElementResize} from "@/hooks/useElementResize";
import {Route, Switch} from "wouter";
import {useEffect} from "react";
import {usePluginActions} from "@/store/pluginStore";
import {GetPluginExecEntryApi, GetPluginsApi, HideAppApi} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {WaPluginRender} from "@/components/watools/wa-plugin-render";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {isDevMode} from "@/lib/env";
import {Logger} from "@/lib/logger";

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
                    console.log('on load plugin', plugin)
                    try {
                        let execEntry = await GetPluginExecEntryApi(plugin.id)
                        execEntry = `/api/plugin-entry?path=${encodeURIComponent(execEntry)}&timestamp=${Date.now()}`
                        console.log('load exec entry', execEntry)
                        const module = await import(/* @vite-ignore */ execEntry)
                        console.log('Loaded plugin:', plugin.packageID, module.default)
                        return {
                            ...module.default,
                            metadata: plugin,
                        }
                    } catch (e) {
                        Logger.error(`Failed to load plugin ${plugin.packageID}: ${e}`)
                        return null
                    }
                })
            )
            setPlugins(loadedPlugins.filter(plugin => plugin))
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

    return <div ref={windowRef} className="bg-white w-full rounded-xl overflow-x-hidden scrollbar-hide">
        {/* TODO: define page paths enum*/}
        <Switch>
            <Route path="/"> <WaCommand/> </Route>
            <Route path="/plugins/:entryID"> <WaPluginRender/> </Route>
        </Switch>
    </div>
}


export default Watools