import {AppClipboardContent, AppClipboardContentType, AppInputValueType} from "@/schemas/app";
import {PluginContext} from "@/schemas/plugin";

const PLUGIN_LAUNCH_STORAGE_PREFIX = "watools.plugin.launch.";
const PLUGIN_LAUNCH_MAX_AGE_MS = 24 * 60 * 60 * 1000;

type StoredPluginLaunchContext = {
    context: PluginContext
    createdAt: number
}

const createEmptyPluginContext = (): PluginContext => ({
    input: {
        value: "",
        valueType: "text",
        clipboardContentType: undefined,
    },
    clipboard: null,
})

const getPluginLaunchStorage = (): Storage | null => {
    if (typeof window === "undefined" || typeof window.sessionStorage === "undefined") {
        return null;
    }
    return window.sessionStorage;
}

const createLaunchId = (): string => {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
        return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

const sanitizeClipboardContent = (
    inputValue: string,
    clipboardContentType?: AppClipboardContentType,
    clipboardImageBase64?: string | null,
    clipboardFiles?: string[] | null,
): AppClipboardContent | null => {
    if (!clipboardContentType) {
        return null;
    }

    return {
        contentType: clipboardContentType,
        text: clipboardContentType === "text" ? inputValue : null,
        imageBase64: clipboardImageBase64 ?? null,
        files: clipboardFiles ?? null,
    };
}

export const buildPluginContext = (
    inputValue: string,
    valueType: AppInputValueType,
    clipboardContentType?: AppClipboardContentType,
    clipboardImageBase64?: string | null,
    clipboardFiles?: string[] | null,
): PluginContext => ({
    input: {
        value: inputValue,
        valueType,
        clipboardContentType: clipboardContentType ?? undefined,
    },
    clipboard: sanitizeClipboardContent(inputValue, clipboardContentType, clipboardImageBase64, clipboardFiles),
})

export const createSeedPluginContext = (seed: string): PluginContext => ({
    input: {
        value: seed,
        valueType: "text",
        clipboardContentType: undefined,
    },
    clipboard: null,
})

export const hasPluginContextPayload = (context: PluginContext | null | undefined): context is PluginContext => {
    if (!context) {
        return false;
    }

    if (context.input.value.trim()) {
        return true;
    }

    if (!context.clipboard) {
        return false;
    }

    return context.clipboard.contentType === "image"
        ? Boolean(context.clipboard.imageBase64)
        : context.clipboard.contentType === "files"
            ? Array.isArray(context.clipboard.files) && context.clipboard.files.length > 0
            : Boolean(context.clipboard.text?.trim());
}

export const persistPluginLaunchContext = (context: PluginContext): string | null => {
    const storage = getPluginLaunchStorage();
    if (!storage) {
        return null;
    }

    try {
        prunePluginLaunchContexts(storage);

        const launchId = createLaunchId();
        const payload: StoredPluginLaunchContext = {
            context,
            createdAt: Date.now(),
        };

        storage.setItem(`${PLUGIN_LAUNCH_STORAGE_PREFIX}${launchId}`, JSON.stringify(payload));
        return launchId;
    } catch {
        return null;
    }
}

export const readPluginLaunchContext = (launchId: string | null | undefined): PluginContext | null => {
    if (!launchId) {
        return null;
    }

    const storage = getPluginLaunchStorage();
    if (!storage) {
        return null;
    }

    const raw = storage.getItem(`${PLUGIN_LAUNCH_STORAGE_PREFIX}${launchId}`);
    if (!raw) {
        return null;
    }

    try {
        const payload = JSON.parse(raw) as StoredPluginLaunchContext;
        if (!payload || typeof payload !== "object" || !payload.context) {
            return null;
        }
        if (!Number.isFinite(payload.createdAt) || Date.now() - payload.createdAt > PLUGIN_LAUNCH_MAX_AGE_MS) {
            storage.removeItem(`${PLUGIN_LAUNCH_STORAGE_PREFIX}${launchId}`);
            return null;
        }
        return payload.context;
    } catch {
        storage.removeItem(`${PLUGIN_LAUNCH_STORAGE_PREFIX}${launchId}`);
        return null;
    }
}

export const resolvePluginLaunchContext = ({
    launchId,
    seed,
    liveContext,
}: {
    launchId?: string | null
    seed?: string | null
    liveContext: PluginContext
}): PluginContext => {
    const storedContext = readPluginLaunchContext(launchId);
    if (storedContext) {
        return storedContext;
    }

    if (seed && seed.trim()) {
        return createSeedPluginContext(seed);
    }

    if (hasPluginContextPayload(liveContext)) {
        return liveContext;
    }

    return createEmptyPluginContext();
}

export const getLegacySeedValue = (context: PluginContext): string => context.input.value.trim();

function prunePluginLaunchContexts(storage: Storage) {
    const now = Date.now();

    for (let index = storage.length - 1; index >= 0; index -= 1) {
        const key = storage.key(index);
        if (!key || !key.startsWith(PLUGIN_LAUNCH_STORAGE_PREFIX)) {
            continue;
        }

        const raw = storage.getItem(key);
        if (!raw) {
            storage.removeItem(key);
            continue;
        }

        try {
            const payload = JSON.parse(raw) as StoredPluginLaunchContext;
            if (!Number.isFinite(payload?.createdAt) || now - payload.createdAt > PLUGIN_LAUNCH_MAX_AGE_MS) {
                storage.removeItem(key);
            }
        } catch {
            storage.removeItem(key);
        }
    }
}
