import {useEffect, useMemo, useState} from "react";
import {PluginEntry, PluginPackage} from "@/schemas/plugin";
import {GetPluginExecEntryApi, GetPluginsApi} from "../../wailsjs/go/coordinator/WaAppCoordinator";

type UsePluginsProps = {
    input: string
}
export const usePlugins = (props: UsePluginsProps) => {
    const [plugins, setPlugins] = useState<PluginPackage[]>([])
    useEffect(() => {
        (async () => {
            const allPlugins = await GetPluginsApi()
            const loadedPlugins = await Promise.all(
                allPlugins.map(async plugin => {
                    let execEntry = await GetPluginExecEntryApi(plugin.id)
                    execEntry = `/api/plugin-entry?path=${encodeURIComponent(execEntry)}`
                    const module: { default: PluginPackage } = await import(/* @vite-ignore */ execEntry)
                    return module.default
                })
            )
            setPlugins(loadedPlugins)
        })()
    }, [])

    return useMemo(() => plugins.flatMap(plugin => plugin.allEntries?.filter((entry: PluginEntry) => entry.match(props.input))), [plugins, props.input])
}