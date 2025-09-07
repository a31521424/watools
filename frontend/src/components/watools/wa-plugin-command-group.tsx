import {CommandGroup, CommandItem} from "@/components/ui/command";
import {usePlugins} from "@/hooks/usePlugins";
import {WaIcon} from "@/components/watools/wa-icon";

type WaPluginCommandGroupProps = {
    searchKey: string
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
                    entry.exec && entry.exec(props.searchKey)
                }}
            >
                {entry.icon && <WaIcon value={entry.icon} size={16}/>}
                <span>{entry.title}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}