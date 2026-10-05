import {appendHistory} from "./calculator-storage.js";
import {calculateExpression, extractExpression, isMathExpression} from "./calculator-core.js";

const entry = [
    {
        type: "executable",
        subTitle: "计算表达式",
        icon: "calculator",
        // A candidate check only: never evaluate an expression during matching.
        matchPreview: (context) => isMathExpression(context.input.head) && (!context.input.truncated || isMathExpression(context.input.tail)),
        execute: async (context) => {
            const expression = extractExpression(context.input.value);
            const result = calculateExpression(expression);

            if (result.type !== "result") {
                throw new Error(`${result.title}：${result.subtitle}`);
            }

            await window.runtime.ClipboardSetText(result.displayValue);
            await appendHistory(result.expression, result.displayValue);
        }
    },
    {
        type: "ui",
        subTitle: "打开计算器面板",
        icon: "calculator",
        matchPreview: (context) => !context.input.truncated && ["calc", "calculator", "jsq", "计算", "计算器"].includes(context.input.head.trim().toLowerCase()),
        file: "index.html"
    }
];

export default entry;
