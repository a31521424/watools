import React, { useEffect } from 'react'
import { usePluginStore } from '@/stores/pluginStore'

export const PluginExample: React.FC = () => {
  const {
    plugins,
    isLoading,
    error,
    fetchPlugins,
    fetchPluginsAsync,  // fire-and-forget version
    getActivePlugins,
    getPluginsByType
  } = usePluginStore()

  useEffect(() => {
    // 方案 1: 明确忽略 Promise（推荐）
    void fetchPlugins()

    // 方案 2: 使用 fire-and-forget 方法
    // fetchPluginsAsync()

    // 方案 3: 添加错误处理
    // fetchPlugins().catch(console.error)
  }, [fetchPlugins, fetchPluginsAsync])

  // 处理用户操作时的刷新（不等待结果）
  const handleRefresh = () => {
    void fetchPlugins()  // 明确表示忽略返回值
  }

  const handleRefreshWithErrorHandling = () => {
    fetchPlugins().catch(error => {
      console.error('Plugin refresh failed:', error)
      // 可以添加用户友好的错误提示
    })
  }

  if (isLoading) return <div>Loading plugins...</div>
  if (error) return <div>Error: {error}</div>

  const activePlugins = getActivePlugins()
  const executablePlugins = getPluginsByType('executable')
  const uiPlugins = getPluginsByType('ui')

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-bold">Plugins ({plugins.length})</h2>
        <div className="space-x-2">
          <button
            onClick={handleRefresh}
            className="px-3 py-1 bg-blue-500 text-white rounded hover:bg-blue-600"
          >
            Refresh (void)
          </button>
          <button
            onClick={handleRefreshWithErrorHandling}
            className="px-3 py-1 bg-green-500 text-white rounded hover:bg-green-600"
          >
            Refresh (with catch)
          </button>
          <button
            onClick={() => fetchPluginsAsync()}
            className="px-3 py-1 bg-purple-500 text-white rounded hover:bg-purple-600"
          >
            Fire & Forget
          </button>
        </div>
      </div>

      <div className="mb-4">
        <h3 className="text-lg font-semibold">Active Plugins ({activePlugins.length})</h3>
        {activePlugins.map(plugin => (
          <div key={plugin.packageId} className="border p-2 mb-2">
            <div className="font-medium">{plugin.name}</div>
            <div className="text-sm text-gray-600">{plugin.description}</div>
            <div className="text-xs text-gray-500">v{plugin.version} by {plugin.author}</div>
          </div>
        ))}
      </div>

      <div className="mb-4">
        <h3 className="text-lg font-semibold">Executable Plugins ({executablePlugins.length})</h3>
        {executablePlugins.map(plugin => (
          <div key={plugin.packageId} className="border p-2 mb-2">
            <div className="font-medium">{plugin.name}</div>
            <div className="text-sm text-gray-600">{plugin.packageId}</div>
          </div>
        ))}
      </div>

      <div className="mb-4">
        <h3 className="text-lg font-semibold">UI Plugins ({uiPlugins.length})</h3>
        {uiPlugins.map(plugin => (
          <div key={plugin.packageId} className="border p-2 mb-2">
            <div className="font-medium">{plugin.name}</div>
            <div className="text-sm text-gray-600">{plugin.packageId}</div>
          </div>
        ))}
      </div>
    </div>
  )
}