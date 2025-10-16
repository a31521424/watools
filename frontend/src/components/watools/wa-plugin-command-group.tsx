import {useEffect, useState} from "react";
import {CommandGroup, CommandItem} from "@/components/ui/command";
import {usePluginStore} from "@/stores";
import {PluginEntry} from "@/schemas/plugin";

type PluginCommandEntry = PluginEntry & {
    packageId: string
    pluginName: string
    triggerId: string
}

type WaPluginCommandGroupProps = {
    searchKey: string
    onTriggerPluginCommand: (entry: PluginCommandEntry, input: string) => void
    onSearchSuccess: (selectedKey?: string) => void
}

export const WaPluginCommandGroup = (props: WaPluginCommandGroupProps) => {
    const {getEnabledPlugins} = usePluginStore()
    const [matchedEntries, setMatchedEntries] = useState<PluginCommandEntry[]>([])

    useEffect(() => {
        if (!props.searchKey) {
            setMatchedEntries([])
            return
        }

        const enabledPlugins = getEnabledPlugins()
        const allEntries: PluginCommandEntry[] = []

        // 收集所有启用插件的 entry
        enabledPlugins.forEach(plugin => {
            plugin.entry.forEach((entry, index) => {
                allEntries.push({
                    ...entry,
                    packageId: plugin.packageId,
                    pluginName: plugin.name,
                    triggerId: `${plugin.packageId}_${index}`
                })
            })
        })

        // 匹配输入
        const matched = allEntries.filter(entry => {
            try {
                return entry.match(props.searchKey)
            } catch (error) {
                console.error(`Plugin match error for ${entry.packageId}:`, error)
                return false
            }
        })

        // 按优先级排序：executable 类型优先
        matched.sort((a, b) => {
            if (a.type === 'executable' && b.type === 'ui') return -1
            if (a.type === 'ui' && b.type === 'executable') return 1
            return 0
        })

        setMatchedEntries(matched.slice(0, 5)) // 限制显示5个结果
    }, [props.searchKey, getEnabledPlugins])

    useEffect(() => {
        setTimeout(() => {
            props.onSearchSuccess(matchedEntries.length > 0 ? matchedEntries[0].triggerId : undefined)
        }, 0)
    }, [matchedEntries, props])

    if (!props.searchKey || matchedEntries.length === 0) {
        return null
    }

    return (
        <CommandGroup key="Plugin" heading="Plugins">
            {matchedEntries.map(entry => (
                <CommandItem
                    key={entry.triggerId}
                    value={entry.triggerId}
                    className='gap-x-4'
                    onSelect={() => {
                        console.log('Triggering plugin command:', entry)
                        props.onTriggerPluginCommand(entry, props.searchKey)
                    }}
                >
                    <span className="text-lg">{entry.icon || '🔌'}</span>
                    <div className="flex flex-col">
                        <span>{entry.pluginName}</span>
                        <span className="text-sm text-gray-500">{entry.subTitle}</span>
                    </div>
                    <span className="ml-auto text-xs text-gray-400 bg-gray-100 px-2 py-1 rounded">
                        {entry.type}
                    </span>
                </CommandItem>
            ))}
        </CommandGroup>
    )
}