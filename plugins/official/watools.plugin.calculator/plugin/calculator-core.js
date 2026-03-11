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
            title: "表达式无效",
            subtitle: "请输入要计算的数学表达式"
        };
    }

    if (!ALLOWED_EXPRESSION_PATTERN.test(normalized)) {
        return {
            type: "error",
            title: "表达式无效",
            subtitle: "仅支持数字、空格、括号以及 + - * / ."
        };
    }

    try {
        const result = new Function(`"use strict"; return (${normalized})`)();
        if (typeof result !== "number" || !Number.isFinite(result)) {
            throw new Error("计算结果无效");
        }

        const rounded = roundResult(result);
        const displayValue = String(rounded);

        return {
            type: "result",
            title: `${normalized} = ${displayValue}`,
            subtitle: "计算成功",
            expression: normalized,
            value: rounded,
            displayValue
        };
    } catch (error) {
        return {
            type: "error",
            title: "计算错误",
            subtitle: error instanceof Error ? error.message : "未知错误"
        };
    }
}
