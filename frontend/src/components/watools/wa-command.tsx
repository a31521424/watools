import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandList} from "@/components/ui/command";
import React, {useCallback, useEffect, useMemo, useRef, useState} from "react";
import {cn} from "@/lib/utils";
import {CommandType} from "@/schemas/command";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {PluginCommandEntry, usePluginItems} from "@/components/watools/wa-plugin-item";
import {ClipboardGetText} from "../../../wailsjs/runtime";
import {HideAppApi, HideOrShowAppApi, TriggerCommandApi,} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {useAppStore, usePluginStore} from "@/stores";
import {Logger} from "@/lib/logger";
import {useLocation} from "wouter";
import {isDevMode} from "@/lib/env";
import {AppInput} from "@/schemas/app";
import {useApplicationItems} from "@/components/watools/wa-application-item";
import {useOperationItems} from "@/components/watools/wa-operation-item";
import {BaseItemProps, WaBaseItem} from "@/components/watools/wa-base-item";


export const WaCommand = () => {
    const inputRef = useRef<HTMLInputElement>(null)
    const commandListRef = useRef<HTMLDivElement>(null)
    const {fetchPlugins, updatePluginUsage} = usePluginStore()
    const [_, navigate] = useLocation()
    const {
        value: inputValue,
        displayValue: inputDisplayValue,
        valueType: inputValueType,
        setValue: setInputValue,
        setValueAuto: setInputValueAuto,
        clearValue: clearInputValue,
    } = useAppStore()
    const isPanelOpen = inputValue.length > 0

    const pluginInput: AppInput = useMemo(() => ({
        value: inputValue,
        valueType: inputValueType,
    }), [inputValue, inputValueType])

    const onTriggerCommand = useCallback((command: CommandType) => {
        clearInputValue()
        TriggerCommandApi(command.triggerId, command.category).then(() => {
            void HideAppApi()
        })
    }, [clearInputValue])

    const onTriggerPluginCommand = useCallback(async (entry: PluginCommandEntry, input: AppInput) => {
        clearInputValue()

        // Update plugin usage statistics
        try {
            await updatePluginUsage(entry.packageId)
        } catch (error) {
            Logger.error(`Failed to update plugin usage: ${error}`)
        }

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
    }, [clearInputValue, updatePluginUsage, navigate])

    // Get items from hooks directly
    const applicationItems = useApplicationItems({
        searchKey: inputValue,
        onTriggerCommand
    });

    const operationItems = useOperationItems({
        searchKey: inputValue,
        onTriggerCommand
    });

    const pluginItems = usePluginItems({
        input: pluginInput,
        onTriggerPluginCommand
    });

    // Combine and sort all items by usedCount only
    const combinedItems = useMemo((): BaseItemProps[] => {
        const allItems = [
            ...pluginItems,
            ...applicationItems,
            ...operationItems,
        ];

        // Sort by usedCount (higher is better)
        return allItems.sort((a, b) => {
            const usedCountA = a.usedCount || 0;
            const usedCountB = b.usedCount || 0;
            return usedCountB - usedCountA;
        });
    }, [applicationItems, operationItems, pluginItems]);

    useEffect(() => {
        void fetchPlugins()
    }, []);

    useWindowFocus((focused) => {
        if (!focused) {
            return
        }
        ClipboardGetText().then(text => {
            setInputValueAuto(text, "clipboard", () => setTimeout(() => {
                if (!inputRef.current) {
                    return
                }
                inputRef.current.select()
            }, 100))
        })
        inputRef.current?.focus()
    })

    useWindowFocus((focused) => {
        if (!focused) {
            if (isDevMode()) {
                return
            }
            void HideAppApi()
        }
    })


    const handlePaste = (e: React.ClipboardEvent) => {
        e.preventDefault()
        try {
            let text = e.clipboardData.getData('text').trim()
            setInputValue(text, "clipboard", () => setTimeout(() => {
                if (!inputRef.current) {
                    return
                }
                inputRef.current.scrollLeft = inputRef.current.scrollWidth
            }, 0))
        } catch (e) {
            Logger.error(`Handle paste error: ${e}`)
        }
    }

    const onClickEscape = useCallback(() => {
        if (isPanelOpen) {
            clearInputValue()
        } else {
            void HideOrShowAppApi()
        }
    }, [isPanelOpen])

    useEffect(() => {
        const handleHotkey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                e.preventDefault()
                e.stopPropagation()
                onClickEscape()
            } else if (e.key === "Tab") {
                e.preventDefault()
                e.stopPropagation()
                inputRef.current?.focus()
            }
        }
        window.addEventListener("keydown", handleHotkey)
        return () => {
            window.removeEventListener("keydown", handleHotkey)
        }
    }, [onClickEscape])

    const selectedKey = useMemo(() => {
        return combinedItems.length > 0 ? combinedItems[0].triggerId : ""
    }, [combinedItems])

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
            {combinedItems.map(item => (
                <WaBaseItem key={item.triggerId} {...item} />
            ))}
        </CommandList>
    </Command>
}