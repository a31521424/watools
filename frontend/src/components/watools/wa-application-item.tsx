import {useEffect, useMemo} from "react";
import {CommandType} from "@/schemas/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";
import {BaseItemProps} from "@/components/watools/wa-base-item";

type UseApplicationItemsParams = {
    searchKey: string;
    onTriggerCommand: (command: CommandType) => void;
}

export const useApplicationItems = ({searchKey, onTriggerCommand}: UseApplicationItemsParams) => {
    const {
        commandGroup,
        isLoading,
        loadCommands,
        searchCommands,
        updateCommandUsage,
        startListening,
        stopListening
    } = useApplicationCommandStore();

    useEffect(() => {
        const initializeCommands = async () => {
            try {
                await loadCommands();
            } catch (error) {
                console.error('Failed to load commands:', error);
            }
        };

        void initializeCommands();
        startListening();
        return () => {
            stopListening();
        };
    }, [loadCommands, startListening, stopListening]);

    const filteredItems = useMemo((): BaseItemProps[] => {
        if (!searchKey || !commandGroup || isLoading) {
            return [];
        }

        const commands = searchCommands(searchKey, 15);
        return commands.map(command => ({
            id: command.id,
            triggerId: command.triggerId,
            name: command.name,
            icon: (
                <WaIcon
                    iconPath={`/api/application-icon?path=${encodeURIComponent(command.iconPath)}`}
                    size={16}
                />
            ),
            usedCount: command.usedCount,
            subtitle: command.path,
            onSelect: async () => {
                await updateCommandUsage(command.id);
                onTriggerCommand(command);
            }
        }));
    }, [searchKey, commandGroup, searchCommands, isLoading, updateCommandUsage, onTriggerCommand]);

    return filteredItems;
};