// This is the only application module coupled to generated binding paths.
import * as App from '../../../bindings/watools/internal/host/appservice'
import * as Command from '../../../bindings/watools/internal/host/commandservice'
import * as Plugin from '../../../bindings/watools/internal/host/pluginservice'
import * as Storage from '../../../bindings/watools/internal/host/storageservice'
import * as System from '../../../bindings/watools/internal/host/systemservice'
import * as Update from '../../../bindings/watools/internal/host/updateservice'
import * as Log from '../../../bindings/watools/internal/host/logservice'
import type {HostAPI, PluginWindowSession} from './contracts'
import {HostError, withHostErrors} from './errors'

const stringMap = (values: {[key: string]: string | undefined}): Record<string, string> =>
    Object.fromEntries(Object.entries(values).filter((pair): pair is [string, string] => typeof pair[1] === 'string'))

export const wailsHost: HostAPI = withHostErrors({
    ...App, ...Command, ...Plugin, ...Storage, ...System, ...Update, ...Log,
    DetachPluginApi: request => Plugin.DetachPluginApi({...request,context:{...request.context,clipboard:request.context.clipboard ? {...request.context.clipboard,files:request.context.clipboard.files ?? []} : null}}),
    GetPluginWindowApi: async id => await Plugin.GetPluginWindowApi(id) as unknown as PluginWindowSession,
    GetApplicationCommandsApi: async () => (await Command.GetApplicationCommandsApi()).map(c => ({...c, category: 'Application' as const})),
    GetOperatorCommandsApi: async () => (await Command.GetOperatorCommandsApi()).map(c => ({...c, category: 'Operation' as const})),
    HttpProxyApi: async request => {
        const response = await System.HttpProxyApi({...request, method: request.method ?? ''})
        if (!response) throw new HostError('OPERATION_FAILED', '宿主返回了空 HTTP 响应', 'system.http')
        return {...response, headers: stringMap(response.headers)}
    },
    QueryLogsApi: async query => {
        const response = await Log.QueryLogsApi({files: [], keyword: '', levels: [], sources: [], pluginIds: [], cursor: '', limit: 0, ...query})
        return {...response, records: response.records.map(r => ({...r, fields: r.fields ? stringMap(r.fields) : undefined, parseStatus: r.parseStatus === 'structured' ? 'structured' as const : 'raw' as const}))}
    }
} satisfies HostAPI)
