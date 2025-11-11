import {cn} from "@/lib/utils";
import {Command as CommandPrimitive} from "cmdk";
import * as React from "react";
import {useEffect, useMemo} from "react";

export const WaComplexInput = (
    {className, classNames, imageBase64, files, ...props}: React.ComponentProps<typeof CommandPrimitive.Input> & {
        classNames?: { wrapper?: string },
        imageBase64?: string | null,
        files: string[] | null
    }
) => {
    useEffect(() => {
        console.log('WaComplexInput imageBase64 changed:', imageBase64);
    }, [imageBase64]);
    const previewAssets = useMemo(() => {
        if (files != null) {
            return <div className="flex h-full w-auto max-w-[65%] gap-x-1 overflow-x-hidden shrink-0">
                {files.map(item => {
                    return <div
                        key={`file-preview-${item}`}
                        className="flex border-2 border-dotted h-full items-center overflow-hidden p-1"
                    >
                        {imageBase64 && <img
                            draggable={false}
                            className="aspect-square max-h-full select-none border-dotted rounded p-1"
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
            return <div className="h-full w-auto shrink-0 p-1 border-2 border-dotted object-contain">
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
            className={cn("flex h-9 gap-x-1 w-full items-center px-2", classNames?.wrapper)}
        >
            {previewAssets}
            <CommandPrimitive.Input
                data-slot="command-input"
                className={cn(
                    "bg-transparent border-transparent shadow-none",
                    "focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:border-transparent",
                    "h-auto flex-1",
                    className
                )}
                placeholder="Hello Watools"
                {...props}
            />
        </div>
    )

}
