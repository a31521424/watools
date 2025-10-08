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
import {PluginPackage} from "@/schemas/plugin";

const Watools = () => {
    const windowRef = useElementResize<HTMLDivElement>({
        onResize: resizeWindowHeight
    })
    const {setPlugins} = usePluginActions()

    useEffect(() => {
        (async () => {
            const allPlugins = await GetPluginsApi()
            const loadedPlugins: (PluginPackage | null)[] = await Promise.all(
                allPlugins.map(async plugin => {
                    try {
                        let fileEntry = await GetPluginExecEntryApi(plugin.id)
                        fileEntry = `/api/plugin-entry?path=${encodeURIComponent(fileEntry)}&timestamp=${Date.now()}`
                        const response = await fetch(fileEntry)
                        if (!response.ok) {
                            Logger.error(`Failed to fetch plugin ${plugin.packageID}`)
                            return null
                        }
                        const pluginCode = await response.text()
                        new Function(pluginCode)()

                        // @ts-ignore
                        const pluginModule = window.WailsAppPlugins[plugin.packageID]

                        return {
                            allEntries: pluginModule.allEntries,
                            metadata: plugin,
                        } as PluginPackage
                    } catch (e) {
                        Logger.error(`Failed to load plugin ${plugin.packageID}: ${e}`)
                        return null
                    }
                })
            )
            setPlugins(loadedPlugins.filter((p): p is PluginPackage => p !== null))
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