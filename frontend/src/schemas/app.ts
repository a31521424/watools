export type AppInputValueType = "text" | "clipboard"
export type AppInput = {
    valueType: AppInputValueType
    value: string
    clipboardContentType?: AppClipboardContentType
}

export type AppClipboardContentType = "text" | "image" | "files"

export type AppClipboardContent = {
    contentType: AppClipboardContentType
    text: string | null
    imageBase64: string | null
    files: string[] | null
}