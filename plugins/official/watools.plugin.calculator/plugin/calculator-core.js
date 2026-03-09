const roundResult = (value) => Math.round(value * 1000000000) / 1000000000;

export function calculateExpression(expression) {
    const cleanExpr = expression.replace(/[^0-9+\-*/.() ]/g, "").trim();
    if (!cleanExpr) {
        return {
            type: "error",
            title: "Invalid expression",
            subtitle: "Expression is empty after sanitization"
        };
    }

    if (cleanExpr.includes("++") || cleanExpr.includes("--")) {
        return {
            type: "error",
            title: "Invalid expression",
            subtitle: "Unsupported operator sequence"
        };
    }

    try {
        const result = new Function(`return (${cleanExpr})`)();
        if (typeof result !== "number" || !Number.isFinite(result)) {
            throw new Error("Invalid calculation result");
        }

        return {
            type: "result",
            title: `${expression} = ${roundResult(result)}`,
            subtitle: "Result copied to clipboard",
            value: roundResult(result)
        };
    } catch (error) {
        return {
            type: "error",
            title: "Calculation error",
            subtitle: error instanceof Error ? error.message : "Unknown error"
        };
    }
}
