import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandList} from "@/components/ui/command";
import React, {useCallback, useEffect, useMemo, useRef} from "react";
import {cn} from "@/lib/utils";
import {CommandType} from "@/schemas/command";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {PluginCommandEntry, usePluginItems} from "@/components/watools/wa-plugin-item";
import {HideAppApi, HideOrShowAppApi, TriggerCommandApi,} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {useAppStore, usePluginStore} from "@/stores";
import {Logger} from "@/lib/logger";
import {useLocation} from "wouter";
import {isDevMode} from "@/lib/env";
import {AppInput} from "@/schemas/app";
import {useApplicationItems} from "@/components/watools/wa-application-item";
import {useOperationItems} from "@/components/watools/wa-operation-item";
import {BaseItemProps, WaBaseItem} from "@/components/watools/wa-base-item";
import {getClipboardContent} from "@/api/app";
import {PluginContext} from "@/schemas/plugin";


export const WaCommand = () => {
    const inputRef = useRef<HTMLInputElement>(null)
    const commandListRef = useRef<HTMLDivElement>(null)
    const [isPasted, setIsPasted] = React.useState<boolean>(false)
    const [selectedKey, setSelectedKey] = React.useState<string>("")
    const {updatePluginUsage} = usePluginStore()
    const [_, navigate] = useLocation()

    // Subscribe to individual store values using selectors
    const value = useAppStore(state => state.value)
    const displayValue = useAppStore(state => state.displayValue)
    const valueType = useAppStore(state => state.valueType)
    const imageBase64 = useAppStore(state => state.imageBase64)
    const files = useAppStore(state => state.files)
    const clipboardContentType = useAppStore(state => state.clipboardContentType)

    // Subscribe to methods
    const setValue = useAppStore(state => state.setValue)
    const setValueAuto = useAppStore(state => state.setValueAuto)
    const clearValue = useAppStore(state => state.clearValue)
    const setClipboardContent = useAppStore(state => state.setClipboardContent)

    // Compute derived state from store values
    const isPanelOpen = useAppStore(state =>
        state.value.length > 0 || state.imageBase64 != null || state.files != null
    )

    const pluginInput: AppInput = useMemo(() => ({
        value,
        valueType,
        clipboardContentType: clipboardContentType ?? undefined,
    }), [value, valueType, clipboardContentType])

    const onTriggerCommand = useCallback((command: CommandType) => {
        clearValue()
        TriggerCommandApi(command.triggerId, command.category).then(() => {
            void HideAppApi()
        })
    }, [clearValue])

    const onTriggerPluginCommand = useCallback(async (entry: PluginCommandEntry, context: PluginContext) => {
        clearValue()

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
                entry.execute && await entry.execute(context)
                void HideAppApi()
            } catch (error) {
                Logger.error(`Failed to execute plugin command: ${error}`)
            }
        }
    }, [clearValue, updatePluginUsage, navigate])

    // Get items from hooks directly
    const applicationItems = useApplicationItems({
        searchKey: value,
        onTriggerCommand
    });

    const operationItems = useOperationItems({
        searchKey: value,
        onTriggerCommand
    });

    // Get clipboard content snapshot once
    const clipboard = useMemo(() => {
        return useAppStore.getState().getClipboardContent()
    }, [imageBase64, files, clipboardContentType])

    const pluginItems = usePluginItems({
        input: pluginInput,
        clipboard,
        onTriggerPluginCommand,
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


    useWindowFocus((focused) => {
        if (!focused) {
            return
        }
        getClipboardContent().then(res => {
            if (!res) {
                return
            }
            if (res.contentType === "text" && res.text) {
                setValueAuto(res.text, "clipboard", () => setTimeout(() => {
                    if (!inputRef.current) {
                        return
                    }
                    inputRef.current.select()
                }, 0))
            }
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


    useEffect(() => {
        const handleHotkey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                e.preventDefault()
                e.stopPropagation()
                if (useAppStore.getState().isPanelOpen()) {
                    clearValue()
                } else {
                    void HideOrShowAppApi()
                }
            } else if (e.key === "Tab") {
                e.preventDefault()
                e.stopPropagation()
                inputRef.current?.focus()
            } else if (e.key === "Backspace") {
                if (useAppStore.getState().canClearAssets()) {
                    clearValue()
                }
            }
        }
        window.addEventListener("keydown", handleHotkey)
        return () => {
            window.removeEventListener("keydown", handleHotkey)
        }
    }, [clearValue])

    useEffect(() => {
        if (combinedItems.length > 0) {
            setSelectedKey(combinedItems[0].triggerId)
        } else {
            setSelectedKey("")
        }

    }, [combinedItems])

    const handlePaste = () => {
        setIsPasted(true)
        getClipboardContent().then(res => {
            console.log('Pasted clipboard content:', res);
            setClipboardContent(res)
        })
    }

    return <Command
        value={selectedKey}
        shouldFilter={false}
        className="rounded-lg border shadow-md w-full p-2"
    >
        <WaComplexInput
            ref={inputRef}
            autoFocus
            imageBase64={imageBase64}
            files={files}
            onValueChange={value => {
                if (!isPasted) {
                    setValue(value, "text")
                } else {
                    setValue(value, "clipboard")
                    setIsPasted(false)
                }
            }}
            onPaste={handlePaste}
            className="text-gray-800"
            classNames={{wrapper: cn("text-xl", isPanelOpen ? undefined : "!border-none")}}
            value={displayValue}
        />
        <CommandList
            ref={commandListRef}
            className={cn(
                "scrollbar-hide",
                isPanelOpen ? undefined : "hidden",
                combinedItems.length ? "mt-2" : null
            )}
        >
            {combinedItems.map(item => (
                <WaBaseItem key={item.triggerId} {...item} />
            ))}
        </CommandList>
    </Command>
}