import {CommandGroup, CommandItem} from "@/components/ui/command";
import {CommandGroupType, CommandType} from "@/schemas/command";
import {WaIcon} from "@/components/watools/wa-icon";
import Fuse from "fuse.js";
import {useMemo} from "react";

type WaBaseCommandGroupProps = {
    commandGroup: CommandGroupType
    searchKey: string
    onTriggerCommand: (command: CommandType) => void
}

const WaBaseCommandFuseConfig = {
    keys: [{
        name: 'name',
        weight: 1
    }, {
        name: 'nameInitial',
        weight: 0.5
    }]
}

export const WaBaseCommandGroup = (props: WaBaseCommandGroupProps) => {
    const fuseCommand = useMemo(() => {
        console.log('commandGroup', props.commandGroup)
        return new Fuse(props.commandGroup.commands, WaBaseCommandFuseConfig)
    }, [props.commandGroup])
    const filterCommandGroup = useMemo(() => {
        return {
            category: props.commandGroup.category,
            commands: fuseCommand.search(props.searchKey).map(command => command.item)
        }
    }, [props.commandGroup, props.searchKey])
    if (filterCommandGroup.commands.length === 0) {
        return null
    }

    return <CommandGroup key={filterCommandGroup.category} heading={filterCommandGroup.category}>
        {filterCommandGroup.commands.map(command => (
            <CommandItem
                key={command.path}
                className='gap-x-4'
                onSelect={() => {
                    props.onTriggerCommand(command)
                }}
            >
                <WaIcon key={command.iconPath} value={command.icon} iconPath={command.iconPath}/>
                <span>{command.name}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}