import type {PluginEntry} from '@/schemas/plugin';
import type {LauncherInput} from '@/schemas/launcher';
import {getPluginMatchContext} from '@/lib/launcher-input';
import {matchesLegacyPluginEntry} from '@/lib/legacy-plugin-matcher';

/** Build one bounded preview for the whole pass; never materialise origin here. */
export const filterPluginEntries = <T extends PluginEntry>(entries: readonly T[], input: LauncherInput): T[] => {
    const preview = getPluginMatchContext(input);
    return entries.filter(entry => {
        try {
            return entry.matcher.kind === 'preview'
                ? entry.matcher.match(preview) === true
                : matchesLegacyPluginEntry(entry.type, entry.matcher, input, preview);
        } catch {
            // A broken plugin must not suppress the remaining candidates or
            // expose clipboard content in its exception message.
            console.error('Plugin entry matcher failed', {type: entry.type, subTitle: entry.subTitle});
            return false;
        }
    });
};
