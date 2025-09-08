import {CommandGroup, CommandItem} from "@/components/ui/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {PluginEntry} from "@/schemas/plugin";
import {usePluginActions} from "@/store/pluginStore";
import {useMemo} from "react";

type WaPluginCommandGroupProps = {
    searchKey: string
    OnTriggerCommand: (entry: PluginEntry) => void
}


export const WaPluginCommandGroup = (props: WaPluginCommandGroupProps) => {

    const {getPlugins} = usePluginActions()
    const plugins = getPlugins()

    const matchPluginEntries = useMemo(() => {
        return plugins.flatMap(plugin => plugin.allEntries).filter(entry => entry.match(props.searchKey))
    }, [plugins, props.searchKey])

    if (!matchPluginEntries.length) {
        return null
    }

    return <CommandGroup key='Plugin' heading="Plugin">
        {matchPluginEntries.map(entry => (
            <CommandItem
                key={entry.title}
                value={entry.title}
                className='gap-x-4'
                onSelect={() => {
                    props.OnTriggerCommand(entry)
                }}
            >
                {entry.icon && <WaIcon value={entry.icon} size={16}/>}
                <span>{entry.title}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}