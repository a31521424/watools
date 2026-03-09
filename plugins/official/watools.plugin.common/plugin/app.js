const isUrlLike = (value) => {
    return value.startsWith("http://") || value.startsWith("https://") || value.startsWith("www.");
};

const isPathLike = (value) => {
    return value.startsWith("~/") || value.startsWith("/");
};

const entry = [
    {
        type: "executable",
        subTitle: "Open URL or Path",
        icon: "external-link",
        match: (context) => {
            const trimmed = context.input.value.trim();
            return isUrlLike(trimmed) || isPathLike(trimmed);
        },
        execute: async (context) => {
            const trimmed = context.input.value.trim();
            if (!trimmed) {
                return;
            }

            if (isUrlLike(trimmed)) {
                const url = trimmed.startsWith("www.") ? `https://${trimmed}` : trimmed;
                await window.runtime.BrowserOpenURL(url);
                return;
            }

            if (isPathLike(trimmed)) {
                await window.watools.OpenFolder(trimmed);
            }
        }
    },
    {
        type: "executable",
        subTitle: "Copy File Path",
        icon: "clipboard-copy",
        match: (context) => context.input.clipboardContentType === "files",
        execute: async (context) => {
            const files = context.clipboard?.files || [];
            if (!files.length) {
                return;
            }
            await window.runtime.ClipboardSetText(files.join("\n"));
        }
    },
    {
        type: "executable",
        subTitle: "Save Clipboard Image",
        icon: "image-down",
        match: (context) => context.input.clipboardContentType === "image",
        execute: async (context) => {
            const imageBase64 = context.clipboard?.imageBase64;
            if (!imageBase64) {
                return;
            }

            const savePath = await window.watools.SaveBase64Image(imageBase64);
            if (savePath) {
                await window.watools.OpenFolder(savePath);
            }
        }
    }
];

export default entry;
