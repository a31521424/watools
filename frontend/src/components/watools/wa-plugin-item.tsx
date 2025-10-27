import {useEffect, useMemo} from "react";
import {usePluginStore} from "@/stores";
import {PluginEntry} from "@/schemas/plugin";
import {WaIcon} from "@/components/watools/wa-icon";
import {AppInput} from "@/schemas/app";
import {BaseItemProps} from "@/components/watools/wa-base-item";

export type PluginCommandEntry = PluginEntry & {
    packageId: string;
    pluginName: string;
    triggerId: string;
    homeUrl: string;
}

type UsePluginItemsParams = {
    input: AppInput;
    onTriggerPluginCommand: (entry: PluginCommandEntry, input: AppInput) => void;
}

export const usePluginItems = ({input, onTriggerPluginCommand}: UsePluginItemsParams) => {
    const {getEnabledPlugins, plugins, fetchPlugins} = usePluginStore();

    useEffect(() => {
        void fetchPlugins()
    }, []);

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

    return useMemo((): BaseItemProps[] => {
        if (!input.value || allPluginEntries.length === 0) {
            return [];
        }

        const matchedEntries = allPluginEntries.filter(entry => {
            try {
                return entry.match(input);
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

        return matchedEntries.slice(0, 5).map(entry => {
            const plugin = enabledPlugins.find(p => p.packageId === entry.packageId);

            return {
                id: entry.triggerId,
                triggerId: entry.triggerId,
                name: entry.pluginName,
                icon: <WaIcon value={entry.icon} size={16}/>,
                usedCount: plugin?.usedCount || 0,
                subtitle: entry.subTitle,
                badge: entry.type,
                onSelect: () => {
                    onTriggerPluginCommand(entry, input);
                }
            };
        });
    }, [input, allPluginEntries, onTriggerPluginCommand]);
};