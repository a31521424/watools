import React, {useEffect, useState} from 'react'
import {Plugin} from '@/schemas/plugin'
import {Button} from '@/components/ui/button'
import {Switch} from '@/components/ui/switch'
import {Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle} from '@/components/ui/sheet'
import {useLocation} from "wouter";
import {InstallPluginByFileDialogApi} from "../../../wailsjs/go/coordinator/WaAppCoordinator";
import {usePluginStore} from "@/stores/pluginStore";

export function WaPluginManagement() {
    const plugins = usePluginStore(state => state.plugins)
    const isLoading = usePluginStore(state => state.isLoading)
    const refreshPlugins = usePluginStore(state => state.refreshPlugins)
    const togglePlugin = usePluginStore(state => state.togglePlugin)
    const uninstallPlugin = usePluginStore(state => state.uninstallPlugin)

    const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null)
    const [isDrawerOpen, setIsDrawerOpen] = useState(false)
    const [_, navigate] = useLocation()

    useEffect(() => {
        const handleHotkey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                navigate("/")
            }
        }
        window.addEventListener('keydown', handleHotkey)
        return () => {
            window.removeEventListener('keydown', handleHotkey)
        }
    }, [])

    const handleTogglePlugin = async (plugin: Plugin) => {
        try {
            await togglePlugin(plugin.packageId, !plugin.enabled)
        } catch (error) {
            console.error('Failed to toggle plugin:', error)
        }
    }

    const handleUninstallPlugin = async (plugin: Plugin) => {
        try {
            await uninstallPlugin(plugin.packageId)
            setIsDrawerOpen(false)
            setSelectedPlugin(null)
        } catch (error) {
            console.error('Failed to uninstall plugin:', error)
        }
    }

    const handleInstallPlugin = async () => {
        void InstallPluginByFileDialogApi().then(() => {
            void refreshPlugins()
        })
    }

    const openPluginDetails = (plugin: Plugin) => {
        setSelectedPlugin(plugin)
        setIsDrawerOpen(true)
    }

    return (
        <div className="p-6 max-w-4xl mx-auto">
            <div className="flex justify-between items-center mb-6">
                <h1 className="text-2xl font-bold">Plugin Management</h1>
                <Button onClick={handleInstallPlugin}>
                    Install Plugin
                </Button>
            </div>

            {isLoading ? (
                <div className="text-center py-12">Loading plugins...</div>
            ) : plugins.length === 0 ? (
                <div className="text-center py-12 text-gray-500">
                    No plugins installed. Click "Install Plugin" to add one.
                </div>
            ) : (
                <div className="space-y-4">
                    {plugins.map(plugin => (
                        <div
                            key={plugin.packageId}
                            className="border rounded-lg p-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
                        >
                            <div className="flex-1 cursor-pointer" onClick={() => openPluginDetails(plugin)}>
                                <div className="flex items-center gap-3">
                                    <h3 className="text-lg font-semibold">{plugin.name}</h3>
                                </div>
                                <p className="text-sm text-gray-600 mt-1">{plugin.description}</p>
                                <div className="flex gap-4 mt-2 text-xs text-gray-500">
                                    <span>v{plugin.version}</span>
                                    <span>by {plugin.author}</span>
                                    <span>Used {plugin.usedCount} times</span>
                                </div>
                            </div>
                            <div className="flex items-center gap-3">
                                <Switch
                                    checked={plugin.enabled}
                                    onCheckedChange={() => handleTogglePlugin(plugin)}
                                />
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* Plugin Details Drawer */}
            <Sheet open={isDrawerOpen} onOpenChange={setIsDrawerOpen}>
                {selectedPlugin && (
                    <>
                        <SheetHeader>
                            <SheetTitle>{selectedPlugin.name}</SheetTitle>
                            <SheetDescription>Plugin Details</SheetDescription>
                        </SheetHeader>
                        <SheetContent>
                            <div className="space-y-4">
                                <div>
                                    <h4 className="font-semibold text-sm mb-2">Description</h4>
                                    <p className="text-sm text-gray-600">{selectedPlugin.description}</p>
                                </div>
                                <div>
                                    <h4 className="font-semibold text-sm mb-2">Information</h4>
                                    <dl className="space-y-2 text-sm">
                                        <div className="flex justify-between">
                                            <dt className="text-gray-600">Package ID:</dt>
                                            <dd className="font-mono text-xs">{selectedPlugin.packageId}</dd>
                                        </div>
                                        <div className="flex justify-between">
                                            <dt className="text-gray-600">Version:</dt>
                                            <dd>{selectedPlugin.version}</dd>
                                        </div>
                                        <div className="flex justify-between">
                                            <dt className="text-gray-600">Author:</dt>
                                            <dd>{selectedPlugin.author}</dd>
                                        </div>
                                        <div className="flex justify-between">
                                            <dt className="text-gray-600">Status:</dt>
                                            <dd>{selectedPlugin.enabled ? 'Enabled' : 'Disabled'}</dd>
                                        </div>
                                        <div className="flex justify-between">
                                            <dt className="text-gray-600">Used Count:</dt>
                                            <dd>{selectedPlugin.usedCount}</dd>
                                        </div>
                                    </dl>
                                </div>
                            </div>
                        </SheetContent>
                        <SheetFooter>
                            <Button variant="outline" onClick={() => setIsDrawerOpen(false)}>
                                Close
                            </Button>
                            <Button
                                variant="destructive"
                                onClick={() => handleUninstallPlugin(selectedPlugin)}
                            >
                                Uninstall
                            </Button>
                        </SheetFooter>
                    </>
                )}
            </Sheet>
        </div>
    )
}
