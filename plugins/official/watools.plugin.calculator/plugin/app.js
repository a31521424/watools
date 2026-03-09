const isMathExpression = (value) => {
    if (!value || !/\d/.test(value)) {
        return false;
    }

    const cleanExpr = value.replace(/\s+/g, "");
    return /^[0-9+\-*/.()]+$/.test(cleanExpr);
};

const extractExpression = (value) => {
    const trimmed = value.trim();
    if (/^(calc|calculator|jsq|计算|计算器)\s+/i.test(trimmed)) {
        return trimmed.replace(/^(calc|calculator|jsq|计算|计算器)\s+/i, "");
    }
    return trimmed;
};

const entry = [
    {
        type: "executable",
        subTitle: "Calculate Expression",
        icon: "calculator",
        match: (context) => {
            const expression = extractExpression(context.input.value);
            if (!isMathExpression(expression)) {
                return false;
            }

            try {
                const cleanExpr = expression.replace(/[^0-9+\-*/.() ]/g, "");
                const result = new Function(`return (${cleanExpr})`)();
                return typeof result === "number" && Number.isFinite(result);
            } catch {
                return false;
            }
        },
        execute: async (context) => {
            const {calculateExpression} = await import("./calculator-core.js");
            const expression = extractExpression(context.input.value);
            const result = calculateExpression(expression);

            if (result.type !== "result") {
                return;
            }

            await window.runtime.ClipboardSetText(String(result.value));

            const storageKey = "history";
            const history = await window.watools.StorageGet(storageKey) || [];
            const nextHistory = [{
                expression,
                result: result.value,
                createdAt: new Date().toISOString()
            }, ...history].slice(0, 50);

            await window.watools.StorageSet(storageKey, nextHistory);
        }
    },
    {
        type: "ui",
        subTitle: "Open Calculator Panel",
        icon: "calculator",
        match: (context) => {
            const trimmed = context.input.value.trim().toLowerCase();
            return ["calc", "calculator", "jsq", "计算", "计算器"].includes(trimmed);
        },
        file: "index.html"
    }
];

export default entry;
