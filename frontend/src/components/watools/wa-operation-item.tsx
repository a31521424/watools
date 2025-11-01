import {useEffect, useMemo, useState} from "react";
import {CommandType, OperationCommandType} from "@/schemas/command";
import {getOperationCommands} from "@/api/command";
import {BaseItemProps} from "@/components/watools/wa-base-item";
import Fuse from "fuse.js";
import {WaIcon} from "@/components/watools/wa-icon";

type UseOperationItemsParams = {
    searchKey: string;
    onTriggerCommand: (command: CommandType) => void;
}

export const useOperationItems = ({searchKey, onTriggerCommand}: UseOperationItemsParams) => {
    const [operationCommands, setOperationCommands] = useState<OperationCommandType[]>([]);

    const operationFuse = useMemo(() => {
        if (operationCommands.length === 0) return null;
        return new Fuse(operationCommands, {
            threshold: 0.4,
            minMatchCharLength: 1,
            useExtendedSearch: true,
            ignoreLocation: true,
            shouldSort: false,
            keys: [{name: 'name', weight: 1.0}]
        });
    }, [operationCommands]);

    useEffect(() => {
        const initApplication = () => {
            getOperationCommands().then(commandGroup => {
                setOperationCommands(commandGroup.commands);
            });
        };
        initApplication();
    }, []);

    const filteredItems = useMemo((): BaseItemProps[] => {
        if (!searchKey || !operationFuse) {
            return [];
        }

        const results = operationFuse.search(searchKey, {limit: 5});
        return results.map(result => {
            const command = result.item;
            return {
                id: command.triggerId,
                triggerId: command.triggerId,
                name: command.name,
                icon: <WaIcon value={command.icon} size={16}/>,
                usedCount: 0,
                subtitle: command.description,
                badge: "Operation",
                onSelect: () => onTriggerCommand(command)
            };
        });
    }, [searchKey, operationFuse, onTriggerCommand]);

    return filteredItems;
};