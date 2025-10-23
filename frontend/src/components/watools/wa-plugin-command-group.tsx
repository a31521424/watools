import {useEffect, useState} from "react";
import {CommandGroup, CommandItem} from "@/components/ui/command";
import {usePluginStore} from "@/stores";
import {PluginEntry, PluginInput} from "@/schemas/plugin";
import {WaIcon} from "@/components/watools/wa-icon";

export type PluginCommandEntry = PluginEntry & {
    packageId: string
    pluginName: string
    triggerId: string
    homeUrl: string
}

type WaPluginCommandGroupProps = {
    input: PluginInput
    onTriggerPluginCommand: (entry: PluginCommandEntry, input: PluginInput) => void
    onSearchSuccess: (selectedKey?: string) => void
}

export const WaPluginCommandGroup = (props: WaPluginCommandGroupProps) => {
    const {getEnabledPlugins} = usePluginStore()
    const [matchedEntries, setMatchedEntries] = useState<PluginCommandEntry[]>([])

    useEffect(() => {
        if (!props.input) {
            setMatchedEntries([])
            return
        }

        const enabledPlugins = getEnabledPlugins()
        const allEntries: PluginCommandEntry[] = []

        // Collect all enabled plugin entries
        enabledPlugins.forEach(plugin => {
            plugin.entry.forEach((entry) => {
                allEntries.push({
                    ...entry,
                    packageId: plugin.packageId,
                    pluginName: plugin.name,
                    triggerId: `${plugin.packageId}_${entry.subTitle}`,
                    homeUrl: plugin.homeUrl
                })
            })
        })

        // Match input
        const matched = allEntries.filter(entry => {
            try {
                return entry.match(props.input)
            } catch (error) {
                console.error(`Plugin match error for ${entry.packageId}:`, error)
                return false
            }
        })

        // Sort by priority: executable type first
        matched.sort((a, b) => {
            if (a.type === 'executable' && b.type === 'ui') return -1
            if (a.type === 'ui' && b.type === 'executable') return 1
            return 0
        })

        setMatchedEntries(matched.slice(0, 5)) // Limit display to 5 results
    }, [props.input, getEnabledPlugins])

    useEffect(() => {
        setTimeout(() => {
            props.onSearchSuccess(matchedEntries.length > 0 ? matchedEntries[0].triggerId : undefined)
        }, 0)
    }, [matchedEntries, props])

    if (matchedEntries.length === 0) {
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
                        props.onTriggerPluginCommand(entry, props.input)
                    }}
                >
                    <WaIcon value={entry.icon} size={16}/>
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