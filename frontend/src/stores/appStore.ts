import {create} from "zustand";
import {AppClipboardContent, AppClipboardContentType, AppInputValueType} from "@/schemas/app";

type AppState = {
    inputRevision: number
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
    inputRevision: 0,
    lastCopiedValue: null,
}

type AppStore = AppState & {
    setTextValue: (value: string) => void
    setClipboardValue: (value: string, isAuto?: boolean) => void
    setValue: (value: string, type: AppInputValueType, onSuccess?: () => void, isAuto?: boolean) => void
    setValueAuto: (value: string, type: AppInputValueType, onSuccess?: () => void, expectedRevision?: number) => void
    getValue: () => string
    clearValue: () => void
    setClipboardContent: (content: AppClipboardContent | null, expectedRevision?: number) => void
    isPanelOpen: () => boolean
    canClearAssets: () => boolean
    getClipboardContent: () => AppClipboardContent | null
}

export const useAppStore = create<AppStore>((set, get) => {
    return {
        ...initialState,
        setTextValue: (value: string) => {
            set(state => ({...resetState, inputRevision: state.inputRevision + 1, displayValue: value, value, valueType: "text"}))
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
                set(state => ({
                    ...resetState,
                    inputRevision: state.inputRevision + 1,
                    displayValue: displayValue,
                    value,
                    valueType: "clipboard",
                    clipboardContentType: 'text',
                    lastCopiedValue: value,
                    compressedDisplay
                }))
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
        setValueAuto: (value: string, valueType: AppInputValueType, onSuccess?: () => void, expectedRevision?: number) => {
            if (expectedRevision !== undefined && expectedRevision !== get().inputRevision) return
            if (get().value) {
                return
            }
            if (value.length < 800000 && value == get().lastCopiedValue) {
                return
            }
            get().setValue(value, valueType, onSuccess, true)
        },
        setClipboardContent: (content: AppClipboardContent | null, expectedRevision?: number) => {
            if (expectedRevision !== undefined && expectedRevision !== get().inputRevision) return
            if (!content) {
                return
            }
            if (content.contentType === "text") {
                if (content.text) get().setClipboardValue(content.text)
                else get().clearValue()
            } else if (content.contentType === "image" && content.imageBase64) {
                set(state => ({
                    ...resetState,
                    inputRevision: state.inputRevision + 1,
                    imageBase64: content.imageBase64,
                    files: null,
                    clipboardContentType: content.contentType,
                    value: '',
                    displayValue: '',
                    valueType: 'clipboard',
                }))
            } else if (content.contentType === "files" && content.files) {
                set(state => ({
                    ...resetState,
                    inputRevision: state.inputRevision + 1,
                    files: content.files,
                    imageBase64: null,
                    clipboardContentType: content.contentType,
                    value: '',
                    displayValue: '',
                    valueType: 'clipboard',
                }))
            }
        },
        getValue: () => get().value,
        clearValue: () => {
            set(state => ({...resetState, inputRevision: state.inputRevision + 1}))
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
