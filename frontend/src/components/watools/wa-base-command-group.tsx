import {CommandGroup, CommandItem} from "@/components/ui/command";
import {CommandGroupType, CommandType} from "@/schemas/command";
import Fuse, {IFuseOptions} from "fuse.js";
import {ReactNode, useEffect, useMemo} from "react";

type WaBaseCommandGroupProps<T extends CommandType> = {
    commandGroup: CommandGroupType<T>
    searchKey: string
    onTriggerCommand: (command: T) => void
    onSearchSuccess: (selectedKey?: string) => void
    fuseOptions: IFuseOptions<T>
    renderItemIcon: (command: T) => ReactNode
}


export const WaBaseCommandGroup = <T extends CommandType>(props: WaBaseCommandGroupProps<T>) => {
    const fuseCommand = useMemo(() => {
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
            props.onSearchSuccess(filterCommandGroup.commands.length > 0 ? filterCommandGroup.commands[0].triggerId : undefined)
        }, 0)
    }, [filterCommandGroup.commands]);

    if (filterCommandGroup.commands.length === 0) {
        return null
    }

    return <CommandGroup key={filterCommandGroup.category} heading={filterCommandGroup.category}>
        {filterCommandGroup.commands.map(command => (
            <CommandItem
                key={command.triggerId}
                value={command.triggerId}
                className='gap-x-4'
                onSelect={() => {
                    props.onTriggerCommand(command)
                }}
            >
                {props.renderItemIcon(command)}
                <span>{command.name}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}