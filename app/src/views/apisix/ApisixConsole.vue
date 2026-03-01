<script setup lang="ts">
import type { TableColumnsType } from 'ant-design-vue'
import { message, Modal } from 'ant-design-vue'
import apisixApi from '@/api/apisix'

interface APISIXProxyConfig {
  base_url?: string
  api_key?: string
}

interface APISIXListItem {
  key?: string
  value: Record<string, unknown>
}

interface APISIXListResponse {
  total?: number
  list?: APISIXListItem[] | Record<string, APISIXListItem>
}

interface ResourceConfig {
  key: string
  label: string
  listPath: string
  deletePath: (item: APISIXListItem) => string
  putPath: (payload: Record<string, unknown>) => string
  postPath?: string
  createTemplate: Record<string, unknown>
}

const resources: ResourceConfig[] = [
  {
    key: 'routes',
    label: 'Routes',
    listPath: '/apisix/admin/routes',
    deletePath: item => `/apisix/admin/routes/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/routes/${encodeURIComponent(String(payload.id ?? ''))}`,
    postPath: '/apisix/admin/routes',
    createTemplate: { uri: '/', upstream_id: '' },
  },
  {
    key: 'services',
    label: 'Services',
    listPath: '/apisix/admin/services',
    deletePath: item => `/apisix/admin/services/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/services/${encodeURIComponent(String(payload.id ?? ''))}`,
    postPath: '/apisix/admin/services',
    createTemplate: { upstream_id: '' },
  },
  {
    key: 'upstreams',
    label: 'Upstreams',
    listPath: '/apisix/admin/upstreams',
    deletePath: item => `/apisix/admin/upstreams/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/upstreams/${encodeURIComponent(String(payload.id ?? ''))}`,
    postPath: '/apisix/admin/upstreams',
    createTemplate: { type: 'roundrobin', nodes: { '127.0.0.1:80': 1 } },
  },
  {
    key: 'stream_routes',
    label: 'Stream Routes',
    listPath: '/apisix/admin/stream_routes',
    deletePath: item => `/apisix/admin/stream_routes/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/stream_routes/${encodeURIComponent(String(payload.id ?? ''))}`,
    postPath: '/apisix/admin/stream_routes',
    createTemplate: { upstream_id: '' },
  },
  {
    key: 'ssls',
    label: 'SSLs',
    listPath: '/apisix/admin/ssls',
    deletePath: item => `/apisix/admin/ssls/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/ssls/${encodeURIComponent(String(payload.id ?? ''))}`,
    postPath: '/apisix/admin/ssls',
    createTemplate: { cert: '', key: '', snis: [] },
  },
  {
    key: 'consumers',
    label: 'Consumers',
    listPath: '/apisix/admin/consumers',
    deletePath: item => `/apisix/admin/consumers/${encodeURIComponent(String(item.value.username ?? ''))}`,
    putPath: () => '/apisix/admin/consumers',
    createTemplate: { username: '' },
  },
  {
    key: 'consumer_groups',
    label: 'Consumer Groups',
    listPath: '/apisix/admin/consumer_groups',
    deletePath: item => `/apisix/admin/consumer_groups/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/consumer_groups/${encodeURIComponent(String(payload.id ?? ''))}`,
    createTemplate: { id: '', plugins: {} },
  },
  {
    key: 'plugin_configs',
    label: 'Plugin Configs',
    listPath: '/apisix/admin/plugin_configs',
    deletePath: item => `/apisix/admin/plugin_configs/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/plugin_configs/${encodeURIComponent(String(payload.id ?? ''))}`,
    createTemplate: { id: '', plugins: {} },
  },
  {
    key: 'global_rules',
    label: 'Global Rules',
    listPath: '/apisix/admin/global_rules',
    deletePath: item => `/apisix/admin/global_rules/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/global_rules/${encodeURIComponent(String(payload.id ?? ''))}`,
    createTemplate: { id: '', plugins: {} },
  },
  {
    key: 'protos',
    label: 'Protos',
    listPath: '/apisix/admin/protos',
    deletePath: item => `/apisix/admin/protos/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload => `/apisix/admin/protos/${encodeURIComponent(String(payload.id ?? ''))}`,
    postPath: '/apisix/admin/protos',
    createTemplate: { id: '', content: '' },
  },
  {
    key: 'secrets',
    label: 'Secrets',
    listPath: '/apisix/admin/secrets',
    deletePath: item =>
      `/apisix/admin/secrets/${encodeURIComponent(String(item.value.manager ?? ''))}/${encodeURIComponent(String(item.value.id ?? ''))}`,
    putPath: payload =>
      `/apisix/admin/secrets/${encodeURIComponent(String(payload.manager ?? ''))}/${encodeURIComponent(String(payload.id ?? ''))}`,
    createTemplate: { manager: 'vault', id: '', uri: '' },
  },
]

const columns: TableColumnsType = [
  {
    title: () => $gettext('Key'),
    dataIndex: 'resource_key',
    key: 'resource_key',
    width: 240,
  },
  {
    title: () => $gettext('Summary'),
    dataIndex: 'summary',
    key: 'summary',
    ellipsis: true,
  },
  {
    title: () => $gettext('Actions'),
    key: 'actions',
    width: 220,
  },
]

const activeResource = ref(resources[0].key)
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const rows = ref<APISIXListItem[]>([])
const editorVisible = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editorContent = ref('')

const currentResource = computed(() => {
  return resources.find(item => item.key === activeResource.value) ?? resources[0]
})

const proxyConfig = computed<APISIXProxyConfig>(() => {
  return {}
})

const dataSource = computed(() => {
  return rows.value.map((item, index) => {
    const keyValue = extractPrimaryKey(item)
    const summary = summarize(item.value)
    return {
      ...item,
      _rowKey: `${keyValue}-${index}`,
      resource_key: keyValue,
      summary,
    }
  })
})

watch(activeResource, async () => {
  page.value = 1
  await loadList()
})

function extractPrimaryKey(item: APISIXListItem) {
  if (item.value.manager && item.value.id) {
    return `${String(item.value.manager)}/${String(item.value.id)}`
  }

  if (item.value.username) {
    return String(item.value.username)
  }

  if (item.value.id) {
    return String(item.value.id)
  }

  return String(item.key ?? '')
}

function summarize(value: Record<string, unknown>) {
  const picks = ['name', 'desc', 'uri', 'host', 'upstream_id', 'type']
  const summaryParts: string[] = []

  picks.forEach(key => {
    if (value[key] !== undefined && value[key] !== null && value[key] !== '') {
      summaryParts.push(`${key}: ${String(value[key])}`)
    }
  })

  if (summaryParts.length > 0) {
    return summaryParts.join(' | ')
  }

  return JSON.stringify(value).slice(0, 120)
}

function normalizeListItems(
  list: APISIXListResponse['list'],
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

async function loadList() {
  loading.value = true
  try {
    const data = await apisixApi.request<APISIXListResponse>('GET', currentResource.value.listPath, proxyConfig.value, {
      query: {
        page: page.value,
        page_size: pageSize.value,
      },
    })
    rows.value = normalizeListItems(data.list)
    total.value = data.total ?? rows.value.length
  }
  catch (error) {
    message.error((error as Error).message || $gettext('Failed to load APISIX resource'))
  }
  finally {
    loading.value = false
  }
}

function openCreate() {
  editorMode.value = 'create'
  editorContent.value = JSON.stringify(currentResource.value.createTemplate, null, 2)
  editorVisible.value = true
}

function openEdit(item: APISIXListItem) {
  editorMode.value = 'edit'
  editorContent.value = JSON.stringify(item.value, null, 2)
  editorVisible.value = true
}

async function saveEditor() {
  let payload: Record<string, unknown>
  try {
    payload = JSON.parse(editorContent.value)
  }
  catch {
    message.error($gettext('JSON format is invalid'))
    return
  }

  try {
    const usePost = editorMode.value === 'create' && Boolean(currentResource.value.postPath)
    const method = usePost ? 'POST' : 'PUT'
    const path = usePost
      ? String(currentResource.value.postPath)
      : currentResource.value.putPath(payload)

    await apisixApi.request(method, path, proxyConfig.value, {
      body: payload,
    })
    message.success($gettext('Saved successfully'))
    editorVisible.value = false
    await loadList()
  }
  catch (error) {
    message.error((error as Error).message || $gettext('Save failed'))
  }
}

function showRaw(item: APISIXListItem) {
  Modal.info({
    title: $gettext('Raw JSON'),
    width: 900,
    content: h('pre', {
      style: {
        maxHeight: '420px',
        overflow: 'auto',
        whiteSpace: 'pre-wrap',
      },
    }, JSON.stringify(item.value, null, 2)),
  })
}

function handleView(record: Record<string, unknown>) {
  showRaw(record as unknown as APISIXListItem)
}

function handleEdit(record: Record<string, unknown>) {
  openEdit(record as unknown as APISIXListItem)
}

function handleDelete(record: Record<string, unknown>) {
  removeItem(record as unknown as APISIXListItem)
}

async function removeItem(item: APISIXListItem) {
  Modal.confirm({
    title: $gettext('Delete this item?'),
    onOk: async () => {
      try {
        await apisixApi.request('DELETE', currentResource.value.deletePath(item), proxyConfig.value)
        message.success($gettext('Deleted'))
        await loadList()
      }
      catch (error) {
        message.error((error as Error).message || $gettext('Delete failed'))
      }
    },
  })
}

function changePage(nextPage: number, nextPageSize: number) {
  page.value = nextPage
  pageSize.value = nextPageSize
  loadList()
}

onMounted(async () => {
  await loadList()
})
</script>

<template>
  <div class="apisix-console">
    <ACard :title="$gettext('APISIX Console')">
      <ASpace class="mb-4" wrap>
        <AButton :loading="loading" @click="loadList">
          {{ $gettext('Refresh') }}
        </AButton>
      </ASpace>

      <ATabs v-model:active-key="activeResource">
        <ATabPane
          v-for="item in resources"
          :key="item.key"
          :tab="item.label"
        />
      </ATabs>

      <div class="mb-3">
        <ASpace>
          <AButton type="primary" @click="openCreate">
            {{ $gettext('New') }}
          </AButton>
        </ASpace>
      </div>

      <ATable
        :loading="loading"
        :data-source="dataSource"
        :columns="columns"
        :pagination="{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t: number) => `${$gettext('Total')}: ${t}`,
          onChange: changePage,
          onShowSizeChange: changePage,
        }"
        :row-key="(record: Record<string, string>) => record._rowKey"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'actions'">
            <ASpace>
              <AButton type="link" size="small" @click="handleView(record)">
                {{ $gettext('View') }}
              </AButton>
              <AButton type="link" size="small" @click="handleEdit(record)">
                {{ $gettext('Edit') }}
              </AButton>
              <AButton type="link" danger size="small" @click="handleDelete(record)">
                {{ $gettext('Delete') }}
              </AButton>
            </ASpace>
          </template>
        </template>
      </ATable>
    </ACard>

    <AModal
      v-model:open="editorVisible"
      :title="editorMode === 'create' ? $gettext('Create Resource') : $gettext('Edit Resource')"
      width="900px"
      @ok="saveEditor"
    >
      <ATextarea
        v-model:value="editorContent"
        :rows="22"
        class="font-mono"
      />
    </AModal>
  </div>
</template>

<style scoped>
.apisix-console {
  padding: 4px;
}
</style>
