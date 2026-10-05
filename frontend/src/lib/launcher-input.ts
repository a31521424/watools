import type {AutomaticClipboard} from '@/api/host/contracts';
import type {AppClipboardContent, AppInputValueType} from '@/schemas/app';
import type {LauncherInput} from '@/schemas/launcher';
import type {PluginContext, PluginMatchContext} from '@/schemas/plugin';
import {getSearchQuery, getTextDisplay, getTextEdges, MAX_INLINE_TEXT_LENGTH, MAX_MATCH_TEXT_EDGE_LENGTH} from '@/lib/input-text';
import {snapshotPluginContext} from '@/lib/plugin-context';

export const EMPTY_LAUNCHER_INPUT: LauncherInput = {kind: 'inline', context: {input: {value: '', valueType: 'text'}, clipboard: null}};

export const createInlineInput = (context: PluginContext): LauncherInput => ({kind: 'inline', context: snapshotPluginContext(context)});

export const createTextInput = (value: string, valueType: AppInputValueType = 'text'): LauncherInput => value.length ? {
    kind: 'inline',
    context: {
        input: {value, valueType, clipboardContentType: valueType === 'clipboard' ? 'text' : undefined},
        clipboard: valueType === 'clipboard' ? {contentType: 'text', text: value, imageBase64: null, files: null} : null,
    },
} : EMPTY_LAUNCHER_INPUT;

export const createClipboardInput = (content: AppClipboardContent): LauncherInput => content.contentType === 'text'
    ? createTextInput(content.text ?? '', 'clipboard')
    : createInlineInput({
        input: {value: '', valueType: 'clipboard', clipboardContentType: content.contentType},
        clipboard: {
            contentType: content.contentType,
            text: null,
            imageBase64: content.contentType === 'image' ? content.imageBase64 : null,
            files: content.contentType === 'files' ? content.files : null,
        },
    });

export const createAutomaticClipboardInput = (content: AutomaticClipboard): LauncherInput => content.length <= MAX_INLINE_TEXT_LENGTH
    ? createTextInput(content.text, 'clipboard')
    : {kind: 'clipboard-snapshot', snapshot: {id: content.snapshotId, length: content.length, head: content.prefix, tail: content.suffix, preview: content.preview}};

export const getClipboardSnapshotId = (input: LauncherInput): string | undefined =>
    input.kind === 'clipboard-snapshot' ? input.snapshot.id : undefined;

export const hasLauncherInput = (input: LauncherInput): boolean => {
    if (input.kind === 'clipboard-snapshot') return input.snapshot.length > 0;
    const {context} = input;
    return context.input.value.length > 0 || Boolean(context.clipboard?.imageBase64) || Boolean(context.clipboard?.files?.length);
};

export const getPluginMatchContext = (input: LauncherInput): PluginMatchContext => {
    const deferred = input.kind === 'clipboard-snapshot';
    const context = deferred ? null : input.context;
    const length = deferred ? input.snapshot.length : input.context.input.value.length;
    const {head, tail} = deferred ? {head: getTextEdges(input.snapshot.head).head, tail: getTextEdges(input.snapshot.tail).tail} : getTextEdges(input.context.input.value);
    return Object.freeze({
        input: Object.freeze({
            valueType: context?.input.valueType ?? 'clipboard',
            clipboardContentType: deferred ? 'text' : context?.input.clipboardContentType,
            length, head, tail, truncated: length > MAX_MATCH_TEXT_EDGE_LENGTH,
        }),
        clipboard: deferred
            ? Object.freeze({contentType: 'text' as const, fileCount: 0})
            : context?.clipboard ? Object.freeze({contentType: context.clipboard.contentType, fileCount: context.clipboard.files?.length ?? 0}) : null,
    });
};

/** Display and query are projections; neither is the execution payload. */
export const getLauncherPresentation = (input: LauncherInput) => {
    const snapshot = input.kind === 'clipboard-snapshot' ? input.snapshot : null;
    const context = input.kind === 'inline' ? input.context : null;
    const value = context?.input.value ?? '';
    const query = getSearchQuery(value);
    return {
        ...(snapshot ? {displayValue: snapshot.preview, compressedDisplay: true} : getTextDisplay(value)),
        query,
        rankingInput: {value: query, valueType: context?.input.valueType ?? 'clipboard', clipboardContentType: snapshot ? 'text' as const : context?.input.clipboardContentType},
        imageBase64: context?.clipboard?.imageBase64 ?? null,
        files: context?.clipboard?.files ?? null,
    };
};
