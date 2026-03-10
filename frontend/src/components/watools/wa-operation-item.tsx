import {useEffect, useMemo, useState} from "react";
import {CommandType, OperationCommandType} from "@/schemas/command";
import {getOperationCommands} from "@/api/command";
import {BaseItemProps} from "@/components/watools/wa-base-item";
import Fuse from "fuse.js";
import {WaIcon} from "@/components/watools/wa-icon";
import {compareRankableItems, RankingInputContext, RankingSelectionRecord} from "@/lib/command-ranking";

const dedupeOperations = (commands: OperationCommandType[]) => {
    const uniqueCommands = new Map<string, OperationCommandType>();
    for (const command of commands) {
        if (!uniqueCommands.has(command.triggerId)) {
            uniqueCommands.set(command.triggerId, command);
        }
    }
    return Array.from(uniqueCommands.values());
}

type UseOperationItemsParams = {
    searchKey: string;
    rankingContext: RankingInputContext;
    rankingHistory: RankingSelectionRecord[];
    onTriggerCommand: (command: CommandType) => void;
}

export const useOperationItems = ({
    searchKey,
    rankingContext,
    rankingHistory,
    onTriggerCommand
}: UseOperationItemsParams) => {
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
                setOperationCommands(dedupeOperations(commandGroup.commands));
            });
        };
        initApplication();
    }, []);

    const filteredItems = useMemo((): BaseItemProps[] => {
        if (!searchKey || !operationFuse) {
            return [];
        }

        const results = operationFuse.search(searchKey, {limit: 5})
            .map((result, index) => ({
                command: result.item,
                rankingMeta: {
                    source: "operation" as const,
                    sourceOrder: index,
                }
            }))
            .sort((a, b) => compareRankableItems({
                triggerId: a.command.triggerId,
                title: a.command.name,
                rankingMeta: a.rankingMeta,
            }, {
                triggerId: b.command.triggerId,
                title: b.command.name,
                rankingMeta: b.rankingMeta,
            }, rankingContext, rankingHistory));

        return results.map(({command, rankingMeta}) => {
            return {
                id: command.triggerId,
                triggerId: command.triggerId,
                title: command.name,
                icon: <WaIcon value={command.icon} size={16}/>,
                usedCount: 0,
                rankingMeta,
                subtitle: command.description,
                badge: "Operation",
                onSelect: () => onTriggerCommand(command)
            };
        });
    }, [searchKey, operationFuse, onTriggerCommand, rankingContext, rankingHistory]);

    return filteredItems;
};
