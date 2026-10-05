import type {LauncherInput} from '@/schemas/launcher';
import type {PluginEntry, PluginMatcher, PluginMatchContext} from '@/schemas/plugin';
import {MAX_INLINE_TEXT_LENGTH} from '@/lib/input-text';

/** All pre-preview compatibility belongs here, outside the modern match path. */
export const matchesLegacyPluginEntry = (
    type: PluginEntry['type'],
    matcher: Extract<PluginMatcher, {kind: 'legacy'}>,
    input: LauncherInput,
    preview: PluginMatchContext,
): boolean => {
    if (input.kind === 'inline' && input.context.input.value.length <= MAX_INLINE_TEXT_LENGTH
        && (input.context.clipboard?.text?.length ?? 0) <= MAX_INLINE_TEXT_LENGTH) {
        const context = input.context;
        // Legacy matchers can mutate their arguments. Give each one its own
        // wrappers and file list; immutable origin strings need no deep copy.
        return matcher.match({
            input: {...context.input},
            clipboard: context.clipboard ? {
                ...context.clipboard,
                files: context.clipboard.files ? [...context.clipboard.files] : null,
            } : null,
        }) === true;
    }

    if (type === 'ui') return true;
    const head = preview.input.head.trimStart().toLowerCase();
    return matcher.largeTextPrefixes.some(prefix => head.startsWith(prefix));
};
