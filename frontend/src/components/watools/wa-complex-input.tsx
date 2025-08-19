import {cn} from "@/lib/utils";
import {Command as CommandPrimitive} from "cmdk";
import * as React from "react";

export const WaComplexInput = (
    {className, classNames, ...props}: React.ComponentProps<typeof CommandPrimitive.Input> & {
        classNames?: { wrapper?: string }
    }
) => {
    return (
        <div
            data-slot="command-input-wrapper"
            className={cn("flex h-9 items-center gap-2 p-3", classNames?.wrapper)}
        >
            <CommandPrimitive.Input
                data-slot="command-input"
                className={cn(
                    "bg-transparent border-transparent shadow-none",
                    "focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:border-transparent",
                    "md:text-1xl text-2xl",
                    "h-auto w-full",
                    className
                )}
                placeholder="Hello Watools"
                {...props}
            />
        </div>
    )

}
