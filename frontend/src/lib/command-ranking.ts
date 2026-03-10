import {AppInput} from "@/schemas/app";

export type RankingSourceType = "application" | "plugin" | "operation" | "app-feature";

export type RankingInputContext = {
    key: string;
    normalizedValue: string;
    valueType: string;
    clipboardContentType?: string;
}

export type RankingSelectionRecord = {
    triggerId: string;
    source: RankingSourceType;
    inputKey: string;
    normalizedValue: string;
    valueType: string;
    clipboardContentType?: string;
    selectedAt: string;
}

export type RankingMeta = {
    source: RankingSourceType;
    usedCount?: number;
    lastUsedAt?: Date | null;
    sourceOrder?: number;
}

type RankableItem = {
    triggerId: string;
    title: string;
    rankingMeta?: RankingMeta;
}

const RECENT_HISTORY_WINDOW = 6;

export const normalizeRankingValue = (value: string) => {
    return value.trim().toLowerCase().replace(/\s+/g, " ");
};

export const createRankingInputContext = (
    input: Pick<AppInput, "value" | "valueType" | "clipboardContentType">
): RankingInputContext => {
    const normalizedValue = normalizeRankingValue(input.value || "");
    const valueType = input.valueType || "text";
    const clipboardContentType = input.clipboardContentType || "";

    return {
        key: `${valueType}:${clipboardContentType}:${normalizedValue}`,
        normalizedValue,
        valueType,
        clipboardContentType: input.clipboardContentType,
    };
};

const isInputCompatible = (current: RankingInputContext, record: RankingSelectionRecord) => {
    if (current.valueType !== record.valueType) {
        return false;
    }

    if ((current.clipboardContentType || "") !== (record.clipboardContentType || "")) {
        return false;
    }

    if (current.key === record.inputKey) {
        return true;
    }

    if (!current.normalizedValue || !record.normalizedValue) {
        return false;
    }

    if (current.normalizedValue === record.normalizedValue) {
        return true;
    }

    return current.normalizedValue.startsWith(record.normalizedValue)
        || record.normalizedValue.startsWith(current.normalizedValue)
        || (current.normalizedValue.length >= 2 && record.normalizedValue.includes(current.normalizedValue))
        || (record.normalizedValue.length >= 2 && current.normalizedValue.includes(record.normalizedValue));
};

const getCompatibleConsecutiveStreak = (
    triggerId: string,
    context: RankingInputContext,
    history: RankingSelectionRecord[]
) => {
    let streak = 0;

    for (const record of history) {
        if (record.triggerId !== triggerId) {
            break;
        }

        if (!isInputCompatible(context, record)) {
            break;
        }

        streak += 1;
    }

    return streak;
};

const getCompatibleRecentHits = (
    triggerId: string,
    context: RankingInputContext,
    history: RankingSelectionRecord[]
) => {
    return history
        .slice(0, RECENT_HISTORY_WINDOW)
        .filter(record => record.triggerId === triggerId && isInputCompatible(context, record))
        .length;
};

const getLastCompatibleSelectionAt = (
    triggerId: string,
    context: RankingInputContext,
    history: RankingSelectionRecord[]
) => {
    const match = history.find(record => record.triggerId === triggerId && isInputCompatible(context, record));
    if (!match) {
        return 0;
    }

    const selectedAt = new Date(match.selectedAt).getTime();
    return Number.isFinite(selectedAt) ? selectedAt : 0;
};

const getRecentSelectionBonus = (selectedAt: number) => {
    if (!selectedAt) {
        return 0;
    }

    const ageMs = Date.now() - selectedAt;

    if (ageMs <= 5 * 60 * 1000) {
        return 15000;
    }
    if (ageMs <= 30 * 60 * 1000) {
        return 10000;
    }
    if (ageMs <= 6 * 60 * 60 * 1000) {
        return 5000;
    }
    if (ageMs <= 24 * 60 * 60 * 1000) {
        return 1000;
    }

    return 0;
};

const getLastUsedBonus = (lastUsedAt?: Date | null) => {
    if (!lastUsedAt) {
        return 0;
    }

    const timestamp = lastUsedAt.getTime();
    if (!Number.isFinite(timestamp) || timestamp <= 0) {
        return 0;
    }

    const ageMs = Date.now() - timestamp;
    if (ageMs <= 24 * 60 * 60 * 1000) {
        return 2500;
    }
    if (ageMs <= 7 * 24 * 60 * 60 * 1000) {
        return 800;
    }

    return 0;
};

const getRankingScore = (
    triggerId: string,
    rankingMeta: RankingMeta | undefined,
    context: RankingInputContext,
    history: RankingSelectionRecord[]
) => {
    const compatibleStreak = getCompatibleConsecutiveStreak(triggerId, context, history);
    const recentHits = getCompatibleRecentHits(triggerId, context, history);
    const lastSelectedAt = getLastCompatibleSelectionAt(triggerId, context, history);
    const sourceOrder = rankingMeta?.sourceOrder ?? 99;
    const usedCount = rankingMeta?.usedCount ?? 0;

    let score = 0;

    if (compatibleStreak >= 2) {
        score += 1_000_000 + compatibleStreak * 100_000;
    } else if (compatibleStreak === 1) {
        score += 80_000;
    }

    score += recentHits * 15_000;
    score += getRecentSelectionBonus(lastSelectedAt);
    score += usedCount * 120;
    score += getLastUsedBonus(rankingMeta?.lastUsedAt);
    score += Math.max(0, 500 - sourceOrder * 10);

    return score;
};

export const compareRankableItems = <T extends RankableItem>(
    a: T,
    b: T,
    context: RankingInputContext,
    history: RankingSelectionRecord[]
) => {
    const scoreA = getRankingScore(a.triggerId, a.rankingMeta, context, history);
    const scoreB = getRankingScore(b.triggerId, b.rankingMeta, context, history);

    if (scoreA !== scoreB) {
        return scoreB - scoreA;
    }

    const usedCountA = a.rankingMeta?.usedCount ?? 0;
    const usedCountB = b.rankingMeta?.usedCount ?? 0;
    if (usedCountA !== usedCountB) {
        return usedCountB - usedCountA;
    }

    const lastUsedAtA = a.rankingMeta?.lastUsedAt?.getTime() ?? 0;
    const lastUsedAtB = b.rankingMeta?.lastUsedAt?.getTime() ?? 0;
    if (lastUsedAtA !== lastUsedAtB) {
        return lastUsedAtB - lastUsedAtA;
    }

    const sourceOrderA = a.rankingMeta?.sourceOrder ?? Number.MAX_SAFE_INTEGER;
    const sourceOrderB = b.rankingMeta?.sourceOrder ?? Number.MAX_SAFE_INTEGER;
    if (sourceOrderA !== sourceOrderB) {
        return sourceOrderA - sourceOrderB;
    }

    return a.title.localeCompare(b.title);
};
