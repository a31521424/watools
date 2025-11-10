import {create} from "zustand";
import {AppInputValueType} from "@/schemas/app";

type AppState = {
    displayValue: string
    compressedDisplay: boolean
    value: string
    valueType: AppInputValueType
    lastCopiedValue: string | null
}

const initialState: AppState = {
    displayValue: '',
    compressedDisplay: false,
    value: '',
    valueType: 'text',
    lastCopiedValue: null,
}

type AppStore = AppState & {
    setValue: (value: string, type: AppInputValueType, onSuccess?: () => void, isAuto?: boolean) => void
    setValueAuto: (value: string, type: AppInputValueType, onSuccess?: () => void) => void
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
    }, 50)

    return {
        ...initialState,
        setValue: (value: string, valueType: AppInputValueType, onSuccess?: () => void, isAuto?: boolean) => {
            value = value.trim()
            if (valueType === "text") {
                set({displayValue: value, valueType: valueType})
                debouncedSetValue(value)
            } else if (valueType === "clipboard") {
                if (!value.length) {
                    return
                }

                let displayValue = value
                let compressedDisplay = false
                if (isAuto && displayValue.length > 2000) {
                    displayValue = displayValue.slice(0, 30) + '  ......  ' + displayValue.slice(-30)
                    compressedDisplay = true
                }
                if (value) {
                    set({displayValue: displayValue, value, valueType, lastCopiedValue: value, compressedDisplay})
                }
            }
            if (onSuccess) {
                onSuccess()
            }

        },
        // Set value by clipboard auto only when there is no similar value
        setValueAuto: (value: string, valueType: AppInputValueType, onSuccess?: () => void) => {
            if (get().value) {
                return
            }
            if (value.length < 800000 && value == get().lastCopiedValue) {
                return
            }
            get().setValue(value, valueType, onSuccess, true)
        },
        getValue: () => get().value,
        clearValue: () => {
            const {lastCopiedValue, ...reset} = initialState
            set(reset)
        },
    }
})