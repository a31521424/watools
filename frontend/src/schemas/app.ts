export type AppInputValueType = "text" | "clipboard"
export type AppInput = {
    valueType: AppInputValueType
    value: string
}

export type ClipboardContentType = "text" | "image" | "files"

export type ClipboardContent = {
    contentType: ClipboardContentType
    text: string | null
    imageBase64: string | null
    files: string[] | null
}