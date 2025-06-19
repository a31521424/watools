import {CommandGroup, CommandItem} from "@/components/ui/command";
import {CommandGroupType} from "@/schemas/command";
import {WaIcon} from "@/components/watools/wa-icon";

type WaBaseCommandGroupProps = {
    commandGroup: CommandGroupType
    searchKey: string
}

export const WaBaseCommandGroup = (props: WaBaseCommandGroupProps) => {
    const filterCommandGroup = {
        category: props.commandGroup.category,
        commands: props.commandGroup.commands.filter(command => command.name.toLowerCase().includes(props.searchKey.toLowerCase()))
    }
    if (filterCommandGroup.commands.length === 0) {
        return null
    }

    return <CommandGroup key={filterCommandGroup.category} heading={filterCommandGroup.category}>
        {filterCommandGroup.commands.map(command => (
            <CommandItem
                key={command.name}
                className='gap-x-4'
            >
                <WaIcon key={command.iconPath} value={command.icon} iconPath={command.iconPath}/>
                <span>{command.name}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}