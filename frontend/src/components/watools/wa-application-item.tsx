import {useMemo} from "react";
import {CommandType} from "@/schemas/command";
import {WaIcon} from "@/components/watools/wa-icon";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";
import {BaseItemProps} from "@/components/watools/wa-base-item";
import {compareRankableItems, RankingInputContext, RankingSelectionRecord} from "@/lib/command-ranking";

type UseApplicationItemsParams = {
    searchKey: string;
    rankingContext: RankingInputContext;
    rankingHistory: RankingSelectionRecord[];
    onTriggerCommand: (command: CommandType) => void;
}

export const useApplicationItems = ({
    searchKey,
    rankingContext,
    rankingHistory,
    onTriggerCommand
}: UseApplicationItemsParams) => {
    const {
        commandGroup,
        isLoading,
        searchCommands,
        updateCommandUsage
    } = useApplicationCommandStore();


    return useMemo((): BaseItemProps[] => {
        if (!searchKey || !commandGroup || isLoading) {
            return [];
        }

        const commands = searchCommands(searchKey, 10)
            .map((command, index) => ({
                command,
                rankingMeta: {
                    source: "application" as const,
                    usedCount: command.usedCount,
                    lastUsedAt: command.lastUsedAt,
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

        return commands.map(({command, rankingMeta}) => ({
            id: command.id,
            triggerId: command.triggerId,
            title: command.name,
            icon: (
                <WaIcon
                    iconPath={`/api/application-icon?path=${encodeURIComponent(command.iconPath)}`}
                    size={16}
                />
            ),
            usedCount: command.usedCount,
            rankingMeta,
            subtitle: command.path,
            onSelect: async () => {
                await updateCommandUsage(command.id);
                onTriggerCommand(command);
            }
        }));
    }, [searchKey, commandGroup, searchCommands, isLoading, updateCommandUsage, onTriggerCommand, rankingContext, rankingHistory]);
};
