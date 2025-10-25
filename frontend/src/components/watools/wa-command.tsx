import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandList} from "@/components/ui/command";
import React, {useCallback, useEffect, useMemo, useRef, useState} from "react";
import {WaApplicationCommandGroup} from "@/components/watools/wa-application-command-group";
import {cn} from "@/lib/utils";
import {CommandType} from "@/schemas/command";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {WaOperationCommandGroup} from "@/components/watools/wa-operation-command-group";
import {PluginCommandEntry, WaPluginCommandGroup} from "@/components/watools/wa-plugin-command-group";
import {ClipboardGetText} from "../../../wailsjs/runtime";
import {HideAppApi, HideOrShowAppApi, TriggerCommandApi,} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {usePluginStore} from "@/stores";
import {Logger} from "@/lib/logger";
import {useLocation} from "wouter";
import {isDevMode} from "@/lib/env";

import {AppInput} from "@/schemas/app";
import {useAppStore} from "@/stores/appStore";


export const WaCommand = () => {
    const inputRef = useRef<HTMLInputElement>(null)
    const [selectedKey, setSelectedKey] = useState<string>('')
    const commandListRef = useRef<HTMLDivElement>(null)
    const firstSelectedKeyRef = useRef<string>('')
    const {fetchPlugins} = usePluginStore()
    const [_, navigate] = useLocation()
    const {
        value: inputValue,
        displayValue: inputDisplayValue,
        valueType: inputValueType,
        setValue: setInputValue,
        clearValue: clearInputValue,
        lastCopiedValue
    } = useAppStore()

    const pluginInput: AppInput = useMemo(() => ({
        value: inputValue,
        valueType: inputValueType,
    }), [inputValue])

    // Reset selected key when search input changes
    useEffect(() => {
        if (inputValue) {
            setSelectedKey('')
            firstSelectedKeyRef.current = ''
        }
    }, [inputValue])
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
            if (text && text !== lastCopiedValue && !inputValue) {
                setInputValue(text, "clipboard")
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

    useWindowFocus((focused) => {
        if (!focused) {
            if (isDevMode()) {
                return
            }
            void HideAppApi()
        }
    })


    const isPanelOpen = inputValue.length > 0

    const onClickEscape = () => {
        console.log('onClickEscape', isPanelOpen)
        if (isPanelOpen) {
            clearInputValue()
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
    }, [inputValue])

    const handlePaste = (e: React.ClipboardEvent) => {
        e.preventDefault()
        try {
            let text = e.clipboardData.getData('text').trim()
            if (text.length > 1500) {
                text = text.substring(0, 1500) + '...'
            }
            if (inputRef.current) {
                inputRef.current.value = text
                inputRef.current.focus()
                inputRef.current.setSelectionRange(text.length, text.length)
                inputRef.current.scrollLeft = inputRef.current.scrollWidth
            }
            setInputValue(text, "clipboard")
        } catch (e) {
            Logger.error(`Handle paste error: ${e}`)
        }
    }
    useEffect(() => {
        window.addEventListener("keydown", handleHotkey)
        return () => {
            window.removeEventListener("keydown", handleHotkey)
        }
    }, [handleHotkey])


    const onTriggerCommand = (command: CommandType) => {
        clearInputValue()
        TriggerCommandApi(command.triggerId, command.category).then(() => {
            void HideAppApi()
        })
    }

    const onTriggerPluginCommand = async (entry: PluginCommandEntry, input: AppInput) => {
        clearInputValue()
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
    >
        <WaComplexInput
            ref={inputRef}
            autoFocus
            onValueChange={value => setInputValue(value, "text")}
            onPaste={handlePaste}
            className="text-gray-800 text-xl"
            classNames={{wrapper: isPanelOpen ? undefined : "!border-none"}}
            value={inputDisplayValue}
        />
        <CommandList
            ref={commandListRef}
            className={cn("scrollbar-hide", isPanelOpen ? undefined : "hidden")}
        >
            <WaPluginCommandGroup
                input={pluginInput}
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
                searchKey={inputValue}
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
                searchKey={inputValue}
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