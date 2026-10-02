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
import {useAppFeatureItems} from "@/components/watools/wa-app-feature-item";
import {BaseItemProps, WaBaseItem} from "@/components/watools/wa-base-item";
import {getClipboardContent} from "@/api/app";
import {PluginContext} from "@/schemas/plugin";
import {useShallow} from "zustand/react/shallow";
import {withPluginApi} from "@/lib/plugin-execution";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";
import {compareRankableItems, createRankingInputContext} from "@/lib/command-ranking";
import {useCommandRankingStore} from "@/stores";
import {persistPluginLaunchContext} from "@/lib/plugin-context";


export const WaCommand = () => {
    const inputRef = useRef<HTMLInputElement>(null)
    const commandListRef = useRef<HTMLDivElement>(null)
    const [executionError, setExecutionError] = React.useState<string | null>(null)
    const updateApplicationUsage = useApplicationCommandStore(state => state.updateCommandUsage)
    const {updatePluginUsage} = usePluginStore()
    const flushPluginUsage = usePluginStore(state => state.flushBufferUpdates)
    const flushApplicationUsage = useApplicationCommandStore(state => state.flushBufferUpdates)
    const rankingHistory = useCommandRankingStore(state => state.history)
    const recordSelection = useCommandRankingStore(state => state.recordSelection)
    const [_, navigate] = useLocation()

    // Subscribe to individual store values using selectors
    const value = useAppStore(state => state.value)
    const displayValue = useAppStore(state => state.displayValue)
    const valueType = useAppStore(state => state.valueType)
    const imageBase64 = useAppStore(state => state.imageBase64)
    const files = useAppStore(state => state.files)
    const clipboardContentType = useAppStore(state => state.clipboardContentType)
    const clipboard = useAppStore(useShallow(state => state.getClipboardContent()))
    const isPanelOpen = useAppStore(state => state.isPanelOpen())

    // Subscribe to methods
    const setValue = useAppStore(state => state.setValue)
    const setValueAuto = useAppStore(state => state.setValueAuto)
    const clearValue = useAppStore(state => state.clearValue)
    const setClipboardContent = useAppStore(state => state.setClipboardContent)
    const getCanClearAssets = useAppStore(state => state.canClearAssets)
    const getIsPanelOpen = useAppStore(state => state.isPanelOpen)


    const pluginInput: AppInput = useMemo(() => ({
        value,
        valueType,
        clipboardContentType: clipboardContentType ?? undefined,
    }), [value, valueType, clipboardContentType])
    const rankingContext = useMemo(() => createRankingInputContext(pluginInput), [pluginInput])

    const onTriggerCommand = useCallback((command: CommandType) => {
        setExecutionError(null)
        TriggerCommandApi(command.triggerId, command.category).then(async () => {
            if (command.category === "Application" && "id" in command && typeof command.id === "string") await updateApplicationUsage(command.id)
            clearValue()
            void Promise.allSettled([flushApplicationUsage(), flushPluginUsage()]).finally(() => {
                void HideAppApi()
            })
        }).catch(error => { Logger.error(`命令执行失败: ${error}`); setExecutionError(`命令执行失败：${String(error)}`) })
    }, [clearValue, flushApplicationUsage, flushPluginUsage, updateApplicationUsage])

    const onTriggerPluginCommand = useCallback(async (entry: PluginCommandEntry, context: PluginContext) => {
        // Update plugin usage statistics
        try {
            await updatePluginUsage(entry.packageId)
        } catch (error) {
            Logger.error(`Failed to update plugin usage: ${error}`)
        }

        if (entry.type === 'ui') {
            const params = new URLSearchParams({
                packageId: entry.packageId,
                file: entry.file || '',
            })
            const launchId = persistPluginLaunchContext(context)
            if (launchId) {
                params.set('launchId', launchId)
            }
            navigate(`/plugin?${params.toString()}`)
        } else if (entry.type === 'executable') {
            try {
                await withPluginApi(entry.packageId, api => {
                    if (!usePluginStore.getState().getPluginById(entry.packageId)?.enabled) throw new Error('插件已停用');
                    return entry.execute?.(context, api);
                })
                clearValue()
                await Promise.allSettled([flushApplicationUsage(), flushPluginUsage()])
                void HideAppApi()
            } catch (error) {
                Logger.error(`Failed to execute plugin command: ${error}`)
                setExecutionError(`插件执行失败：${String(error)}`)
            }
        }
    }, [updatePluginUsage, navigate, clearValue, flushApplicationUsage, flushPluginUsage])

    // Get items from hooks directly
    const applicationItems = useApplicationItems({
        searchKey: value,
        rankingContext,
        rankingHistory,
        onTriggerCommand
    });

    const operationItems = useOperationItems({
        searchKey: value,
        rankingContext,
        rankingHistory,
        onTriggerCommand
    });

    const appFeatureItems = useAppFeatureItems({
        searchKey: value,
        rankingContext,
        rankingHistory,
        onTriggerAppFeature: clearValue
    });


    const pluginItems = usePluginItems({
        input: pluginInput,
        clipboard,
        rankingContext,
        rankingHistory,
        onTriggerPluginCommand,
    });

    const combinedItems = useMemo((): BaseItemProps[] => {
        const allItems = [
            ...pluginItems,
            ...applicationItems,
            ...operationItems,
            ...appFeatureItems,
        ];

        const uniqueItems = new Map<string, BaseItemProps>();
        for (const item of allItems) {
            const existing = uniqueItems.get(item.triggerId);
            if (!existing || compareRankableItems(item, existing, rankingContext, rankingHistory) < 0) {
                uniqueItems.set(item.triggerId, item);
            }
        }

        return Array.from(uniqueItems.values())
            .sort((a, b) => compareRankableItems(a, b, rankingContext, rankingHistory))
            .map(item => {
                const originalOnSelect = item.onSelect;
                return {
                    ...item,
                    onSelect: () => {
                        if (item.rankingMeta) {
                            recordSelection({
                                triggerId: item.triggerId,
                                source: item.rankingMeta.source,
                                input: rankingContext,
                            });
                        }
                        originalOnSelect();
                    }
                };
            });
    }, [applicationItems, operationItems, pluginItems, appFeatureItems, rankingContext, rankingHistory, recordSelection]);

    const selectedKey = useMemo(() => {
        return combinedItems.length > 0 ? combinedItems[0].triggerId : undefined
    }, [combinedItems])


    useWindowFocus((focused) => {
        if (focused) {
            const revision = useAppStore.getState().inputRevision
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
                    }, 0), revision)
                }
            })
            inputRef.current?.focus()
        } else {
            // for dev mode, do not hide app when window loses focus
            if (isDevMode()) {
                return
            }
            void Promise.allSettled([flushApplicationUsage(), flushPluginUsage()]).finally(() => {
                void HideAppApi()
            })
        }
    })

    useEffect(() => {
        const handleHotkey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                e.preventDefault()
                e.stopPropagation()
                if (getIsPanelOpen()) {
                    clearValue()
                } else {
                    void Promise.allSettled([flushApplicationUsage(), flushPluginUsage()]).finally(() => {
                        void HideOrShowAppApi()
                    })
                }
            } else if (e.key === "Tab") {
                e.preventDefault()
                e.stopPropagation()
                inputRef.current?.focus()
            } else if (e.key === "Backspace") {
                if (getCanClearAssets()) {
                    clearValue()
                }
            }
        }
        window.addEventListener("keydown", handleHotkey)
        return () => {
            window.removeEventListener("keydown", handleHotkey)
        }
    }, [clearValue, flushApplicationUsage, flushPluginUsage])

    const handlePaste = useCallback((event: React.ClipboardEvent<HTMLInputElement>) => {
        event.preventDefault()
        const revision = useAppStore.getState().inputRevision
        const text = event.clipboardData.getData('text/plain')
        getClipboardContent().then(content => {
            const result = content?.contentType === 'text' && text
                ? {...content, text}
                : content || {contentType: 'text' as const, text, imageBase64: null, files: null}
            setClipboardContent(result, revision)
        }).catch(error => {
            Logger.error(`读取剪贴板失败: ${error}`)
            if (text) setClipboardContent({contentType: 'text', text, imageBase64: null, files: null}, revision)
        })
    }, [setClipboardContent])

    return <Command
        value={selectedKey}
        shouldFilter={false}
        className="rounded-lg border shadow-md w-full p-2"
        disablePointerSelection={true}
    >
        <WaComplexInput
            ref={inputRef}
            autoFocus
            imageBase64={imageBase64}
            files={files}
            onValueChange={value => {
                setExecutionError(null)
                setValue(value, "text")
            }}
            onPaste={handlePaste}
            className="text-gray-800"
            classNames={{wrapper: cn("text-xl", isPanelOpen ? undefined : "!border-none")}}
            value={displayValue}
        />
        {executionError && <div role="alert" className="px-3 py-2 text-sm text-red-600">{executionError}</div>}
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
