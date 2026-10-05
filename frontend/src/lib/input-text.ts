// Limits apply before trim/normalisation/search, so even whitespace-only
// megabyte inputs do not turn into a full scan on the launcher thread.
export const MAX_SEARCH_QUERY_LENGTH = 128;
export const MAX_INLINE_TEXT_LENGTH = 2000;
export const MAX_MATCH_TEXT_EDGE_LENGTH = 128;

// Fixed work at the edges; do not split an original Unicode surrogate pair.
export const getTextEdges = (value: string): {head: string; tail: string} => {
    let end = Math.min(value.length, MAX_MATCH_TEXT_EDGE_LENGTH);
    let start = Math.max(0, value.length - MAX_MATCH_TEXT_EDGE_LENGTH);
    const high = (unit: number) => unit >= 0xd800 && unit <= 0xdbff;
    const low = (unit: number) => unit >= 0xdc00 && unit <= 0xdfff;
    if (high(value.charCodeAt(end - 1)) && low(value.charCodeAt(end))) end--;
    if (low(value.charCodeAt(start)) && high(value.charCodeAt(start - 1))) start++;
    return {head: value.slice(0, end), tail: value.slice(start)};
};

export const getSearchQuery = (value: string): string =>
    value.length <= MAX_SEARCH_QUERY_LENGTH ? value.trim() : '';

export const getTextDisplay = (value: string): {displayValue: string; compressedDisplay: boolean} => {
    if (value.length <= MAX_INLINE_TEXT_LENGTH) return {displayValue: value, compressedDisplay: false};
    // Array.from is deliberately limited to the edges, avoiding a full-text
    // allocation and keeping surrogate pairs intact.
    const head = Array.from(value.slice(0, 62)).slice(0, 30).join('');
    const tail = Array.from(value.slice(-62)).slice(-30).join('');
    return {displayValue: `${head}  ......  ${tail}`, compressedDisplay: true};
};
