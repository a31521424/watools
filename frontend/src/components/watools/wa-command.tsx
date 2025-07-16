import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandEmpty, CommandList} from "@/components/ui/command";
import {useCallback, useEffect, useRef, useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";
import {cn} from "@/lib/utils";
import {HideApp, HideOrShowApp} from "../../../wailsjs/go/app/WaApp";
import {CommandType} from "@/schemas/command";
import {RunApplication} from "../../../wailsjs/go/launch/WaLaunchApp";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {isDevMode} from "@/lib/env";


export const WaCommand = () => {
    const [input, setInput] = useState<string>('')
    const commandListRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        if (commandListRef) {
            const commandList = commandListRef.current
            if (commandList) {
                commandList.scrollTo({
                    top: 0
                })
            }
        }
    }, [input]);


    useWindowFocus((focus) => {
        console.log('window onFocusChange', focus)
        if (isDevMode()) {
            return
        }
        if (!focus) {
            HideApp()
        }
    })

    const isPanelOpen = input.length > 0
    const clearInput = () => {
        setInput('')
    }

    const onClickEscape = () => {
        console.log('onClickEscape', isPanelOpen)
        if (isPanelOpen) {
            clearInput()
        } else {
            HideOrShowApp()
        }
    }

    const handleHotkey = useCallback((e: KeyboardEvent) => {
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

    const onTriggerCommand = (command: CommandType) => {
        clearInput()
        HideApp()
        HideOrShowApp()
        RunApplication(command.path)
        HideOrShowApp()
    }
    return <Command
        shouldFilter={false}
        className="rounded-lg border shadow-md w-full p-2"
    >
        <WaComplexInput
            onValueChange={setInput}
            classNames={{wrapper: isPanelOpen ? undefined : "!border-none"}}
            value={input}
        />
        <CommandList
            ref={commandListRef}
            className={cn("scrollbar-hide", isPanelOpen ? undefined : "hidden")}
        >
            <CommandEmpty>No results found.</CommandEmpty>
            <WaApplicationCommandGroup searchKey={input} onTriggerCommand={onTriggerCommand}/>
        </CommandList>
    </Command>
}