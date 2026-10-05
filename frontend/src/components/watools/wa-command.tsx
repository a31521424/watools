import {WaComplexInput} from "@/components/watools/wa-complex-input";
import {Command, CommandList} from "@/components/ui/command";
import React, {useCallback, useEffect, useMemo, useRef} from "react";
import {cn} from "@/lib/utils";
import {CommandType} from "@/schemas/command";
import {useWindowFocus} from "@/hooks/useWindowFocus";
import {PluginCommandEntry, usePluginItems} from "@/components/watools/wa-plugin-item";
import {host} from "@/api/host"
import {useAppStore, usePluginStore} from "@/stores";
import {Logger} from "@/lib/logger";
import {useLocation} from "wouter";
import {isDevMode} from "@/lib/env";
import {useApplicationItems} from "@/components/watools/wa-application-item";
import {useOperationItems} from "@/components/watools/wa-operation-item";
import {useAppFeatureItems} from "@/components/watools/wa-app-feature-item";
import {BaseItemProps, WaBaseItem} from "@/components/watools/wa-base-item";
import {getClipboardContent, getAutomaticClipboard, resolvePluginContext} from "@/api/app";
import {withPluginApi} from "@/lib/plugin-execution";
import {useApplicationCommandStore} from "@/stores/applicationCommandStore";
import {compareRankableItems, createRankingInputContext} from "@/lib/command-ranking";
import {useCommandRankingStore} from "@/stores";
import {persistPluginLaunchContext} from "@/lib/plugin-context";
import {getLauncherPresentation} from '@/lib/launcher-input';
import {ClipboardSetText} from '@/lib/legacy-runtime';


export const WaCommand = () => {
    const inputRef = useRef<HTMLInputElement>(null)
    const commandListRef = useRef<HTMLDivElement>(null)
    const [executionError, setExecutionError] = React.useState<string | null>(null)
    const [launching, setLaunching] = React.useState(false)
    const launchPending = useRef(false)
    const focusGeneration = useRef(0)
    const updateApplicationUsage = useApplicationCommandStore(state => state.updateCommandUsage)
    const {updatePluginUsage} = usePluginStore()
    const flushPluginUsage = usePluginStore(state => state.flushBufferUpdates)
    const flushApplicationUsage = useApplicationCommandStore(state => state.flushBufferUpdates)
    const rankingHistory = useCommandRankingStore(state => state.history)
    const recordSelection = useCommandRankingStore(state => state.recordSelection)
    const [_, navigate] = useLocation()

    const input = useAppStore(state => state.input)
    const {displayValue, compressedDisplay, query: searchQuery, rankingInput, imageBase64, files} = useMemo(() => getLauncherPresentation(input), [input])
    const isPanelOpen = useAppStore(state => state.isPanelOpen())

    // Subscribe to methods
    const setTextValue = useAppStore(state => state.setTextValue)
    const setAutomaticClipboard = useAppStore(state => state.setAutomaticClipboard)
    const clearValue = useAppStore(state => state.clearValue)
    const setClipboardContent = useAppStore(state => state.setClipboardContent)
    const getCanClearAssets = useAppStore(state => state.canClearAssets)
    const getIsPanelOpen = useAppStore(state => state.isPanelOpen)


    const rankingContext = useMemo(() => createRankingInputContext(rankingInput), [rankingInput])

    useEffect(() => () => {focusGeneration.current++}, [])

    const onTriggerCommand = useCallback((command: CommandType) => {
        setExecutionError(null)
        const revision = useAppStore.getState().inputRevision
        const generation = focusGeneration.current
        host.TriggerCommandApi(command.triggerId, command.category).then(async () => {
            if (command.category === "Application" && "id" in command && typeof command.id === "string") await updateApplicationUsage(command.id)
            if (generation !== focusGeneration.current || !clearValue(revision)) return
            const clearedRevision = useAppStore.getState().inputRevision
            await Promise.allSettled([flushApplicationUsage(), flushPluginUsage()])
            if (clearedRevision === useAppStore.getState().inputRevision && generation === focusGeneration.current) void host.HideAppApi()
        }).catch(error => {
            Logger.error('命令执行失败')
            if (revision === useAppStore.getState().inputRevision && generation === focusGeneration.current) setExecutionError(`命令执行失败：${String(error)}`)
        })
    }, [clearValue, flushApplicationUsage, flushPluginUsage, updateApplicationUsage])

    const onTriggerPluginCommand = useCallback(async (entry: PluginCommandEntry) => {
        const state = useAppStore.getState()
        if (launchPending.current || state.input !== input) return
        launchPending.current = true
        setLaunching(true)
        setExecutionError(null)
        const revision = state.inputRevision
        const generation = focusGeneration.current
        try {
            const launchContext = await resolvePluginContext(input)
            if (revision !== useAppStore.getState().inputRevision || generation !== focusGeneration.current) return
            if (!usePluginStore.getState().getPluginById(entry.packageId)?.enabled) throw new Error('插件已停用')
            // Updating usage must not hold the route behind a SQLite write.
            void updatePluginUsage(entry.packageId).catch(() => Logger.error('插件使用统计保存失败'))
            if (entry.type === 'ui') {
                const params = new URLSearchParams({packageId: entry.packageId, file: entry.file || ''})
                const launchId = persistPluginLaunchContext(launchContext)
                params.set('launchId', launchId)
                navigate(`/plugin?${params.toString()}`)
            } else if (entry.type === 'executable') {
                await withPluginApi(entry.packageId, api => {
                    // The compatibility queue may have waited for another
                    // plugin; recheck cancellation when this action can run.
                    if (revision !== useAppStore.getState().inputRevision || generation !== focusGeneration.current) return;
                    if (!usePluginStore.getState().getPluginById(entry.packageId)?.enabled) throw new Error('插件已停用');
                    return entry.execute(launchContext, api);
                })
                if (generation !== focusGeneration.current || !clearValue(revision)) return
                const clearedRevision = useAppStore.getState().inputRevision
                await Promise.allSettled([flushApplicationUsage(), flushPluginUsage()])
                if (clearedRevision === useAppStore.getState().inputRevision && generation === focusGeneration.current) void host.HideAppApi()
            }
        } catch (error) {
            Logger.error('插件启动或执行失败')
            if (revision === useAppStore.getState().inputRevision && generation === focusGeneration.current) setExecutionError(`插件启动或执行失败：${String(error)}`)
        } finally {
            launchPending.current = false
            setLaunching(false)
        }
    }, [input, updatePluginUsage, navigate, clearValue, flushApplicationUsage, flushPluginUsage])

    // Get items from hooks directly
    const applicationItems = useApplicationItems({
        searchKey: searchQuery,
        rankingContext,
        rankingHistory,
        onTriggerCommand
    });

    const operationItems = useOperationItems({
        searchKey: searchQuery,
        rankingContext,
        rankingHistory,
        onTriggerCommand
    });

    const appFeatureItems = useAppFeatureItems({
        searchKey: searchQuery,
        rankingContext,
        rankingHistory,
        onTriggerAppFeature: clearValue
    });


    const pluginItems = usePluginItems({
        input,
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


    const handleWindowFocus = useCallback((focused: boolean) => {
        const generation = ++focusGeneration.current
        if (focused) {
            const state = useAppStore.getState()
            if (!state.isPanelOpen()) {
                const revision = state.inputRevision
                getAutomaticClipboard().then(res => {
                    if (!res) return
                    const accepted = generation === focusGeneration.current && setAutomaticClipboard(res, revision)
                    if (accepted) inputRef.current?.select()
                    // A rejected handler does not own this shared read. Its
                    // bounded backend cache expires unless another focus accepts
                    // it; releasing here could invalidate that newer consumer.
                }).catch(() => Logger.error('自动读取剪贴板失败'))
            }
            inputRef.current?.focus()
        } else {
            // for dev mode, do not hide app when window loses focus
            if (isDevMode()) {
                return
            }
            const revision = useAppStore.getState().inputRevision
            void Promise.allSettled([flushApplicationUsage(), flushPluginUsage()]).finally(() => {
                if (revision === useAppStore.getState().inputRevision && generation === focusGeneration.current) void host.HideAppApi()
            })
        }
    }, [setAutomaticClipboard, flushApplicationUsage, flushPluginUsage])
    useWindowFocus(handleWindowFocus, [handleWindowFocus])

    useEffect(() => {
        const handleHotkey = (e: KeyboardEvent) => {
            if (e.isComposing || e.keyCode === 229) return
            if (e.key === "Escape") {
                e.preventDefault()
                e.stopPropagation()
                if (getIsPanelOpen()) {
                    clearValue()
                } else {
                    const revision = useAppStore.getState().inputRevision
                    const generation = focusGeneration.current
                    void Promise.allSettled([flushApplicationUsage(), flushPluginUsage()]).finally(() => {
                        if (revision === useAppStore.getState().inputRevision && generation === focusGeneration.current) void host.HideOrShowAppApi()
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
        if (text && !event.clipboardData.types.includes('Files')) {
            setClipboardContent({contentType: 'text', text, imageBase64: null, files: null}, revision)
            inputRef.current?.select()
            return
        }
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

    const copyOrigin = useCallback(async () => {
        try {
            const context = await resolvePluginContext(input)
            await ClipboardSetText(context.input.value)
        } catch (error) {
            if (useAppStore.getState().input === input) setExecutionError(`复制原文失败：${String(error)}`)
        }
    }, [input])

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
                setTextValue(value)
            }}
            onPaste={handlePaste}
            className="text-gray-800"
            textPreview={compressedDisplay ? displayValue : null}
            onCopyOrigin={copyOrigin}
            classNames={{wrapper: cn("text-xl", isPanelOpen ? undefined : "!border-none")}}
            value={compressedDisplay ? '' : displayValue}
        />
        {launching && <div role="status" className="px-3 py-1 text-sm text-gray-500">正在打开插件…</div>}
        {compressedDisplay && <div className="px-3 py-1 text-xs text-gray-500">选择插件处理完整文本；输入新查询会替换当前文本。</div>}
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
