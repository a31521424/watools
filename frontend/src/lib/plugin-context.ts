import {PluginContext} from "@/schemas/plugin";

const PLUGIN_LAUNCH_STORAGE_PREFIX = "watools.plugin.launch.";
const PLUGIN_LAUNCH_MAX_AGE_MS = 5 * 60 * 1000;

type StoredPluginLaunchContext = {
    context: PluginContext
    createdAt: number
}
let expiryTimer: ReturnType<typeof setTimeout> | undefined;
let legacyRecordsRemoved = false;

const createEmptyPluginContext = (): PluginContext => ({
    input: {
        value: "",
        valueType: "text",
        clipboardContentType: undefined,
    },
    clipboard: null,
})

const launchContexts = new Map<string, StoredPluginLaunchContext>();

export const snapshotPluginContext = (context: PluginContext): PluginContext => ({
    input: {...context.input},
    clipboard: context.clipboard ? {...context.clipboard, files: context.clipboard.files?.slice() ?? null} : null,
});

const scheduleExpiry = () => {
    if (expiryTimer !== undefined) clearTimeout(expiryTimer);
    expiryTimer = undefined;
    if (!launchContexts.size) return;
    const deadline = Math.min(...Array.from(launchContexts.values(), item => item.createdAt + PLUGIN_LAUNCH_MAX_AGE_MS));
    expiryTimer = setTimeout(() => {
        const now = Date.now();
        for (const [id, item] of launchContexts) {
            if (now - item.createdAt >= PLUGIN_LAUNCH_MAX_AGE_MS) launchContexts.delete(id);
        }
        scheduleExpiry();
    }, Math.max(1, deadline - Date.now()));
};

const createLaunchId = (): string => {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
        return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

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

    if (context.input.value.length) {
        return true;
    }

    if (!context.clipboard) {
        return false;
    }

    return context.clipboard.contentType === "image"
        ? Boolean(context.clipboard.imageBase64)
        : context.clipboard.contentType === "files"
            ? Array.isArray(context.clipboard.files) && context.clipboard.files.length > 0
            : Boolean(context.clipboard.text?.length);
}

export const persistPluginLaunchContext = (context: PluginContext): string => {
    const now = Date.now();
    for (const [id, stored] of launchContexts) {
        if (now - stored.createdAt > PLUGIN_LAUNCH_MAX_AGE_MS) launchContexts.delete(id);
    }
    while (launchContexts.size >= 20) launchContexts.delete(launchContexts.keys().next().value!);
    // Remove plaintext launch records left by older host versions.
    try {
        if (!legacyRecordsRemoved && typeof window !== 'undefined') {
            for (let index = window.sessionStorage.length - 1; index >= 0; index--) {
                const key = window.sessionStorage.key(index);
                if (key?.startsWith(PLUGIN_LAUNCH_STORAGE_PREFIX)) window.sessionStorage.removeItem(key);
            }
            legacyRecordsRemoved = true;
        }
    } catch { /* session storage may be unavailable; in-memory delivery still works */ }
    const id = createLaunchId();
    const snapshot = snapshotPluginContext(context);
    // Bound aggregate retained text, while allowing one large current launch.
    let retainedLength = snapshot.input.value.length + (snapshot.clipboard?.text?.length ?? 0) + (snapshot.clipboard?.imageBase64?.length ?? 0);
    for (const item of launchContexts.values()) retainedLength += item.context.input.value.length + (item.context.clipboard?.text?.length ?? 0) + (item.context.clipboard?.imageBase64?.length ?? 0);
    while (retainedLength > 32 * 1024 * 1024 && launchContexts.size) {
        const oldest = launchContexts.keys().next().value!;
        const item = launchContexts.get(oldest)!.context;
        retainedLength -= item.input.value.length + (item.clipboard?.text?.length ?? 0) + (item.clipboard?.imageBase64?.length ?? 0);
        launchContexts.delete(oldest);
    }
    launchContexts.set(id, {context: snapshot, createdAt: now});
    scheduleExpiry();
    return id;
}

export const readPluginLaunchContext = (launchId: string | null | undefined): PluginContext | null => {
    if (!launchId) return null;
    const stored = launchContexts.get(launchId);
    if (!stored || Date.now() - stored.createdAt > PLUGIN_LAUNCH_MAX_AGE_MS) {
        launchContexts.delete(launchId);
        scheduleExpiry();
        return null;
    }
    return stored.context;
}

export const releasePluginLaunchContext = (launchId: string | null | undefined) => {
    if (launchId) launchContexts.delete(launchId);
    scheduleExpiry();
}

export const resolvePluginLaunchContext = ({
    launchId,
    seed,
    liveContext,
}: {
    launchId?: string | null
    seed?: string | null
    liveContext: PluginContext | null
}): PluginContext => {
    const storedContext = readPluginLaunchContext(launchId);
    if (storedContext) {
        return storedContext;
    }
    if (launchId) throw new Error('插件启动上下文已释放或过期，请返回启动器重试');

    if (seed && seed.trim()) {
        return createSeedPluginContext(seed);
    }

    if (liveContext === null) throw new Error('请从启动器选择插件以载入剪贴板原文');

    if (hasPluginContextPayload(liveContext)) {
        return liveContext;
    }

    return createEmptyPluginContext();
}
