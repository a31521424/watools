import {CommandGroup, CommandItem} from "@/components/ui/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {CommandGroupType} from "@/schemas/command";

type WaBaseCommandGroupProps = {
    commandGroup: CommandGroupType
    searchKey: string
}

export const WaBaseCommandGroup = (props: WaBaseCommandGroupProps) => {
    const filterCommandGroup = {
        category: props.commandGroup.category,
        commands: props.commandGroup.commands.filter(command => command.name.toLowerCase().includes(props.searchKey.toLowerCase()))
    }
    console.log('filterCommandGroup', filterCommandGroup)

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