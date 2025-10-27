import {useMemo} from "react";
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
    const {getEnabledPlugins, plugins} = usePluginStore();

    const enabledPlugins = useMemo(() => {
        return getEnabledPlugins();
    }, [getEnabledPlugins, plugins]);

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

    const filteredItems = useMemo((): BaseItemProps[] => {
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
            if (a.type === 'executable' && b.type === 'ui') return -1;
            if (a.type === 'ui' && b.type === 'executable') return 1;
            return 0;
        });

        return matchedEntries.slice(0, 5).map(entry => ({
            id: entry.triggerId,
            triggerId: entry.triggerId,
            name: entry.pluginName,
            icon: <WaIcon value={entry.icon} size={16}/>,
            score: entry.type === 'executable' ? 0.9 : 0.8,
            onSelect: () => {
                onTriggerPluginCommand(entry, input);
            },
            children: (
                <div className="flex items-center justify-between w-full">
                    <span className="text-sm text-gray-500">{entry.subTitle}</span>
                    <span className="text-xs text-gray-400 bg-gray-100 px-2 py-1 rounded">
                        {entry.type}
                    </span>
                </div>
            )
        }));
    }, [input, allPluginEntries, onTriggerPluginCommand]);

    return filteredItems;
};