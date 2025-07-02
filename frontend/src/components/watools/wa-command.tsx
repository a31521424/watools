import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandEmpty, CommandList} from "@/components/ui/command";
import {useCallback, useEffect, useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";
import {cn} from "@/lib/utils";
import {HideOrShowApp} from "../../../wailsjs/go/app/WaApp";


export const WaCommand = () => {
    const [input, setInput] = useState<string>('')
    const isPanelOpen = input.length > 0

    const clearInput = () => {
        setInput('')
    }

    const onClickEscape = () => {
        console.log('onClickEscape', isPanelOpen)
        if (isPanelOpen) {
            clearInput()
        } else {
            HideOrShowApp().then(_ => _)
        }
    }

    const handleHotkey = useCallback((e: KeyboardEvent) => {
        console.log('onHotkey', e.key)
        if (e.key === "Escape") {
            e.preventDefault()
            e.stopPropagation()
            onClickEscape()
        }
    }, [input])

    useEffect(() => {
        window.addEventListener("keydown", handleHotkey)
        return () => {
            window.removeEventListener("keydown", handleHotkey)
        }
    }, [handleHotkey])


    return <Command
        shouldFilter={false}
        className="rounded-lg border shadow-md w-full p-2"
    >
        <WaComplexInput
            onValueChange={setInput}
            classNames={{wrapper: isPanelOpen ? undefined : "!border-none"}}
            value={input}
        />
        <CommandList className={cn("scrollbar-hide", isPanelOpen ? undefined : "hidden")}>
            <CommandEmpty>No results found.</CommandEmpty>
            <WaApplicationCommandGroup searchKey={input}/>
        </CommandList>
    </Command>
}