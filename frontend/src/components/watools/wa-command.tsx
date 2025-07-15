import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandEmpty, CommandList} from "@/components/ui/command";
import {useCallback, useEffect, useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";
import {cn} from "@/lib/utils";
import {HideApp, HideOrShowApp} from "../../../wailsjs/go/app/WaApp";
import {CommandType} from "@/schemas/command";
import {RunApplication} from "../../../wailsjs/go/launch/WaLaunchApp";
import {useElementResize} from "@/hooks/useElementResize";


export const WaCommand = () => {
    const [input, setInput] = useState<string>('')
    const listContainerRef = useElementResize<HTMLDivElement>({
        onResize: _ => {
            if (listContainerRef.current) {
                listContainerRef.current.scrollTop = 0
            }
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
            HideOrShowApp().then(_ => _)
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
        HideApp().then(_ => _)
        RunApplication(command.path).then(res => {
            console.log(res)
        })
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
        <CommandList ref={listContainerRef} className={cn("", isPanelOpen ? undefined : "hidden")}>
            <CommandEmpty>No results found.</CommandEmpty>
            <WaApplicationCommandGroup searchKey={input} onTriggerCommand={onTriggerCommand}/>
        </CommandList>
    </Command>
}