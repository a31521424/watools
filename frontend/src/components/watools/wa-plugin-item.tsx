import {useMemo, useRef} from "react";
import {usePluginStore} from "@/stores";
import {PluginEntry} from "@/schemas/plugin";
import {WaIcon} from "@/components/watools/wa-icon";
import type {LauncherInput} from '@/schemas/launcher';
import {BaseItemProps} from "@/components/watools/wa-base-item";
import {compareRankableItems, RankingInputContext, RankingSelectionRecord} from "@/lib/command-ranking";
import {filterPluginEntries} from '@/lib/plugin-matching';
import {hasLauncherInput} from '@/lib/launcher-input';
import {MAX_INLINE_TEXT_LENGTH} from '@/lib/input-text';

export type PluginCommandEntry = PluginEntry & {
    packageId: string;
    pluginName: string;
    triggerId: string;
    homeUrl: string;
}

type UsePluginItemsParams = {
    input: LauncherInput;
    rankingContext: RankingInputContext;
    rankingHistory: RankingSelectionRecord[];
    onTriggerPluginCommand: (entry: PluginCommandEntry) => void;
}

export const usePluginItems = ({
    input,
    rankingContext,
    rankingHistory,
    onTriggerPluginCommand
}: UsePluginItemsParams) => {
    const {getEnabledPlugins, plugins} = usePluginStore();

    const enabledPlugins = useMemo(() => {
        return getEnabledPlugins();
    }, [plugins]);

    const entryCache = useRef<{sources: typeof enabledPlugins; entries: PluginCommandEntry[]} | null>(null);

    const allPluginEntries = useMemo(() => {
        const previous = entryCache.current;
        if (previous && previous.sources.length === enabledPlugins.length && enabledPlugins.every((plugin, index) => {
            const old = previous.sources[index];
            return plugin.packageId === old.packageId && plugin.entry === old.entry && plugin.name === old.name && plugin.homeUrl === old.homeUrl;
        })) return previous.entries;
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
        entryCache.current = {sources: enabledPlugins, entries};
        return entries;
    }, [enabledPlugins]);
    const pluginById = useMemo(() => new Map(enabledPlugins.map(plugin => [plugin.packageId, plugin])), [enabledPlugins]);

    const matchedEntries = useMemo(() => hasLauncherInput(input) ? filterPluginEntries(allPluginEntries, input) : [], [input, allPluginEntries]);
    const textLength = input.kind === 'clipboard-snapshot' ? input.snapshot.length : input.context.input.value.length;

    return useMemo((): BaseItemProps[] => {

        const uniqueEntries = new Map<string, PluginCommandEntry>();
        for (const entry of matchedEntries) {
            if (!uniqueEntries.has(entry.triggerId)) {
                uniqueEntries.set(entry.triggerId, entry);
            }
        }

        const sortedEntries = Array.from(uniqueEntries.values())
            .map((entry, index) => ({
                entry,
                plugin: pluginById.get(entry.packageId),
                rankingMeta: {
                    source: "plugin" as const,
                    usedCount: pluginById.get(entry.packageId)?.usedCount || 0,
                    lastUsedAt: pluginById.get(entry.packageId)?.lastUsedAt || null,
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

        return (textLength > MAX_INLINE_TEXT_LENGTH ? sortedEntries : sortedEntries.slice(0, 3)).map(({entry, plugin, rankingMeta}) => ({
                id: entry.triggerId,
                triggerId: entry.triggerId,
                title: entry.subTitle,
                icon: <WaIcon value={entry.icon} size={16}/>,
                usedCount: plugin?.usedCount || 0,
                rankingMeta,
                subtitle: entry.pluginName,
                badge: entry.type,
                onSelect: () => {
                    onTriggerPluginCommand(entry);
                }
            }));
    }, [matchedEntries, onTriggerPluginCommand, pluginById, textLength, rankingContext, rankingHistory]);
};
