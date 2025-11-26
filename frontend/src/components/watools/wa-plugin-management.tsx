import React, { useState, useEffect } from 'react'
import { Plugin } from '@/schemas/plugin'
import { getPlugins, togglePlugin, uninstallPlugin, installPlugin } from '@/api/plugin'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Sheet, SheetHeader, SheetTitle, SheetDescription, SheetContent, SheetFooter } from '@/components/ui/sheet'

export function WaPluginManagement() {
    const [plugins, setPlugins] = useState<Plugin[]>([])
    const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null)
    const [isDrawerOpen, setIsDrawerOpen] = useState(false)
    const [isLoading, setIsLoading] = useState(false)

    const loadPlugins = async () => {
        setIsLoading(true)
        try {
            const pluginList = await getPlugins()
            setPlugins(pluginList)
        } catch (error) {
            console.error('Failed to load plugins:', error)
        } finally {
            setIsLoading(false)
        }
    }

    useEffect(() => {
        loadPlugins()
    }, [])

    const handleTogglePlugin = async (plugin: Plugin) => {
        try {
            await togglePlugin(plugin.packageId, !plugin.enabled)
            // Update local state
            setPlugins(prev => prev.map(p =>
                p.packageId === plugin.packageId ? { ...p, enabled: !p.enabled } : p
            ))
        } catch (error) {
            console.error('Failed to toggle plugin:', error)
            alert(`Failed to toggle plugin: ${error}`)
        }
    }

    const handleUninstallPlugin = async (plugin: Plugin) => {
        if (!confirm(`Are you sure you want to uninstall ${plugin.name}?`)) {
            return
        }

        try {
            await uninstallPlugin(plugin.packageId)
            setPlugins(prev => prev.filter(p => p.packageId !== plugin.packageId))
            setIsDrawerOpen(false)
            setSelectedPlugin(null)
        } catch (error) {
            console.error('Failed to uninstall plugin:', error)
            alert(`Failed to uninstall plugin: ${error}`)
        }
    }

    const handleInstallPlugin = async () => {
        // TODO: Implement file picker or drag & drop for .wt files
        const filePath = prompt('Enter the path to the .wt plugin file:')
        if (!filePath) return

        try {
            await installPlugin(filePath)
            await loadPlugins() // Reload plugins
            alert('Plugin installed successfully!')
        } catch (error) {
            console.error('Failed to install plugin:', error)
            alert(`Failed to install plugin: ${error}`)
        }
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
