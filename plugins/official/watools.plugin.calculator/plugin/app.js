import {appendHistory} from "./calculator-storage.js";
import {calculateExpression, canEvaluateExpression, extractExpression} from "./calculator-core.js";

const entry = [
    {
        type: "executable",
        subTitle: "Calculate Expression",
        icon: "calculator",
        match: (context) => {
            const expression = extractExpression(context.input.value);
            if (!expression) {
                return false;
            }
            return canEvaluateExpression(expression);
        },
        execute: async (context) => {
            const expression = extractExpression(context.input.value);
            const result = calculateExpression(expression);

            if (result.type !== "result") {
                return;
            }

            await window.runtime.ClipboardSetText(result.displayValue);
            await appendHistory(result.expression, result.displayValue);
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
