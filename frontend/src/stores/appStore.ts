import {create} from "zustand";
import {AppInputValueType} from "@/schemas/app";

interface AppStore {
    displayValue: string
    value: string
    valueType: AppInputValueType
    lastCopiedValue: string | null
    setValue: (value: string, type: AppInputValueType) => void
    getValue: () => string
    clearValue: () => void

}

const createDebounce = (fn: (...args: any[]) => void, delay: number) => {
    let timer: ReturnType<typeof setTimeout> | null = null
    return (...args: any[]) => {
        if (timer) {
            clearTimeout(timer)
        }
        timer = setTimeout(() => {
            fn(...args)
        }, delay)
    }
}

export const useAppStore = create<AppStore>((set, get) => {
    const debouncedSetValue = createDebounce((value: string) => {
        set({value})
    }, 100)

    return {
        displayValue: '',
        value: '',
        valueType: 'text',
        lastCopiedValue: null,
        setValue: (value: string, valueType: AppInputValueType) => {
            const copiedValue = valueType === "clipboard" ? value : null
            set({displayValue: value, valueType, lastCopiedValue: copiedValue})
            if (valueType === "text") {
                debouncedSetValue(value)
            } else {
                set({value})
            }
            console.log('setValue called with', {value, valueType})
        },
        getValue: () => get().value,
        clearValue: () => set({value: '', valueType: 'text'}),
    }
})