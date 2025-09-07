import {CommandGroup, CommandItem} from "@/components/ui/command";
import {usePlugins} from "@/hooks/usePlugins";
import {WaIcon} from "@/components/watools/wa-icon";
import {PluginEntry} from "@/schemas/plugin";

type WaPluginCommandGroupProps = {
    searchKey: string
    OnTriggerCommand: (entry: PluginEntry) => void
}


export const WaPluginCommandGroup = (props: WaPluginCommandGroupProps) => {

    const pluginEntries = usePlugins({input: props.searchKey})
    if (!pluginEntries.length) {
        return null
    }

    return <CommandGroup key='Plugin' heading="Plugin">
        {pluginEntries.map(entry => (
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