import {PluginEntry, PluginMetadata, PluginPackage} from "src/schemas/plugin";
import React, {useState} from "react";
import {createRoot} from "react-dom/client";
import {ClipboardSetText} from "../../wailsjs/runtime";


const WatoolsCalculator: React.FC<{ input: string }> = ({input}) => {
    const [expression, setExpression] = useState(input)
    const [result, setResult] = useState<string>("")
    const [error, setError] = useState<string>("")
    const [copied, setCopied] = useState(false)

    React.useEffect(() => {
        try {
            const safeExpr = expression.replace(/[^0-9+\-*/().\s]/g, "")
            if (safeExpr.trim()) {
                const calcResult = new Function("return " + safeExpr)()
                setResult(calcResult.toString())
                setError("")
            } else {
                setResult("")
                setError("请输入算式")
            }
        } catch {
            setError("无效的表达式")
            setResult("")
        }
    }, [expression])

    return (
        <div style={{
            width: "100%",
            maxWidth: "320px",
            padding: "12px",
            fontFamily: "sans-serif",
        }}>
            <input
                type="text"
                value={expression}
                onChange={(e) => setExpression(e.target.value)}
                placeholder="请输入算式，如 1+2*3"
                style={{
                    width: "100%",
                    padding: "8px",
                    marginBottom: "10px",
                    border: "1px solid #ccc",
                    borderRadius: "6px",
                    fontFamily: "monospace",
                    fontSize: "14px"
                }}
            />

            {result && (
                <div style={{
                    fontSize: "18px",
                    fontWeight: "bold",
                    color: "#2196F3",
                    marginBottom: "8px"
                }}>
                    结果: {result}
                </div>
            )}

            {error && (
                <div style={{
                    background: "#ffeaea",
                    color: "#b00020",
                    padding: "6px 8px",
                    borderRadius: "6px",
                    fontSize: "13px",
                    marginBottom: "8px"
                }}>
                    {error}
                </div>
            )}

            <div style={{display: "flex", gap: "8px"}}>
                <button
                    onClick={() => {
                        ClipboardSetText(result)
                        setCopied(true)
                        setTimeout(() => setCopied(false), 1500)
                    }}
                    disabled={!result}
                    style={{
                        flex: 1,
                        padding: "8px",
                        border: "none",
                        borderRadius: "6px",
                        background: result ? "#007bff" : "#ccc",
                        color: "#fff",
                        cursor: result ? "pointer" : "not-allowed",
                        fontSize: "14px"
                    }}
                >
                    {copied ? "✅ 已复制" : "复制结果"}
                </button>
                <button
                    onClick={() => setExpression("")}
                    style={{
                        flex: 1,
                        padding: "8px",
                        border: "1px solid #ccc",
                        borderRadius: "6px",
                        background: "#f5f5f5",
                        cursor: "pointer",
                        fontSize: "14px"
                    }}
                >
                    清空
                </button>
            </div>
        </div>
    )
}

const calculatorEntry: PluginEntry = {
    entryID: 'watools.calculator.simpleCalculator',
    title: 'Calculator',
    icon: 'calculator',

    match: (input: string): boolean => {
        const mathPattern = /^[\d+\-*/().\s]+$/
        return mathPattern.test(input.trim()) && input.trim().length > 0
    },

    render: (container: Element, input: string): void => {
        const root = createRoot(container)
        root.render(<WatoolsCalculator input={input.trim()}/>)
    },

}

const metadata: PluginMetadata = {
    id: 'e6c8cc94-27ba-42b7-9ad2-4543ab02635b',
    packageID: 'watools.calculator',
    name: 'calculator',
    version: '0.0.1',
    description: 'A simple calculator plugin',
    author: 'Banbxio'
}

const CalculatorPlugin: PluginPackage = {
    metadata,
    allEntries: [calculatorEntry]
}

export default CalculatorPlugin
