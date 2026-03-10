const HOST_STORAGE_KEY = "history";
const LOCAL_HISTORY_KEY = "watools.plugin.calculator.history";
const LOCAL_DRAFT_KEY = "watools.plugin.calculator.draft";
const MAX_HISTORY = 50;

const canUseLocalStorage = () => {
    try {
        return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
    } catch (error) {
        return false;
    }
};

const normalizeHistoryItem = (item) => {
    if (!item || typeof item !== "object") {
        return null;
    }

    const expression = typeof item.expression === "string" ? item.expression.trim() : "";
    const result = item.result === undefined || item.result === null ? "" : String(item.result).trim();
    const createdAt = typeof item.createdAt === "string" && item.createdAt
        ? item.createdAt
        : new Date().toISOString();

    if (!expression || !result) {
        return null;
    }

    return {
        expression,
        result,
        createdAt
    };
};

const normalizeHistory = (items) => {
    if (!Array.isArray(items)) {
        return [];
    }

    return items
        .map(normalizeHistoryItem)
        .filter((item) => item !== null)
        .slice(0, MAX_HISTORY);
};

const readLocalHistory = () => {
    if (!canUseLocalStorage()) {
        return [];
    }

    try {
        const raw = window.localStorage.getItem(LOCAL_HISTORY_KEY);
        if (!raw) {
            return [];
        }
        return normalizeHistory(JSON.parse(raw));
    } catch (error) {
        return [];
    }
};

const writeLocalHistory = (history) => {
    if (!canUseLocalStorage()) {
        return;
    }

    try {
        window.localStorage.setItem(LOCAL_HISTORY_KEY, JSON.stringify(normalizeHistory(history)));
    } catch (error) {
        // Ignore local persistence failures.
    }
};

const readHostHistory = async () => {
    try {
        return normalizeHistory(await window.watools.StorageGet(HOST_STORAGE_KEY));
    } catch (error) {
        return [];
    }
};

const writeHostHistory = async (history) => {
    try {
        await window.watools.StorageSet(HOST_STORAGE_KEY, normalizeHistory(history));
    } catch (error) {
        // Ignore host persistence failures and keep local fallback.
    }
};

export const loadHistory = async () => {
    const hostHistory = await readHostHistory();
    if (hostHistory.length > 0) {
        writeLocalHistory(hostHistory);
        return hostHistory;
    }

    const localHistory = readLocalHistory();
    if (localHistory.length > 0) {
        void writeHostHistory(localHistory);
    }
    return localHistory;
};

export const saveHistory = async (history) => {
    const nextHistory = normalizeHistory(history);
    writeLocalHistory(nextHistory);
    await writeHostHistory(nextHistory);
    return nextHistory;
};

export const appendHistory = async (expression, result) => {
    const safeExpression = typeof expression === "string" ? expression.trim() : "";
    const safeResult = result === undefined || result === null ? "" : String(result).trim();

    if (!safeExpression || !safeResult) {
        return loadHistory();
    }

    const current = await loadHistory();
    const nextHistory = [{
        expression: safeExpression,
        result: safeResult,
        createdAt: new Date().toISOString()
    }, ...current.filter((item) => !(item.expression === safeExpression && item.result === safeResult))].slice(0, MAX_HISTORY);

    await saveHistory(nextHistory);
    return nextHistory;
};

export const clearHistory = async () => {
    return saveHistory([]);
};

export const loadDraft = () => {
    if (!canUseLocalStorage()) {
        return "";
    }

    try {
        return window.localStorage.getItem(LOCAL_DRAFT_KEY) || "";
    } catch (error) {
        return "";
    }
};

export const saveDraft = (value) => {
    if (!canUseLocalStorage()) {
        return;
    }

    try {
        if (value) {
            window.localStorage.setItem(LOCAL_DRAFT_KEY, value);
            return;
        }

        window.localStorage.removeItem(LOCAL_DRAFT_KEY);
    } catch (error) {
        // Ignore local persistence failures.
    }
};

export const removeDraft = () => {
    saveDraft("");
};
