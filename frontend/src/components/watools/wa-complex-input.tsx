import {Search} from "lucide-react";
import {cn} from "@/lib/utils";
import {Command as CommandPrimitive} from "cmdk";
import * as React from "react";
import {useMemo} from "react";

export const WaComplexInput = (
    {className, classNames, imageBase64, files, textPreview, onCopyOrigin, ...props}: React.ComponentProps<typeof CommandPrimitive.Input> & {
        classNames?: { wrapper?: string },
        imageBase64?: string | null,
        files: string[] | null
        textPreview?: string | null,
        onCopyOrigin?: () => void,
    }
) => {
    const previewAssets = useMemo(() => {
        if (files != null) {
            return <div className="flex h-full w-auto max-w-[65%] gap-x-1 overflow-x-hidden shrink-0">
                {files.map(item => {
                    return <div
                        key={`file-preview-${item}`}
                        className="flex rounded-lg border border-dashed h-full items-center overflow-hidden p-1"
                    >
                        {imageBase64 && <img
                            draggable={false}
                            className="aspect-square max-h-full select-none rounded p-0 m-0"
                            src={`data:image/png;base64,${imageBase64}`}
                            alt="Input Image"
                        />}
                        <div className="text-sm whitespace-nowrap scrollbar-hide">
                            {item.split("/").pop()}
                        </div>
                    </div>
                })}
            </div>
        } else if (imageBase64 != null) {
            return <div className="h-full w-auto shrink-0 p-1 rounded-lg border border-dashed object-contain">
                <img
                    draggable={false}
                    className="max-h-full select-none"
                    src={`data:image/png;base64,${imageBase64}`}
                    alt="Input Image"
                />
            </div>
        }
        return null
    }, [imageBase64, files])
    return (
        <div
            data-slot="command-input-wrapper"
            className={cn("flex h-9 gap-x-1 w-full items-center", classNames?.wrapper)}
        >
            <Search size={19} className="shrink-0 text-muted-foreground" aria-hidden="true"/>
            {previewAssets}
            {textPreview && <div className="flex max-w-[65%] min-w-0 shrink-0 items-center gap-2 rounded border border-dashed px-2 py-1 text-sm" aria-label="剪贴板文本摘要">
                <span className="truncate select-none">{textPreview}</span>
                <button type="button" className="shrink-0 text-xs text-primary" onMouseDown={event => event.preventDefault()} onClick={onCopyOrigin}>复制原文</button>
            </div>}
            <CommandPrimitive.Input
                data-slot="command-input"
                className={cn(
                    "bg-transparent border-transparent shadow-none",
                    "focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:border-transparent",
                    "h-auto flex-1",
                    className
                )}
                placeholder="搜索应用、插件或输入指令…"
                {...props}
            />
        </div>
    )

}
