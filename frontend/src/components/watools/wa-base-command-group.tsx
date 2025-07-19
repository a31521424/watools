import {CommandGroup, CommandItem} from "@/components/ui/command";
import {CommandGroupType, CommandType} from "@/schemas/command";
import {WaIcon} from "@/components/watools/wa-icon";
import Fuse, {IFuseOptions} from "fuse.js";
import {useEffect, useMemo} from "react";

type WaBaseCommandGroupProps<T extends CommandType> = {
    commandGroup: CommandGroupType<T>
    searchKey: string
    onTriggerCommand: (command: T) => void
    onSearchSuccess: () => void
    fuseOptions: IFuseOptions<T>
}


export const WaBaseCommandGroup = <T extends CommandType>(props: WaBaseCommandGroupProps<T>) => {
    const fuseCommand = useMemo(() => {
        console.log('commandGroup', props.commandGroup)
        return new Fuse(props.commandGroup.commands, props.fuseOptions)
    }, [props.commandGroup])
    const filterCommandGroup = useMemo(() => {
        return {
            category: props.commandGroup.category,
            commands: fuseCommand.search(props.searchKey, {limit: 5}).map(command => command.item)
        }
    }, [props.commandGroup, props.searchKey])
    useEffect(() => {
        console.log('on Search Success', filterCommandGroup.commands.length)
        setTimeout(() => {
            props.onSearchSuccess()
        }, 0)
    }, [filterCommandGroup.commands]);

    if (filterCommandGroup.commands.length === 0) {
        return null
    }

    return <CommandGroup key={filterCommandGroup.category} heading={filterCommandGroup.category}>
        {filterCommandGroup.commands.map(command => (
            <CommandItem
                key={command.id}
                value={`${command.id}-${command.name}`}
                className='gap-x-4'
                onSelect={() => {
                    props.onTriggerCommand(command)
                }}
            >
                <WaIcon key={command.iconPath} iconPath={command.iconPath}/>
                <span>{command.name}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}