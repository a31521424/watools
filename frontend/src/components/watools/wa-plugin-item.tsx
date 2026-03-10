import {useMemo} from "react";
import {usePluginStore} from "@/stores";
import {PluginContext, PluginEntry} from "@/schemas/plugin";
import {WaIcon} from "@/components/watools/wa-icon";
import {AppClipboardContent, AppInput} from "@/schemas/app";
import {BaseItemProps} from "@/components/watools/wa-base-item";
import {compareRankableItems, RankingInputContext, RankingSelectionRecord} from "@/lib/command-ranking";

export type PluginCommandEntry = PluginEntry & {
    packageId: string;
    pluginName: string;
    triggerId: string;
    homeUrl: string;
}

type UsePluginItemsParams = {
    input: AppInput;
    clipboard: AppClipboardContent | null;
    rankingContext: RankingInputContext;
    rankingHistory: RankingSelectionRecord[];
    onTriggerPluginCommand: (entry: PluginCommandEntry, context: PluginContext) => void;
}

export const usePluginItems = ({
    input,
    clipboard,
    rankingContext,
    rankingHistory,
    onTriggerPluginCommand
}: UsePluginItemsParams) => {
    const {getEnabledPlugins, plugins} = usePluginStore();

    const enabledPlugins = useMemo(() => {
        return getEnabledPlugins();
    }, [plugins]);

    const allPluginEntries = useMemo(() => {
        const entries: PluginCommandEntry[] = [];
        enabledPlugins.forEach(plugin => {
            plugin.entry.forEach((entry, index) => {
                entries.push({
                    ...entry,
                    packageId: plugin.packageId,
                    pluginName: plugin.name,
                    triggerId: `${plugin.packageId}:${entry.type}:${entry.type === 'ui' ? entry.file : entry.subTitle}:${index}`,
                    homeUrl: plugin.homeUrl
                });
            });
        });
        return entries;
    }, [enabledPlugins]);

    // Create context object once
    const context: PluginContext = useMemo(() => ({
        input,
        clipboard,
    }), [input, clipboard]);

    return useMemo((): BaseItemProps[] => {
        if ((!input.value && !input.clipboardContentType) || allPluginEntries.length === 0) {
            return [];
        }

        const matchedEntries = allPluginEntries.filter(entry => {
            try {
                return entry.match(context);
            } catch (error) {
                console.error(`Plugin match error for ${entry.packageId}:`, error);
                return false;
            }
        });

        const uniqueEntries = new Map<string, PluginCommandEntry>();
        for (const entry of matchedEntries) {
            if (!uniqueEntries.has(entry.triggerId)) {
                uniqueEntries.set(entry.triggerId, entry);
            }
        }

        const sortedEntries = Array.from(uniqueEntries.values())
            .map((entry, index) => ({
                entry,
                plugin: enabledPlugins.find(p => p.packageId === entry.packageId),
                rankingMeta: {
                    source: "plugin" as const,
                    usedCount: enabledPlugins.find(p => p.packageId === entry.packageId)?.usedCount || 0,
                    lastUsedAt: enabledPlugins.find(p => p.packageId === entry.packageId)?.lastUsedAt || null,
                    sourceOrder: index,
                }
            }))
            .sort((a, b) => {
                const rankCompare = compareRankableItems({
                    triggerId: a.entry.triggerId,
                    title: a.entry.subTitle,
                    rankingMeta: a.rankingMeta,
                }, {
                    triggerId: b.entry.triggerId,
                    title: b.entry.subTitle,
                    rankingMeta: b.rankingMeta,
                }, rankingContext, rankingHistory);

                if (rankCompare !== 0) {
                    return rankCompare;
                }

                if (a.entry.type === 'executable' && b.entry.type === 'ui') return -1;
                if (a.entry.type === 'ui' && b.entry.type === 'executable') return 1;
                return a.entry.subTitle.localeCompare(b.entry.subTitle);
            });

        return sortedEntries.slice(0, 3).map(({entry, plugin, rankingMeta}) => ({
                id: entry.triggerId,
                triggerId: entry.triggerId,
                title: entry.subTitle,
                icon: <WaIcon value={entry.icon} size={16}/>,
                usedCount: plugin?.usedCount || 0,
                rankingMeta,
                subtitle: entry.pluginName,
                badge: entry.type,
                onSelect: () => {
                    onTriggerPluginCommand(entry, context);
                }
            }));
    }, [input, clipboard, allPluginEntries, onTriggerPluginCommand, context, enabledPlugins, rankingContext, rankingHistory]);
};
