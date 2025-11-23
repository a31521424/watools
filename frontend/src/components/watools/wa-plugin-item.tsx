import {useCallback, useMemo} from "react";
import {usePluginStore} from "@/stores";
import {PluginEntry} from "@/schemas/plugin";
import {WaIcon} from "@/components/watools/wa-icon";
import {AppClipboardContent, AppInput} from "@/schemas/app";
import {BaseItemProps} from "@/components/watools/wa-base-item";

export type PluginCommandEntry = PluginEntry & {
    packageId: string;
    pluginName: string;
    triggerId: string;
    homeUrl: string;
}

type UsePluginItemsParams = {
    input: AppInput;
    clipboardAccessor?: ClipboardAccessor;
    onTriggerPluginCommand: (entry: PluginCommandEntry, input: AppInput, getClipboardContent: () => AppClipboardContent | null) => void;
}

type ClipboardAccessor = {
    readonly content: AppClipboardContent | null;
}

export const usePluginItems = ({input, onTriggerPluginCommand, clipboardAccessor}: UsePluginItemsParams) => {
    const {getEnabledPlugins, plugins} = usePluginStore();

    const enabledPlugins = useMemo(() => {
        return getEnabledPlugins();
    }, [plugins]);

    const allPluginEntries = useMemo(() => {
        const entries: PluginCommandEntry[] = [];
        enabledPlugins.forEach(plugin => {
            plugin.entry.forEach((entry) => {
                entries.push({
                    ...entry,
                    packageId: plugin.packageId,
                    pluginName: plugin.name,
                    triggerId: `${plugin.packageId}_${entry.subTitle}`,
                    homeUrl: plugin.homeUrl
                });
            });
        });
        return entries;
    }, [enabledPlugins]);

    const getClipboardContent = useCallback(() => clipboardAccessor?.content ?? null, [clipboardAccessor]);

    return useMemo((): BaseItemProps[] => {
        if ((!input.value && !input.clipboardContentType) || allPluginEntries.length === 0) {
            return [];
        }

        const matchedEntries = allPluginEntries.filter(entry => {
            try {
                return entry.match(input, getClipboardContent);
            } catch (error) {
                console.error(`Plugin match error for ${entry.packageId}:`, error);
                return false;
            }
        });

        matchedEntries.sort((a, b) => {
            const pluginA = enabledPlugins.find(p => p.packageId === a.packageId);
            const pluginB = enabledPlugins.find(p => p.packageId === b.packageId);

            const usedCountA = pluginA?.usedCount || 0;
            const usedCountB = pluginB?.usedCount || 0;

            if (usedCountA !== usedCountB) {
                return usedCountB - usedCountA;
            }

            if (a.type === 'executable' && b.type === 'ui') return -1;
            if (a.type === 'ui' && b.type === 'executable') return 1;
            return 0;
        });

        return matchedEntries.slice(0, 3).map(entry => {
            const plugin = enabledPlugins.find(p => p.packageId === entry.packageId);

            return {
                id: entry.triggerId,
                triggerId: entry.triggerId,
                title: entry.subTitle,
                icon: <WaIcon value={entry.icon} size={16}/>,
                usedCount: plugin?.usedCount || 0,
                subtitle: entry.pluginName,
                badge: entry.type,
                onSelect: () => {
                    onTriggerPluginCommand(entry, input, getClipboardContent);
                }
            };
        });
    }, [input, allPluginEntries, onTriggerPluginCommand, getClipboardContent]);
};