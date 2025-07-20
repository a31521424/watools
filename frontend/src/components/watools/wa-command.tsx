import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandEmpty, CommandList} from "@/components/ui/command";
import {useCallback, useEffect, useRef, useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";
import {cn} from "@/lib/utils";
import {HideApp, HideOrShowApp} from "../../../wailsjs/go/app/WaApp";
import {CommandType} from "@/schemas/command";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {isDevMode} from "@/lib/env";
import {useDebounce} from "@uidotdev/usehooks";
import {TriggerCommand} from "../../../wailsjs/go/command/WaLaunchApp";


export const WaCommand = () => {
    const [input, setInput] = useState<string>('')
    const commandListRef = useRef<HTMLDivElement>(null)
    const debounceInput = useDebounce(input, 50)

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
        TriggerCommand(command.triggerId)
        HideOrShowApp()
    }
    const scrollToTop = () => {
        if (commandListRef.current) {
            commandListRef.current.scrollTo({top: 0})
        }
    }
    return <Command
        shouldFilter={false}
        loop
        className="rounded-lg border shadow-md w-full p-2"

    >
        <WaComplexInput
            autoFocus
            onValueChange={setInput}
            classNames={{wrapper: isPanelOpen ? undefined : "!border-none"}}
            value={input}
        />
        <CommandList
            ref={commandListRef}
            className={cn("scrollbar-hide", isPanelOpen ? undefined : "hidden")}
        >
            <CommandEmpty>No results found.</CommandEmpty>
            <WaApplicationCommandGroup
                searchKey={debounceInput}
                onTriggerCommand={onTriggerCommand}
                onSearchSuccess={scrollToTop}
            />
        </CommandList>
    </Command>
}