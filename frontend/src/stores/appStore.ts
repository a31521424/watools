import {create} from "zustand";
import {AppClipboardContent, AppClipboardContentType, AppInputValueType} from "@/schemas/app";

type AppState = {
    displayValue: string
    compressedDisplay: boolean
    value: string
    valueType: AppInputValueType
    lastCopiedValue: string | null
    imageBase64: string | null
    files: string[] | null
    clipboardContentType: AppClipboardContentType | null
}
const resetState = {
    displayValue: '',
    compressedDisplay: false,
    value: '',
    valueType: 'text' as AppInputValueType,
    imageBase64: null,
    files: null,
    clipboardContentType: null,
}
const initialState: AppState = {
    ...resetState,
    lastCopiedValue: null,
}

type AppStore = AppState & {
    setTextValue: (value: string) => void
    setClipboardValue: (value: string, isAuto?: boolean) => void
    setValue: (value: string, type: AppInputValueType, onSuccess?: () => void, isAuto?: boolean) => void
    setValueAuto: (value: string, type: AppInputValueType, onSuccess?: () => void) => void
    getValue: () => string
    clearValue: () => void
    setClipboardContent: (content: AppClipboardContent | null) => void
    isPanelOpen: () => boolean
    canClearAssets: () => boolean
    getClipboardContent: () => AppClipboardContent | null
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
        setTextValue: (value: string) => {
            set({displayValue: value, valueType: "text"})
            debouncedSetValue(value)
        },
        setClipboardValue: (value: string, isAuto?: boolean) => {
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
                set({
                    displayValue: displayValue,
                    value,
                    valueType: "clipboard",
                    lastCopiedValue: value,
                    compressedDisplay
                })
            }
        },
        setValue: (value: string, valueType: AppInputValueType, onSuccess?: () => void, isAuto?: boolean) => {
            if (valueType === "text") {
                get().setTextValue(value.trimStart())
            } else if (valueType === "clipboard") {
                get().setClipboardValue(value.trim(), isAuto)
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
        setClipboardContent: (content: AppClipboardContent | null) => {
            if (!content) {
                return
            }
            if (content.contentType === "image" && content.imageBase64) {
                set({
                    imageBase64: content.imageBase64,
                    files: null,
                    clipboardContentType: content.contentType,
                    value: '',
                    displayValue: '',
                    valueType: 'clipboard',
                })
            } else if (content.contentType === "files" && content.files) {
                set({
                    files: content.files,
                    imageBase64: content.imageBase64,
                    clipboardContentType: content.contentType,
                    value: '',
                    displayValue: '',
                    valueType: 'clipboard',
                })
            }
        },
        getValue: () => get().value,
        clearValue: () => {
            set(resetState)
        },
        isPanelOpen: () => {
            const state = get()
            return state.value.length > 0 || state.imageBase64 != null || state.files != null
        },
        canClearAssets: () => {
            const state = get()
            return state.value.length === 0 && (state.imageBase64 != null || state.files != null)
        },
        getClipboardContent: () => {
            const state = get()
            if (!state.clipboardContentType) {
                return null
            }
            return {
                contentType: state.clipboardContentType,
                text: state.value,
                imageBase64: state.imageBase64,
                files: state.files,
            }
        },
    }
})