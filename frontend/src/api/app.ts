import {GetClipboardContentApi} from "../../wailsjs/go/coordinator/WaAppCoordinator";
import {AppClipboardContent} from "@/schemas/app";

export const getClipboardContent = async (): Promise<AppClipboardContent | null> => {
    const content = await GetClipboardContentApi()
    if (content.contentType === "empty") {
        return null
    }
    const data: AppClipboardContent = {
        contentType: content.contentType as "text" | "image" | "files",
        text: null,
        imageBase64: null,
        files: null,
    }
    if (content.hasFiles) {
        data.files = content.files as string[]
    }
    if (content.hasImage) {
        data.imageBase64 = content.imageBase64 as string
    }
    if (content.hasText) {
        data.text = content.text as string
    }
    return data
}