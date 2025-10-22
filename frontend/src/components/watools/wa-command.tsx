import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandList} from "@/components/ui/command";
import {useCallback, useEffect, useRef, useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";
import {cn} from "@/lib/utils";
import {CommandType} from "@/schemas/command";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {useDebounce} from "@uidotdev/usehooks";
import {WaOperationCommandGroup} from "@/components/watools/wa-operation-command-group";
import {PluginCommandEntry, WaPluginCommandGroup} from "@/components/watools/wa-plugin-command-group";
import {ClipboardGetText} from "../../../wailsjs/runtime";
import {HideAppApi, HideOrShowAppApi, TriggerCommandApi,} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {usePluginStore} from "@/stores";
import {Logger} from "@/lib/logger";
import {useLocation} from "wouter";


export const WaCommand = () => {
    const [input, setInput] = useState<string>('')
    const inputRef = useRef<HTMLInputElement>(null)
    const lastClipboardText = useRef<string>('')
    const [selectedKey, setSelectedKey] = useState<string>('')
    const commandListRef = useRef<HTMLDivElement>(null)
    const debounceInput = useDebounce(input, 50)
    const firstSelectedKeyRef = useRef<string>('')
    const {fetchPlugins} = usePluginStore()
    const [_, navigate] = useLocation()

    // Reset selected key when search input changes
    useEffect(() => {
        if (debounceInput) {
            setSelectedKey('')
            firstSelectedKeyRef.current = ''
        }
    }, [debounceInput])
    useEffect(() => {
        void fetchPlugins()
    }, []);

    useWindowFocus((focused) => {
        if (!focused) {
            return
        }
        ClipboardGetText().then(text => {
            text = text.trim()
            if (text.length > 1500) {
                text = text.substring(0, 1500) + '...'
            }
            if (text && text !== lastClipboardText.current && !debounceInput) {
                setInput(text)
                lastClipboardText.current = text
                setTimeout(() => {
                    if (inputRef.current) {
                        inputRef.current.select()
                    }
                }, 50)
            }
        })
        if (inputRef.current) {
            inputRef.current.focus()
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
            void HideOrShowAppApi()
        }
    }


    const handleHotkey = useCallback((e: KeyboardEvent) => {
        if (e.key === "Escape") {
            e.preventDefault()
            e.stopPropagation()
            onClickEscape()
        } else if (e.key === "Tab") {
            e.preventDefault()
            e.stopPropagation()
            if (inputRef.current) {
                inputRef.current.focus()
            }
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
        TriggerCommandApi(command.triggerId, command.category).then(() => {
            void HideAppApi()
        })
    }

    const onTriggerPluginCommand = async (entry: PluginCommandEntry, input: string) => {
        clearInput()
        if (entry.type === 'ui') {
            navigate(`/plugin?packageId=${entry.packageId}&file=${encodeURIComponent(entry.file || '')}`)
        } else if (entry.type === 'executable') {
            try {
                entry.execute && await entry.execute(input)
                void HideAppApi()
            } catch (error) {
                Logger.error(`Failed to execute plugin command: ${error}`)
            }
        }

    }
    const scrollToTop = () => {
        if (commandListRef.current) {
            commandListRef.current.scrollTo({top: 0})
        }
    }
    return <Command
        value={selectedKey}
        shouldFilter={false}
        loop
        className="rounded-lg border shadow-md w-full p-2"
        onValueChange={value => {
            console.log('onValueChange', value)
        }}
    >
        <WaComplexInput
            ref={inputRef}
            autoFocus
            onValueChange={setInput}
            className="text-gray-800 text-xl"
            classNames={{wrapper: isPanelOpen ? undefined : "!border-none"}}
            value={input}
        />
        <CommandList
            ref={commandListRef}
            className={cn("scrollbar-hide", isPanelOpen ? undefined : "hidden")}
        >
            <WaPluginCommandGroup
                searchKey={debounceInput}
                onTriggerPluginCommand={onTriggerPluginCommand}
                onSearchSuccess={(currentSelectedKey) => {
                    scrollToTop()
                    if (currentSelectedKey && !firstSelectedKeyRef.current) {
                        firstSelectedKeyRef.current = currentSelectedKey
                        setSelectedKey(currentSelectedKey)
                    }
                }}
            />
            <WaApplicationCommandGroup
                searchKey={debounceInput}
                onTriggerCommand={onTriggerCommand}
                onSearchSuccess={(currentSelectedKey) => {
                    scrollToTop()
                    if (currentSelectedKey && !firstSelectedKeyRef.current) {
                        firstSelectedKeyRef.current = currentSelectedKey
                        setSelectedKey(currentSelectedKey)
                    }
                }}
            />
            <WaOperationCommandGroup
                searchKey={debounceInput}
                onTriggerCommand={onTriggerCommand}
                onSearchSuccess={(currentSelectedKey) => {
                    scrollToTop()
                    if (currentSelectedKey && !firstSelectedKeyRef.current) {
                        firstSelectedKeyRef.current = currentSelectedKey
                        setSelectedKey(currentSelectedKey)
                    }
                }}
            />
        </CommandList>
    </Command>
}