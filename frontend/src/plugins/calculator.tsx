import {PluginEntry, PluginMetadata, PluginPackage} from "src/schemas/plugin";
import {ClipboardSetText} from "../../wailsjs/runtime";
import React, {useState} from "react";
import {createRoot} from "react-dom/client";
import { tw } from 'twind';


const Calculator: React.FC<{ input: string }> = ({input}) => {
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
        <div className={tw`w-full p-3 font-sans`}>
            {/* 输入框 */}
            <input
                type="text"
                value={expression}
                onChange={(e) => setExpression(e.target.value)}
                placeholder="请输入算式，如 1+2*3"
                className={tw`w-full p-2 mb-2.5 border border-gray-300 rounded-md font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent`}
            />

            {/* 结果 */}
            {result && (
                <div className={tw`text-lg font-bold text-blue-500 mb-2`}>
                    结果: {result}
                </div>
            )}

            {/* 错误提示 */}
            {error && (
                <div className={tw`bg-red-50 text-red-600 px-2 py-1.5 rounded-md text-sm mb-2`}>
                    {error}
                </div>
            )}

            {/* 操作按钮 */}
            <div className={tw`flex gap-2`}>
                <button
                    onClick={() => {
                        ClipboardSetText(result)
                        setCopied(true)
                        setTimeout(() => setCopied(false), 1500)
                    }}
                    disabled={!result}
                    className={tw`flex-1 p-2 border-none rounded-md text-white text-sm transition-colors ${
                        result
                            ? 'bg-blue-600 hover:bg-blue-700 cursor-pointer'
                            : 'bg-gray-400 cursor-not-allowed'
                    }`}
                >
                    {copied ? "✅ 已复制" : "复制结果"}
                </button>
                <button
                    onClick={() => setExpression("")}
                    className={tw`flex-1 p-2 border border-gray-300 rounded-md bg-gray-50 hover:bg-gray-100 cursor-pointer text-sm transition-colors`}
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
        root.render(<Calculator input={input.trim()}/>)
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

const Plugin: PluginPackage = {
    metadata,
    allEntries: [calculatorEntry]
}

// @ts-ignore
if (!window.WailsAppPlugins) {
    // @ts-ignore
    window.WailsAppPlugins = {}
}
// @ts-ignore
window.WailsAppPlugins[metadata.packageID] = Plugin

export default Plugin
