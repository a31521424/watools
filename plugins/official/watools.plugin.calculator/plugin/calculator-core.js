export const CALCULATOR_PREFIX_PATTERN = /^(calc|calculator|jsq|计算|计算器)\s+/i;

const ALLOWED_EXPRESSION_PATTERN = /^[0-9+\-*/.()\s]+$/;

const roundResult = (value) => Math.round(value * 1000000000) / 1000000000;

export const extractExpression = (value) => {
    const trimmed = typeof value === "string" ? value.trim() : "";
    if (!trimmed) {
        return "";
    }

    if (CALCULATOR_PREFIX_PATTERN.test(trimmed)) {
        return trimmed.replace(CALCULATOR_PREFIX_PATTERN, "").trim();
    }

    return trimmed;
};

export const normalizeExpression = (value) => {
    const extracted = extractExpression(value);
    if (!extracted) {
        return "";
    }

    return extracted
        .replace(/[xX×]/g, "*")
        .replace(/[÷／]/g, "/")
        .replace(/[,，]/g, "")
        .replace(/[（]/g, "(")
        .replace(/[）]/g, ")")
        .replace(/[＋]/g, "+")
        .replace(/[－–—]/g, "-")
        .replace(/\s+/g, " ")
        .trim();
};

export const isMathExpression = (value) => {
    const normalized = normalizeExpression(value);
    if (!normalized || !/\d/.test(normalized)) {
        return false;
    }

    return ALLOWED_EXPRESSION_PATTERN.test(normalized);
};

export const canEvaluateExpression = (value) => {
    return calculateExpression(value).type === "result";
};

export function calculateExpression(expression) {
    const normalized = normalizeExpression(expression);
    if (!normalized) {
        return {
            type: "error",
            title: "Invalid expression",
            subtitle: "Enter a math expression to calculate"
        };
    }

    if (!ALLOWED_EXPRESSION_PATTERN.test(normalized)) {
        return {
            type: "error",
            title: "Invalid expression",
            subtitle: "Only numbers, spaces, parentheses, and + - * / . are supported"
        };
    }

    try {
        const result = new Function(`"use strict"; return (${normalized})`)();
        if (typeof result !== "number" || !Number.isFinite(result)) {
            throw new Error("Invalid calculation result");
        }

        const rounded = roundResult(result);
        const displayValue = String(rounded);

        return {
            type: "result",
            title: `${normalized} = ${displayValue}`,
            subtitle: "Calculated successfully",
            expression: normalized,
            value: rounded,
            displayValue
        };
    } catch (error) {
        return {
            type: "error",
            title: "Calculation error",
            subtitle: error instanceof Error ? error.message : "Unknown error"
        };
    }
}
