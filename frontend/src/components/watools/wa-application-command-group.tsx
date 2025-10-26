import {useEffect, useMemo} from "react";
import {ApplicationCommandType, CommandType} from "@/schemas/command";
import {CommandGroup, CommandItem} from "@/components/ui/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";

type WaApplicationCommandGroupProps = {
    searchKey: string
    onTriggerCommand: (command: CommandType) => void
    onSearchSuccess: (selectedKey?: string) => void
}


export const WaApplicationCommandGroup = (props: WaApplicationCommandGroupProps) => {
    const {
        commandGroup,
        isLoading,
        loadCommands,
        searchCommands,
        updateCommandUsage,
        startListening,
        stopListening
    } = useApplicationCommandStore()

    useEffect(() => {
        const initializeCommands = async () => {
            try {
                await loadCommands()
            } catch (error) {
                console.error('Failed to load commands:', error)
            }
        }

        void initializeCommands()
        startListening()
        return () => {
            stopListening()
        }
    }, [loadCommands, startListening, stopListening])

    const filteredCommands = useMemo(() => {
        if (!props.searchKey || !commandGroup) {
            return []
        }
        return searchCommands(props.searchKey, 5)
    }, [props.searchKey, commandGroup, searchCommands])

    useEffect(() => {
        console.log('on Search Success', filteredCommands.length)
        setTimeout(() => {
            props.onSearchSuccess(filteredCommands.length > 0 ? filteredCommands[0].triggerId : undefined)
        }, 0)
    }, [filteredCommands, props.onSearchSuccess])

    const handleTriggerCommand = async (command: ApplicationCommandType) => {
        await updateCommandUsage(command.id)
        props.onTriggerCommand(command)
    }

    if (!props.searchKey || isLoading) {
        return null
    }

    if (filteredCommands.length === 0) {
        return null
    }

    return <CommandGroup key="Application" heading="Application">
        {filteredCommands.map(command => (
            <CommandItem
                key={command.triggerId}
                value={command.triggerId}
                className='gap-x-4'
                onSelect={() => {
                    void handleTriggerCommand(command)
                }}
            >
                <WaIcon
                    iconPath={`/api/application-icon?path=${encodeURIComponent(command.iconPath)}`}
                    size={16}
                />
                <span>{command.name}</span>
            </CommandItem>
        ))}
    </CommandGroup>
}