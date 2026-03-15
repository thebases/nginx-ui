import { http } from '@uozi-admin/request'

const pageSizeMax = 500
const adminProxyPrefix = '/apisix/admin'
const headerOverrideBaseURL = 'X-APISIX-BASE-URL'
const headerOverrideAPIKey = 'X-API-KEY'

const apiRoutes = '/apisix/admin/routes'
const apiStreamRoutes = '/apisix/admin/stream_routes'
const apiUpstreams = '/apisix/admin/upstreams'
const apiProtos = '/apisix/admin/protos'
const apiServices = '/apisix/admin/services'
const apiGlobalRules = '/apisix/admin/global_rules'
const apiPlugins = '/apisix/admin/plugins'
const apiPluginsList = '/apisix/admin/plugins/list'
const apiPluginMetadata = '/apisix/admin/plugin_metadata'
const apiSecrets = '/apisix/admin/secrets'
const apiConsumers = '/apisix/admin/consumers'
const apiConsumerGroups = '/apisix/admin/consumer_groups'
const apiSSLs = '/apisix/admin/ssls'
const apiPluginConfigs = '/apisix/admin/plugin_configs'

type HTTPMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface APISIXProxyConfig {
  base_url?: string
  api_key?: string
}

export interface PageSearchType {
  page?: number
  page_size?: number
  [key: string]: QueryValue
}

export interface WithID {
  id: string
  [key: string]: unknown
}

export interface WithUsername {
  username: string
  [key: string]: unknown
}

export interface WithSecretManagerAndID {
  manager: string
  id: string
  [key: string]: unknown
}

export interface PluginMetadataPut {
  name: string
  config: Record<string, unknown>
}

interface APISIXListItem {
  value: Record<string, unknown>
}

interface APISIXListResponse {
  total: number
  list: APISIXListItem[] | Record<string, APISIXListItem>
}

type QueryValue = string | number | boolean | undefined

function normalizeListItems(
  list: APISIXListResponse['list'] | undefined,
): APISIXListItem[] {
  if (!list) {
    return []
  }
  if (Array.isArray(list)) {
    return list
  }
  if (typeof list === 'object') {
    return Object.values(list).filter((item): item is APISIXListItem => {
      return Boolean(item && typeof item === 'object')
    })
  }
  return []
}

function normalizeQuery(
  query?: Record<string, QueryValue>,
): Record<string, string> | undefined {
  if (!query) {
    return undefined
  }

  const normalized: Record<string, string> = {}
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null) {
      return
    }
    normalized[key] = String(value)
  })

  return Object.keys(normalized).length > 0 ? normalized : undefined
}

function apisixRequest<T = unknown>(
  method: HTTPMethod,
  path: string,
  config?: APISIXProxyConfig,
  options?: {
    query?: Record<string, QueryValue>
    headers?: Record<string, string>
    body?: unknown
  },
): Promise<T> {
  const headers: Record<string, string> = {
    ...(options?.headers ?? {}),
  }
  if (config?.base_url) {
    headers[headerOverrideBaseURL] = config.base_url
  }
  if (config?.api_key) {
    headers[headerOverrideAPIKey] = config.api_key
  }
  const hasHeaders = Object.keys(headers).length > 0

  const normalizedPath = path.startsWith(adminProxyPrefix)
    ? path
    : `${adminProxyPrefix}/${path.replace(/^\/+/, '')}`

  const requestConfig = {
    params: normalizeQuery(options?.query),
    ...(hasHeaders ? { headers } : {}),
  }

  switch (method) {
    case 'GET':
      return http.get<T>(normalizedPath, requestConfig)
    case 'POST':
      return http.post<T>(normalizedPath, options?.body, requestConfig)
    case 'PUT':
      return http.put<T>(normalizedPath, options?.body, requestConfig)
    case 'PATCH':
      return http.patch<T>(normalizedPath, options?.body, requestConfig)
    case 'DELETE':
      return http.delete<T>(normalizedPath, requestConfig)
    default:
      throw new Error(`Unsupported APISIX method: ${method}`)
  }
}

async function deleteAllByField(
  path: string,
  keyField: string,
  config?: APISIXProxyConfig,
) {
  while (true) {
    const listResp = await apisixRequest<APISIXListResponse>('GET', path, config, {
      query: { page: 1, page_size: pageSizeMax },
    })
    const list = normalizeListItems(listResp.list)

    if (list.length === 0) {
      return
    }

    const ids = list
      .map(item => String(item.value?.[keyField] ?? ''))
      .filter(Boolean)

    if (ids.length === 0) {
      return
    }

    await Promise.all(
      ids.map(id => apisixRequest('DELETE', `${path}/${encodeURIComponent(id)}`, config)),
    )
  }
}

export const apisixApi = {
  request: apisixRequest,

  getRouteListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiRoutes, config, { query: params }),
  getRouteReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiRoutes}/${encodeURIComponent(id)}`, config),
  putRouteReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiRoutes}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  postRouteReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('POST', apiRoutes, config, { body: data }),
  deleteAllRoutes: (config?: APISIXProxyConfig) =>
    deleteAllByField(apiRoutes, 'id', config),

  getServiceListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiServices, config, { query: params }),
  getServiceReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiServices}/${encodeURIComponent(id)}`, config),
  putServiceReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiServices}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  postServiceReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('POST', apiServices, config, { body: data }),
  deleteAllServices: async (config?: APISIXProxyConfig) => {
    await apisixApi.deleteAllRoutes(config)
    await apisixApi.deleteAllStreamRoutes(config)
    await deleteAllByField(apiServices, 'id', config)
  },

  getConsumerListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiConsumers, config, { query: params }),
  getConsumerReq: (username: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiConsumers}/${encodeURIComponent(username)}`, config),
  putConsumerReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('PUT', apiConsumers, config, { body: data }),
  deleteAllConsumers: (config?: APISIXProxyConfig) =>
    deleteAllByField(apiConsumers, 'username', config),

  getConsumerGroupListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiConsumerGroups, config, { query: params }),
  getConsumerGroupReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiConsumerGroups}/${encodeURIComponent(id)}`, config),
  putConsumerGroupReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiConsumerGroups}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  deleteAllConsumerGroups: (config?: APISIXProxyConfig) =>
    deleteAllByField(apiConsumerGroups, 'id', config),

  getCredentialListReq: (
    params: WithUsername,
    config?: APISIXProxyConfig,
  ) => apisixRequest('GET', `${apiConsumers}/${encodeURIComponent(params.username)}/credentials`, config, {
    query: { username: params.username },
  }),
  getCredentialReq: (username: string, id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiConsumers}/${encodeURIComponent(username)}/credentials/${encodeURIComponent(id)}`, config),
  putCredentialReq: (data: WithUsername & WithID, config?: APISIXProxyConfig) => {
    const { username, id, ...rest } = data
    return apisixRequest('PUT', `${apiConsumers}/${encodeURIComponent(username)}/credentials/${encodeURIComponent(id)}`, config, {
      body: rest,
    })
  },

  getUpstreamListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiUpstreams, config, { query: params }),
  getUpstreamReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiUpstreams}/${encodeURIComponent(id)}`, config),
  postUpstreamReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('POST', apiUpstreams, config, { body: data }),
  putUpstreamReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiUpstreams}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  deleteAllUpstreams: (config?: APISIXProxyConfig) =>
    deleteAllByField(apiUpstreams, 'id', config),

  getStreamRouteListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiStreamRoutes, config, { query: params }),
  getStreamRouteReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiStreamRoutes}/${encodeURIComponent(id)}`, config),
  putStreamRouteReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiStreamRoutes}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  postStreamRouteReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('POST', apiStreamRoutes, config, { body: data }),
  deleteAllStreamRoutes: (config?: APISIXProxyConfig) =>
    deleteAllByField(apiStreamRoutes, 'id', config),

  getSSLListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiSSLs, config, { query: params }),
  getSSLReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiSSLs}/${encodeURIComponent(id)}`, config),
  putSSLReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiSSLs}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  postSSLReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('POST', apiSSLs, config, { body: data }),
  deleteAllSSLs: (config?: APISIXProxyConfig) =>
    deleteAllByField(apiSSLs, 'id', config),

  getGlobalRuleListReq: (config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiGlobalRules, config),
  getGlobalRuleReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiGlobalRules}/${encodeURIComponent(id)}`, config),
  putGlobalRuleReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiGlobalRules}/${encodeURIComponent(id)}`, config, { body: rest })
  },

  getPluginConfigListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiPluginConfigs, config, { query: params }),
  getPluginConfigReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiPluginConfigs}/${encodeURIComponent(id)}`, config),
  putPluginConfigReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiPluginConfigs}/${encodeURIComponent(id)}`, config, { body: rest })
  },

  getProtoListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiProtos, config, { query: params }),
  getProtoReq: (id: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiProtos}/${encodeURIComponent(id)}`, config),
  putProtoReq: (data: WithID, config?: APISIXProxyConfig) => {
    const { id, ...rest } = data
    return apisixRequest('PUT', `${apiProtos}/${encodeURIComponent(id)}`, config, { body: rest })
  },
  postProtoReq: (data: Record<string, unknown>, config?: APISIXProxyConfig) =>
    apisixRequest('POST', apiProtos, config, { body: data }),

  getSecretListReq: (params: PageSearchType, config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiSecrets, config, { query: params }),
  getSecretReq: (props: WithSecretManagerAndID, config?: APISIXProxyConfig) =>
    apisixRequest(
      'GET',
      `${apiSecrets}/${encodeURIComponent(props.manager)}/${encodeURIComponent(props.id)}`,
      config,
    ),
  putSecretReq: (data: WithSecretManagerAndID, config?: APISIXProxyConfig) => {
    const { manager, id, ...rest } = data
    return apisixRequest(
      'PUT',
      `${apiSecrets}/${encodeURIComponent(manager)}/${encodeURIComponent(id)}`,
      config,
      { body: rest },
    )
  },

  getPluginsListReq: (config?: APISIXProxyConfig) =>
    apisixRequest('GET', apiPluginsList, config),
  getPluginsListWithSchemaReq: (
    props?: { subsystem?: string },
    config?: APISIXProxyConfig,
  ) =>
    apisixRequest('GET', apiPlugins, config, {
      query: { all: true, subsystem: props?.subsystem },
    }),
  getPluginSchemaReq: (name: string, config?: APISIXProxyConfig) =>
    apisixRequest('GET', `${apiPlugins}/${encodeURIComponent(name)}`, config),
  putPluginMetadataReq: (props: PluginMetadataPut, config?: APISIXProxyConfig) =>
    apisixRequest('PUT', `${apiPluginMetadata}/${encodeURIComponent(props.name)}`, config, {
      body: props.config,
    }),
  deletePluginMetadataReq: (name: string, config?: APISIXProxyConfig) =>
    apisixRequest('DELETE', `${apiPluginMetadata}/${encodeURIComponent(name)}`, config),
  getPluginMetadataReq: (
    pluginName: string,
    headers?: Record<string, string>,
    config?: APISIXProxyConfig,
  ) =>
    apisixRequest('GET', `${apiPluginMetadata}/${encodeURIComponent(pluginName)}`, config, {
      headers,
    }),
}

export default apisixApi
